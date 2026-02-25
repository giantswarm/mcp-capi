package harness

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/giantswarm/k8senv"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	controlplanev1 "sigs.k8s.io/cluster-api/api/controlplane/kubeadm/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"                      //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
	capierrors "sigs.k8s.io/cluster-api/errors"                               //nolint:staticcheck // TODO: migrate when CAPI v1beta2 provides a replacement type
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kubeconfigContextName = "k8senv"
	acquireTimeout        = 5 * time.Minute
)

// testUIDNamespace is a fixed UUID v4 used as the namespace for generating
// deterministic v5 UIDs in test owner references.
var testUIDNamespace = uuid.MustParse("a01b2c3d-e5f6-4890-abcd-ef1234567890")

// deterministicUID generates a deterministic UUID v5 from a name string.
func deterministicUID(name string) types.UID {
	return types.UID(uuid.NewSHA1(testUIDNamespace, []byte(name)).String())
}

var (
	// mgr is the package-level k8senv manager singleton.
	// mgr and mgrErr are written once inside mgrOnce.Do() and are read-only thereafter.
	// The sync.Once provides the happens-before guarantee for all subsequent reads.
	mgr     k8senv.Manager
	mgrOnce sync.Once
	mgrErr  error
)

// InitManager initializes the k8senv manager with CAPI CRDs.
// Must be called once from TestMain before any tests run.
// It is safe to call from multiple goroutines; only the first call performs
// initialization. Subsequent calls return the cached result.
//
// On failure, callers should print the error and exit:
//
//	if err := harness.InitManager(); err != nil {
//	    fmt.Fprintf(os.Stderr, "failed to initialize test harness: %v\n", err)
//	    os.Exit(1)
//	}
func InitManager() error {
	mgrOnce.Do(func() {
		// Silence k8senv logs during tests
		k8senv.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))

		crdPath, err := getCRDPath()
		if err != nil {
			mgrErr = fmt.Errorf("getting CRD path: %w", err)
			return
		}

		// Pool size matches GOMAXPROCS, which is also the default for
		// go test -parallel. Override via HARNESS_POOL_SIZE env var.
		poolSize := runtime.GOMAXPROCS(0)
		if raw := os.Getenv("HARNESS_POOL_SIZE"); raw != "" {
			v, err := strconv.Atoi(raw)
			if err != nil {
				mgrErr = fmt.Errorf("invalid HARNESS_POOL_SIZE %q: %w", raw, err)
				return
			}
			if v <= 0 {
				mgrErr = fmt.Errorf("invalid HARNESS_POOL_SIZE %q: must be a positive integer", raw)
				return
			}
			poolSize = v
		}

		mgr = k8senv.NewManager(
			k8senv.WithCRDDir(crdPath),
			k8senv.WithPoolSize(poolSize),
			k8senv.WithReleaseStrategy(k8senv.ReleasePurge),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := mgr.Initialize(ctx); err != nil {
			mgrErr = fmt.Errorf("initializing k8senv: %w", err)
			return
		}
	})
	return mgrErr
}

// ShutdownManager stops the k8senv manager and releases all resources.
// Should be called from TestMain after all tests complete.
//
// Safety: reading mgr without synchronization is safe here because mgr is
// read-only after mgrOnce.Do() completes, and this runs after all tests finish.
func ShutdownManager() {
	if mgr != nil {
		if err := mgr.Shutdown(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to shutdown k8senv: %v\n", err)
		}
	}
}

// testEnv wraps a k8senv instance (a local Kubernetes API server) for integration testing.
// It provides methods to create test resources (namespaces, clusters) and manages
// the lifecycle of the test environment.
type testEnv struct {
	t              TestingT
	inst           k8senv.Instance
	k8sClient      kubernetes.Interface
	ctrlClient     client.Client
	kubeconfigPath string
}

