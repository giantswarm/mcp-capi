package harness

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/giantswarm/k8senv"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	controlplanev1 "sigs.k8s.io/cluster-api/api/controlplane/kubeadm/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors" //nolint:staticcheck // TODO: migrate when CAPI v1beta2 provides a replacement type
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kubeconfigContextName = "k8senv"
	acquireTimeout        = 5 * time.Minute
)

// mgr is the package-level k8senv manager singleton.
var mgr k8senv.Manager

// InitManager initializes the k8senv manager with CAPI CRDs.
// Must be called once from TestMain before any tests run.
func InitManager() {
	// Silence k8senv logs during tests
	k8senv.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	crdPath, err := getCRDPath()
	if err != nil {
		panic(fmt.Sprintf("failed to get CRD path: %v", err))
	}

	poolSize := runtime.GOMAXPROCS(0) // same default as testing.parallel
	if f := flag.Lookup("test.parallel"); f != nil {
		if n, err := strconv.Atoi(f.Value.String()); err == nil && n > 0 {
			poolSize = n
		}
	}

	mgr = k8senv.NewManager(
		k8senv.WithCRDDir(crdPath),
		k8senv.WithPoolSize(poolSize),
		k8senv.WithReleaseStrategy(k8senv.ReleasePurge),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := mgr.Initialize(ctx); err != nil {
		panic(fmt.Sprintf("failed to initialize k8senv: %v", err))
	}
}

// ShutdownManager stops the k8senv manager and releases all resources.
// Should be called from TestMain after all tests complete.
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

// createNamespace creates a namespace
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

// createCluster creates a basic CAPI Cluster resource
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
	conditions        []clusterv1.Condition
	customInfraRef    *customRef // custom InfrastructureRef (overrides provider)
	controlPlaneReady *bool      // explicit ControlPlaneReady status
	infraReady        *bool      // explicit InfrastructureReady status
}