// newTestEnv acquires a k8senv instance and sets up clients for integration testing.
//
// Safety: reading mgr is safe here because callers must call InitManager first,
// so mgrOnce.Do() has completed and mgr is read-only.
func newTestEnv(t TestingT) *testEnv {
	t.Helper()

	if mgr == nil {
		t.Fatal("harness.InitManager() must be called from TestMain before running tests")
	}

	// Acquire an instance from the pool
	ctx, cancel := context.WithTimeout(context.Background(), acquireTimeout)
	defer cancel()

	inst, err := mgr.Acquire(ctx)
	if err != nil {
		t.Fatalf("failed to acquire k8senv instance: %v", err)
	}

	// Cleanup on failure - t.Fatalf() triggers deferred functions via runtime.Goexit()
	success := false
	defer func() {
		if !success {
			if releaseErr := inst.Release(); releaseErr != nil {
				t.Logf("failed to release instance during cleanup: %v", releaseErr)
			}
		}
	}()

	// Get rest.Config from the instance
	cfg, err := inst.Config()
	if err != nil {
		t.Fatalf("failed to get k8senv config: %v", err)
	}

	// Create a dedicated scheme to avoid mutating the global scheme
	// This ensures thread-safety when tests run in parallel
	s := k8sruntime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatalf("failed to add client-go scheme: %v", err)
	}
	if err := clusterv1.AddToScheme(s); err != nil {
		t.Fatalf("failed to add CAPI scheme: %v", err)
	}
	if err := controlplanev1.AddToScheme(s); err != nil {
		t.Fatalf("failed to add KubeadmControlPlane scheme: %v", err)
	}

	// Create Kubernetes clientset
	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatalf("failed to create k8s client: %v", err)
	}

	// Create controller-runtime client with dedicated scheme
	ctrlClient, err := client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		t.Fatalf("failed to create controller-runtime client: %v", err)
	}

	// Write kubeconfig to temp file managed by t.TempDir()
	kubeconfigPath := filepath.Join(t.TempDir(), "k8senv.kubeconfig")
	if err := writeKubeconfig(cfg, kubeconfigPath); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	success = true
	return &testEnv{
		t:              t,
		inst:           inst,
		k8sClient:      k8sClient,
		ctrlClient:     ctrlClient,
		kubeconfigPath: kubeconfigPath,
	}
}

// getCRDPath returns the absolute path to the crds directory.
// This function assumes the following project structure:
//   - This file: test/harness/testenv.go
//   - CRDs directory: crds/
//
// If this file is moved, the relative path calculation must be updated.
func getCRDPath() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("failed to get current file path from runtime.Caller")
	}
	crdPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "crds")
	if _, err := os.Stat(crdPath); os.IsNotExist(err) {
		return "", fmt.Errorf("CRD directory does not exist: %s (run 'make download-crds' to fetch CRDs)", crdPath)
	}
	return crdPath, nil
}

// teardown releases the k8senv instance.
// Uses clean=true to fully stop processes and guarantee isolation between tests.
// Note: kubeconfig cleanup is handled automatically by t.TempDir().
func (te *testEnv) teardown() {
	te.t.Helper()
	if te.inst != nil {
		if err := te.inst.Release(); err != nil {
			te.t.Errorf("failed to release k8senv instance: %v", err)
		}
	}
}

// createNamespace creates a namespace.
func (te *testEnv) createNamespace(ctx context.Context, name string) {
	te.t.Helper()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	if err := te.ctrlClient.Create(ctx, ns); err != nil {
		te.t.Fatalf("failed to create namespace %s: %v", name, err)
	}
}

// createSecret creates a Kubernetes Secret resource.
func (te *testEnv) createSecret(ctx context.Context, namespace, name string, data map[string][]byte) {
	te.t.Helper()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}

	if err := te.ctrlClient.Create(ctx, secret); err != nil {
		te.t.Fatalf("failed to create secret %s/%s: %v", namespace, name, err)
	}
}

// createCluster creates a basic CAPI Cluster resource.
func (te *testEnv) createCluster(ctx context.Context, namespace, name string) {
	te.t.Helper()
	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: name,
			},
		},
		Spec: clusterv1.ClusterSpec{
			ClusterNetwork: &clusterv1.ClusterNetwork{
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"192.168.0.0/16"},
				},
			},
		},
	}

	if err := te.ctrlClient.Create(ctx, cluster); err != nil {
		te.t.Fatalf("failed to create cluster %s/%s: %v", namespace, name, err)
	}
}

// clusterCreateOptions holds all parameters for creating a fully-configured cluster
// in minimal API calls.
type clusterCreateOptions struct {
	namespace         string
	name              string
	provider          string
	version           string
	phase             string
	paused            bool
	labels            map[string]string
	annotations       map[string]string
	conditions        []clusterv1.Condition
	customInfraRef    *customRef // custom InfrastructureRef (overrides provider)
	controlPlaneReady *bool      // explicit ControlPlaneReady status
	infraReady        *bool      // explicit InfrastructureReady status
	hasStatus         bool       // explicit flag to trigger status update even with zero values
}

// needsStatusUpdate reports whether any status fields are set that require
// a Status().Update() call after cluster creation.
func (o *clusterCreateOptions) needsStatusUpdate() bool {
	return o.hasStatus || o.phase != "" || len(o.conditions) > 0 || o.controlPlaneReady != nil || o.infraReady != nil
}

// providerInfraRef returns the APIVersion and Kind for a provider's infrastructure reference.
// Returns empty strings for unknown providers (no InfrastructureRef is set).
func providerInfraRef(provider string) (apiVersion, kind string) {
	switch provider {
	case "aws":
		return "infrastructure.cluster.x-k8s.io/v1beta2", "AWSCluster"
	case "azure":
		return "infrastructure.cluster.x-k8s.io/v1beta1", "AzureCluster"
	case "gcp":
		return "infrastructure.cluster.x-k8s.io/v1beta1", "GCPCluster"
	case "vsphere":
		return "infrastructure.cluster.x-k8s.io/v1beta1", "VSphereCluster"
	case "vcd":
		return "infrastructure.cluster.x-k8s.io/v1beta1", "VCDCluster"
	default:
		return "", ""
	}
}

// createClusterFull creates a fully-configured CAPI Cluster in minimal API calls.
// It sets InfrastructureRef and Topology on the initial Create (avoiding Get+Update),
// and combines phase + conditions into a single Status().Update() when both are present.
func (te *testEnv) createClusterFull(ctx context.Context, opts clusterCreateOptions) {
	te.t.Helper()

	// Build labels: always include ClusterNameLabel, then merge caller-supplied labels
	clusterLabels := map[string]string{
		clusterv1.ClusterNameLabel: opts.name,
	}
	for k, v := range opts.labels {
		clusterLabels[k] = v
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:        opts.name,
			Namespace:   opts.namespace,
			Labels:      clusterLabels,
			Annotations: opts.annotations,
		},
		Spec: clusterv1.ClusterSpec{
			Paused: opts.paused,
			ClusterNetwork: &clusterv1.ClusterNetwork{
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"192.168.0.0/16"},
				},
			},
		},
	}

	// Set infrastructure ref before Create
	if opts.customInfraRef != nil {
		// Custom InfrastructureRef overrides provider
		cluster.Spec.InfrastructureRef = &corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       opts.customInfraRef.kind,
			Name:       opts.customInfraRef.name,
			Namespace:  opts.namespace,
		}
	} else if opts.provider != "" {
		if apiVersion, kind := providerInfraRef(opts.provider); kind != "" {
			cluster.Spec.InfrastructureRef = &corev1.ObjectReference{
				APIVersion: apiVersion,
				Kind:       kind,
				Name:       opts.name,
				Namespace:  opts.namespace,
			}
		}
	}

	// Set version via Topology before Create (avoids Get+Update round-trip)
	if opts.version != "" {
		cluster.Spec.Topology = &clusterv1.Topology{
			Class:   "default",
			Version: opts.version,
		}
	}

	if err := te.ctrlClient.Create(ctx, cluster); err != nil {
		te.t.Fatalf("failed to create cluster %s/%s: %v", opts.namespace, opts.name, err)
	}

	// Combine phase + conditions + status booleans into a single Status().Update() when possible
	if opts.needsStatusUpdate() {
		if opts.phase != "" {
			cluster.Status.Phase = opts.phase
		}
		if len(opts.conditions) > 0 {
			cluster.Status.Conditions = opts.conditions
		}
		if opts.controlPlaneReady != nil {
			cluster.Status.ControlPlaneReady = *opts.controlPlaneReady
		}
		if opts.infraReady != nil {
			cluster.Status.InfrastructureReady = *opts.infraReady
		}
		if err := te.ctrlClient.Status().Update(ctx, cluster); err != nil {
			te.t.Fatalf("failed to update status on cluster %s/%s: %v", opts.namespace, opts.name, err)
		}
	}
}