// createClusterFull creates a fully-configured CAPI Cluster in minimal API calls.
// It sets InfrastructureRef and Topology on the initial Create (avoiding Get+Update),
// and combines phase + conditions into a single Status().Update() when both are present.
func (te *testEnv) createClusterFull(ctx context.Context, opts clusterCreateOptions) {
	te.t.Helper()

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.name,
			Namespace: opts.namespace,
			Labels: map[string]string{
				clusterv1.ClusterNameLabel: opts.name,
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

	// Set infrastructure ref before Create
	if opts.customInfraRef != nil {
		// Custom InfrastructureRef overrides provider
		cluster.Spec.InfrastructureRef = &corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       opts.customInfraRef.kind,
			Name:       opts.customInfraRef.name,
			Namespace:  opts.namespace,
		}
	} else {
		// Set infrastructure ref based on provider
		switch opts.provider {
		case "aws":
			cluster.Spec.InfrastructureRef = &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
				Kind:       "AWSCluster",
				Name:       opts.name,
				Namespace:  opts.namespace,
			}
		case "azure":
			cluster.Spec.InfrastructureRef = &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "AzureCluster",
				Name:       opts.name,
				Namespace:  opts.namespace,
			}
		case "gcp":
			cluster.Spec.InfrastructureRef = &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "GCPCluster",
				Name:       opts.name,
				Namespace:  opts.namespace,
			}
		case "vsphere":
			cluster.Spec.InfrastructureRef = &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "VSphereCluster",
				Name:       opts.name,
				Namespace:  opts.namespace,
			}
		case "vcd":
			cluster.Spec.InfrastructureRef = &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "VCDCluster",
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
	needsStatusUpdate := opts.phase != "" || len(opts.conditions) > 0 || opts.controlPlaneReady != nil || opts.infraReady != nil
	if needsStatusUpdate {
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
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{Namespace: namespace, Name: clusterName}
	if err := te.ctrlClient.Get(ctx, key, cluster); err != nil {
		te.t.Fatalf("failed to get cluster %s/%s: %v", namespace, clusterName, err)
	}
	cluster.Spec.ControlPlaneRef = &corev1.ObjectReference{
		APIVersion: "controlplane.cluster.x-k8s.io/v1beta1",
		Kind:       "KubeadmControlPlane",
		Name:       kcpName,
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
		replicas := int32(opts.replicas)
		md.Spec.Replicas = &replicas
	}

	if opts.version != "" {
		md.Spec.Template.Spec.Version = &opts.version
	}

	if err := te.ctrlClient.Create(ctx, md); err != nil {
		te.t.Fatalf("failed to create MachineDeployment %s/%s: %v", opts.namespace, opts.name, err)
	}

	// Update status if needed
	needsStatusUpdate := opts.hasStatus || opts.phase != "" || opts.readyReplicas > 0 || opts.updatedReplicas > 0 || opts.availableReplicas > 0 || opts.statusReplicas > 0
	if needsStatusUpdate {
		if opts.phase != "" {
			md.Status.Phase = opts.phase
		}
		md.Status.Replicas = int32(opts.statusReplicas)
		md.Status.ReadyReplicas = int32(opts.readyReplicas)
		md.Status.UpdatedReplicas = int32(opts.updatedReplicas)
		md.Status.AvailableReplicas = int32(opts.availableReplicas)
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
	replicas          int
	nilReplicas       bool
	version           string
	phase             string
	hasStatus         bool // explicit flag to trigger status update even with zero values
	statusReplicas    int
	readyReplicas     int
	updatedReplicas   int
	availableReplicas int
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
		replicas := int32(opts.replicas)
		ms.Spec.Replicas = &replicas
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
		isController := true
		ms.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: "cluster.x-k8s.io/v1beta1",
				Kind:       "MachineDeployment",
				Name:       opts.ownerMDName,
				UID:        "test-uid",
				Controller: &isController,
			},
		}
	} else if opts.ownerKind != "" {
		isController := true
		ms.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: "cluster.x-k8s.io/v1beta1",
				Kind:       opts.ownerKind,
				Name:       opts.ownerName,
				UID:        "test-uid",
				Controller: &isController,
			},
		}
	}

	if err := te.ctrlClient.Create(ctx, ms); err != nil {
		te.t.Fatalf("failed to create MachineSet %s/%s: %v", opts.namespace, opts.name, err)
	}

	// Update status if needed
	needsStatusUpdate := opts.statusReplicas > 0 || opts.readyReplicas > 0 || opts.availableReplicas > 0 || opts.failureReason != "" || opts.failureMessage != "" || len(opts.conditions) > 0
	if needsStatusUpdate {
		ms.Status.Replicas = int32(opts.statusReplicas)
		ms.Status.ReadyReplicas = int32(opts.readyReplicas)
		ms.Status.AvailableReplicas = int32(opts.availableReplicas)
		if opts.failureReason != "" {
			reason := capierrors.MachineSetStatusError(opts.failureReason)
			ms.Status.FailureReason = &reason
		}
		if opts.failureMessage != "" {
			ms.Status.FailureMessage = &opts.failureMessage
		}
		for _, cond := range opts.conditions {
			ms.Status.Conditions = append(ms.Status.Conditions, clusterv1.Condition{
				Type:               clusterv1.ConditionType(cond.condType),
				Status:             corev1.ConditionStatus(cond.status),
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
	replicas          int
	nilReplicas       bool
	version           string
	statusReplicas    int
	readyReplicas     int
	availableReplicas int
	infraRefKind      string
	infraRefName      string
	bootstrapKind     string
	bootstrapName     string
	ownerMDName       string
	ownerKind         string
	ownerName         string
	conditions        []machineSetCondition
	failureReason     string
	failureMessage    string
}

// nodeCreateOptions holds all parameters for creating a fully-configured Kubernetes Node.
type nodeCreateOptions struct {
	name          string
	providerID    string
	unschedulable bool
	conditions    []nodeCondition
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
		OperatingSystem:         opts.nodeInfo.os,
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
			Status:  corev1.ConditionStatus(cond.status),
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

	// Set capacity
	node.Status.Capacity = corev1.ResourceList{}
	if opts.capacity.cpu != "" {
		node.Status.Capacity[corev1.ResourceCPU] = resource.MustParse(opts.capacity.cpu)
	}
	if opts.capacity.memory != "" {
		node.Status.Capacity[corev1.ResourceMemory] = resource.MustParse(opts.capacity.memory)
	}
	if opts.capacity.pods != "" {
		node.Status.Capacity[corev1.ResourcePods] = resource.MustParse(opts.capacity.pods)
	}

	// Set allocatable
	node.Status.Allocatable = corev1.ResourceList{}
	if opts.allocatable.cpu != "" {
		node.Status.Allocatable[corev1.ResourceCPU] = resource.MustParse(opts.allocatable.cpu)
	}
	if opts.allocatable.memory != "" {
		node.Status.Allocatable[corev1.ResourceMemory] = resource.MustParse(opts.allocatable.memory)
	}
	if opts.allocatable.pods != "" {
		node.Status.Allocatable[corev1.ResourcePods] = resource.MustParse(opts.allocatable.pods)
	}

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
	conditions    []machineCondition
	addresses     []machineAddress
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

	needsStatusUpdate := opts.phase != "" || opts.nodeRefName != "" || len(opts.conditions) > 0 || len(opts.addresses) > 0
	if needsStatusUpdate {
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
				Status:             corev1.ConditionStatus(cond.status),
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