// createMachine creates a CAPI Machine resource for the given cluster.
// If ready is true, the machine's Status.NodeRef will be set to simulate a ready machine.
func (te *testEnv) createMachine(ctx context.Context, namespace, clusterName, machineName string, ready bool) {
	te.t.Helper()
	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machineName,
			Namespace: namespace,
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: clusterName,
			},
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: clusterName,
			Bootstrap: clusterv1.Bootstrap{
				DataSecretName: ptr("bootstrap-secret"),
			},
		},
	}

	if err := te.ctrlClient.Create(ctx, machine); err != nil {
		te.t.Fatalf("failed to create machine %s/%s: %v", namespace, machineName, err)
	}

	// If ready, set NodeRef in status
	if ready {
		machine.Status.NodeRef = &corev1.ObjectReference{
			Kind: "Node",
			Name: machineName + "-node",
		}
		if err := te.ctrlClient.Status().Update(ctx, machine); err != nil {
			te.t.Fatalf("failed to set NodeRef on machine %s/%s: %v", namespace, machineName, err)
		}
	}
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}

// createKubeadmControlPlane creates a KubeadmControlPlane resource.
func (te *testEnv) createKubeadmControlPlane(ctx context.Context, namespace, name, version string, replicas int32) {
	te.t.Helper()
	kcp := &controlplanev1.KubeadmControlPlane{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: controlplanev1.KubeadmControlPlaneSpec{
			Version:  version,
			Replicas: &replicas,
			MachineTemplate: controlplanev1.KubeadmControlPlaneMachineTemplate{
				InfrastructureRef: corev1.ObjectReference{
					APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
					Kind:       "GenericInfrastructureMachineTemplate",
					Name:       name + "-machine-template",
				},
			},
		},
	}

	if err := te.ctrlClient.Create(ctx, kcp); err != nil {
		te.t.Fatalf("failed to create KubeadmControlPlane %s/%s: %v", namespace, name, err)
	}
}

// setClusterControlPlaneRef sets the ControlPlaneRef on a cluster to point to a KubeadmControlPlane.
func (te *testEnv) setClusterControlPlaneRef(ctx context.Context, namespace, clusterName, kcpName string) {
	te.t.Helper()
	te.setClusterControlPlaneRefCustom(ctx, namespace, clusterName, "KubeadmControlPlane", kcpName)
}

// setClusterControlPlaneRefCustom sets the ControlPlaneRef on a cluster to an arbitrary kind and name.
// Unlike setClusterControlPlaneRef, this does not assume KubeadmControlPlane and allows
// testing non-KubeadmControlPlane types or references to non-existent resources.
func (te *testEnv) setClusterControlPlaneRefCustom(ctx context.Context, namespace, clusterName, kind, cpName string) {
	te.t.Helper()
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{Namespace: namespace, Name: clusterName}
	if err := te.ctrlClient.Get(ctx, key, cluster); err != nil {
		te.t.Fatalf("failed to get cluster %s/%s: %v", namespace, clusterName, err)
	}
	cluster.Spec.ControlPlaneRef = &corev1.ObjectReference{
		APIVersion: "controlplane.cluster.x-k8s.io/v1beta1",
		Kind:       kind,
		Name:       cpName,
		Namespace:  namespace,
	}
	// Ensure Topology.Class is set if Topology exists (required by CAPI validation)
	if cluster.Spec.Topology != nil && cluster.Spec.Topology.Class == "" {
		cluster.Spec.Topology.Class = "default"
	}
	if err := te.ctrlClient.Update(ctx, cluster); err != nil {
		te.t.Fatalf("failed to set ControlPlaneRef on cluster %s/%s: %v", namespace, clusterName, err)
	}
}

// createMachineDeployment creates a CAPI MachineDeployment resource for the given cluster.
func (te *testEnv) createMachineDeployment(ctx context.Context, opts machineDeploymentCreateOptions) {
	te.t.Helper()

	md := &clusterv1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.name,
			Namespace: opts.namespace,
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: opts.clusterName,
			},
		},
		Spec: clusterv1.MachineDeploymentSpec{
			ClusterName: opts.clusterName,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"machinedeployment": opts.name,
				},
			},
			Template: clusterv1.MachineTemplateSpec{
				ObjectMeta: clusterv1.ObjectMeta{
					Labels: map[string]string{
						"machinedeployment":                  opts.name,
						clusterv1.ClusterNameLabel:           opts.clusterName,
						clusterv1.MachineDeploymentNameLabel: opts.name,
					},
				},
				Spec: clusterv1.MachineSpec{
					ClusterName: opts.clusterName,
					Bootstrap: clusterv1.Bootstrap{
						DataSecretName: ptr("bootstrap-secret"),
					},
				},
			},
		},
	}

	if !opts.nilReplicas {
		md.Spec.Replicas = &opts.replicas
	}

	if opts.version != "" {
		md.Spec.Template.Spec.Version = &opts.version
	}

	if opts.infraRefKind != "" {
		md.Spec.Template.Spec.InfrastructureRef = corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       opts.infraRefKind,
			Name:       opts.infraRefName,
		}
	}

	if err := te.ctrlClient.Create(ctx, md); err != nil {
		te.t.Fatalf("failed to create MachineDeployment %s/%s: %v", opts.namespace, opts.name, err)
	}

	// Update status if needed
	if opts.needsStatusUpdate() {
		if opts.phase != "" {
			md.Status.Phase = opts.phase
		}
		md.Status.Replicas = opts.statusReplicas
		md.Status.ReadyReplicas = opts.readyReplicas
		md.Status.UpdatedReplicas = opts.updatedReplicas
		md.Status.AvailableReplicas = opts.availableReplicas
		if err := te.ctrlClient.Status().Update(ctx, md); err != nil {
			te.t.Fatalf("failed to update status on MachineDeployment %s/%s: %v", opts.namespace, opts.name, err)
		}
	}
}

// machineDeploymentCreateOptions holds parameters for creating a MachineDeployment.
type machineDeploymentCreateOptions struct {
	namespace         string
	name              string
	clusterName       string
	replicas          int32
	nilReplicas       bool
	version           string
	phase             string
	infraRefKind      string
	infraRefName      string
	hasStatus         bool // explicit flag to trigger status update even with zero values
	statusReplicas    int32
	readyReplicas     int32
	updatedReplicas   int32
	availableReplicas int32
}

// needsStatusUpdate reports whether any status fields are set that require
// a Status().Update() call after MachineDeployment creation.
func (o *machineDeploymentCreateOptions) needsStatusUpdate() bool {
	return o.hasStatus || o.phase != ""
}

// createMachineSet creates a CAPI MachineSet resource for the given cluster.
func (te *testEnv) createMachineSet(ctx context.Context, opts machineSetCreateOptions) {
	te.t.Helper()

	ms := &clusterv1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.name,
			Namespace: opts.namespace,
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: opts.clusterName,
			},
		},
		Spec: clusterv1.MachineSetSpec{
			ClusterName: opts.clusterName,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"machineset": opts.name,
				},
			},
			Template: clusterv1.MachineTemplateSpec{
				ObjectMeta: clusterv1.ObjectMeta{
					Labels: map[string]string{
						"machineset":               opts.name,
						clusterv1.ClusterNameLabel: opts.clusterName,
					},
				},
				Spec: clusterv1.MachineSpec{
					ClusterName: opts.clusterName,
					Bootstrap: clusterv1.Bootstrap{
						DataSecretName: ptr("bootstrap-secret"),
					},
				},
			},
		},
	}

	if !opts.nilReplicas {
		ms.Spec.Replicas = &opts.replicas
	}

	if opts.version != "" {
		ms.Spec.Template.Spec.Version = &opts.version
	}

	if opts.infraRefKind != "" {
		ms.Spec.Template.Spec.InfrastructureRef = corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       opts.infraRefKind,
			Name:       opts.infraRefName,
		}
	}

	if opts.bootstrapKind != "" {
		ms.Spec.Template.Spec.Bootstrap.ConfigRef = &corev1.ObjectReference{
			APIVersion: "bootstrap.cluster.x-k8s.io/v1beta1",
			Kind:       opts.bootstrapKind,
			Name:       opts.bootstrapName,
		}
	}

	if opts.ownerMDName != "" {
		ms.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: "cluster.x-k8s.io/v1beta1",
				Kind:       "MachineDeployment",
				Name:       opts.ownerMDName,
				UID:        deterministicUID(opts.ownerMDName),
				Controller: ptr(true),
			},
		}
	} else if opts.ownerKind != "" {
		ms.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: "cluster.x-k8s.io/v1beta1",
				Kind:       opts.ownerKind,
				Name:       opts.ownerName,
				UID:        deterministicUID(opts.ownerName),
				Controller: ptr(true),
			},
		}
	}

	if err := te.ctrlClient.Create(ctx, ms); err != nil {
		te.t.Fatalf("failed to create MachineSet %s/%s: %v", opts.namespace, opts.name, err)
	}

	// Update status if needed
	if opts.needsStatusUpdate() {
		ms.Status.Replicas = opts.statusReplicas
		ms.Status.ReadyReplicas = opts.readyReplicas
		ms.Status.AvailableReplicas = opts.availableReplicas
		if opts.failureReason != "" {
			reason := capierrors.MachineSetStatusError(opts.failureReason)
			ms.Status.FailureReason = &reason //nolint:staticcheck // deprecated but needed for v1beta1 test coverage
		}
		if opts.failureMessage != "" {
			ms.Status.FailureMessage = &opts.failureMessage //nolint:staticcheck // deprecated but needed for v1beta1 test coverage
		}
		for _, cond := range opts.conditions {
			ms.Status.Conditions = append(ms.Status.Conditions, clusterv1.Condition{
				Type:               clusterv1.ConditionType(cond.condType),
				Status:             cond.status,
				Reason:             cond.reason,
				Message:            cond.message,
				LastTransitionTime: metav1.Now(),
			})
		}
		if err := te.ctrlClient.Status().Update(ctx, ms); err != nil {
			te.t.Fatalf("failed to update status on MachineSet %s/%s: %v", opts.namespace, opts.name, err)
		}
	}
}

// machineSetCreateOptions holds parameters for creating a MachineSet.
type machineSetCreateOptions struct {
	namespace         string
	name              string
	clusterName       string
	replicas          int32
	nilReplicas       bool
	version           string
	hasStatus         bool // explicit flag to trigger status update even with zero values
	statusReplicas    int32
	readyReplicas     int32
	availableReplicas int32
	infraRefKind      string
	infraRefName      string
	bootstrapKind     string
	bootstrapName     string
	ownerMDName       string
	ownerKind         string
	ownerName         string
	conditions        []simpleCondition
	failureReason     string
	failureMessage    string
}

// needsStatusUpdate reports whether any status fields are set that require
// a Status().Update() call after MachineSet creation.
func (o *machineSetCreateOptions) needsStatusUpdate() bool {
	return o.hasStatus || o.failureReason != "" || o.failureMessage != "" || len(o.conditions) > 0
}

// nodeCreateOptions holds all parameters for creating a fully-configured Kubernetes Node.
type nodeCreateOptions struct {
	name          string
	providerID    string
	unschedulable bool
	conditions    []simpleCondition
	addresses     []nodeAddress
	taints        []nodeTaint
	capacity      nodeResources
	allocatable   nodeResources
	nodeInfo      nodeInfoConfig
}

// createNode creates a Kubernetes Node resource with the given configuration.
func (te *testEnv) createNode(ctx context.Context, opts nodeCreateOptions) {
	te.t.Helper()

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: opts.name,
		},
		Spec: corev1.NodeSpec{
			Unschedulable: opts.unschedulable,
		},
	}

	if opts.providerID != "" {
		node.Spec.ProviderID = opts.providerID
	}

	// Set taints
	for _, taint := range opts.taints {
		node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
			Key:    taint.key,
			Value:  taint.value,
			Effect: corev1.TaintEffect(taint.effect),
		})
	}

	if err := te.ctrlClient.Create(ctx, node); err != nil {
		te.t.Fatalf("failed to create node %s: %v", opts.name, err)
	}

	// Update status (conditions, addresses, capacity, allocatable, nodeInfo)
	node.Status.NodeInfo = corev1.NodeSystemInfo{
		OperatingSystem:         opts.nodeInfo.osName,
		OSImage:                 opts.nodeInfo.osImage,
		KernelVersion:           opts.nodeInfo.kernelVersion,
		ContainerRuntimeVersion: opts.nodeInfo.containerRuntimeVersion,
		KubeletVersion:          opts.nodeInfo.kubeletVersion,
		Architecture:            opts.nodeInfo.architecture,
	}

	// Set conditions
	for _, cond := range opts.conditions {
		node.Status.Conditions = append(node.Status.Conditions, corev1.NodeCondition{
			Type:    corev1.NodeConditionType(cond.condType),
			Status:  cond.status,
			Reason:  cond.reason,
			Message: cond.message,
		})
	}

	// Set addresses
	for _, addr := range opts.addresses {
		node.Status.Addresses = append(node.Status.Addresses, corev1.NodeAddress{
			Type:    corev1.NodeAddressType(addr.addrType),
			Address: addr.address,
		})
	}

	// Set capacity and allocatable
	node.Status.Capacity = buildResourceList(opts.capacity)
	node.Status.Allocatable = buildResourceList(opts.allocatable)

	if err := te.ctrlClient.Status().Update(ctx, node); err != nil {
		te.t.Fatalf("failed to update status on node %s: %v", opts.name, err)
	}
}

// machineCustomCreateOptions holds all parameters for creating a fully-configured CAPI Machine.
type machineCustomCreateOptions struct {
	namespace     string
	name          string
	clusterName   string
	phase         string
	version       string
	providerID    string
	nodeRefName   string
	configRefKind string
	configRefName string
	infraRefKind  string
	infraRefName  string
	conditions    []simpleCondition
	addresses     []machineAddress
	hasStatus     bool // explicit flag to trigger status update even with zero values
}

// needsStatusUpdate reports whether any status fields are set that require
// a Status().Update() call after Machine creation.
func (o *machineCustomCreateOptions) needsStatusUpdate() bool {
	return o.hasStatus || o.phase != "" || o.nodeRefName != "" || len(o.conditions) > 0 || len(o.addresses) > 0
}

// createMachineCustom creates a CAPI Machine resource with fine-grained field control.
func (te *testEnv) createMachineCustom(ctx context.Context, opts machineCustomCreateOptions) {
	te.t.Helper()
	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.name,
			Namespace: opts.namespace,
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: opts.clusterName,
			},
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: opts.clusterName,
			Bootstrap: clusterv1.Bootstrap{
				DataSecretName: ptr("bootstrap-secret"),
			},
		},
	}

	if opts.version != "" {
		machine.Spec.Version = &opts.version
	}

	if opts.providerID != "" {
		machine.Spec.ProviderID = &opts.providerID
	}

	if opts.configRefKind != "" {
		machine.Spec.Bootstrap.ConfigRef = &corev1.ObjectReference{
			APIVersion: "bootstrap.cluster.x-k8s.io/v1beta1",
			Kind:       opts.configRefKind,
			Name:       opts.configRefName,
		}
	}

	if opts.infraRefKind != "" {
		machine.Spec.InfrastructureRef = corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       opts.infraRefKind,
			Name:       opts.infraRefName,
		}
	}

	if err := te.ctrlClient.Create(ctx, machine); err != nil {
		te.t.Fatalf("failed to create Machine %s/%s: %v", opts.namespace, opts.name, err)
	}

	if opts.needsStatusUpdate() {
		if opts.phase != "" {
			machine.Status.Phase = opts.phase
		}
		if opts.nodeRefName != "" {
			machine.Status.NodeRef = &corev1.ObjectReference{
				Kind: "Node",
				Name: opts.nodeRefName,
			}
		}
		for _, cond := range opts.conditions {
			machine.Status.Conditions = append(machine.Status.Conditions, clusterv1.Condition{
				Type:               clusterv1.ConditionType(cond.condType),
				Status:             cond.status,
				Reason:             cond.reason,
				Message:            cond.message,
				LastTransitionTime: metav1.Now(),
			})
		}
		for _, addr := range opts.addresses {
			machine.Status.Addresses = append(machine.Status.Addresses, clusterv1.MachineAddress{
				Type:    clusterv1.MachineAddressType(addr.addrType),
				Address: addr.address,
			})
		}
		if err := te.ctrlClient.Status().Update(ctx, machine); err != nil {
			te.t.Fatalf("failed to update status on Machine %s/%s: %v", opts.namespace, opts.name, err)
		}
	}
}

// buildResourceList converts a nodeResources struct into a corev1.ResourceList,
// skipping any fields that are empty strings.
func buildResourceList(res nodeResources) corev1.ResourceList {
	rl := corev1.ResourceList{}
	if res.cpu != "" {
		rl[corev1.ResourceCPU] = resource.MustParse(res.cpu)
	}
	if res.memory != "" {
		rl[corev1.ResourceMemory] = resource.MustParse(res.memory)
	}
	if res.pods != "" {
		rl[corev1.ResourcePods] = resource.MustParse(res.pods)
	}
	return rl
}

// writeKubeconfig converts a rest.Config to a kubeconfig file
// and writes it to the specified path.
func writeKubeconfig(config *rest.Config, outputPath string) error {
	if config == nil {
		return errors.New("rest.Config cannot be nil")
	}
	if outputPath == "" {
		return errors.New("output path cannot be empty")
	}

	// Create kubeconfig structure
	kubeconfig := api.Config{
		Clusters: map[string]*api.Cluster{
			kubeconfigContextName: {
				Server:                   config.Host,
				CertificateAuthorityData: config.CAData,
				InsecureSkipTLSVerify:    config.Insecure,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			kubeconfigContextName: {
				ClientCertificateData: config.CertData,
				ClientKeyData:         config.KeyData,
				Token:                 config.BearerToken,
			},
		},
		Contexts: map[string]*api.Context{
			kubeconfigContextName: {
				Cluster:  kubeconfigContextName,
				AuthInfo: kubeconfigContextName,
			},
		},
		CurrentContext: kubeconfigContextName,
	}

	// Write to file
	return clientcmd.WriteToFile(kubeconfig, outputPath)
}
