package harness

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/api/controlplane/kubeadm/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const kubeconfigContextName = "envtest"

// testEnv wraps envtest (a local Kubernetes API server) for integration testing.
// It provides methods to create test resources (namespaces, clusters) and manages
// the lifecycle of the test environment.
type testEnv struct {
	t              TestingT
	env            *envtest.Environment
	k8sClient      kubernetes.Interface
	ctrlClient     client.Client
	kubeconfigPath string
}

// newTestEnv initializes envtest with CAPI CRDs
func newTestEnv(t TestingT) *testEnv {
	t.Helper()

	// Create envtest environment with CRD directory
	// This will load all CAPI CRDs from the crds/ directory
	// Use 'make download-crds' to fetch the latest CRDs from upstream
	// Silence controller-runtime logs during tests
	ctrl.SetLogger(zap.New(zap.WriteTo(io.Discard)))

	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{getCRDPath(t)},
		ErrorIfCRDPathMissing: true,
	}

	// Start the test environment
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("failed to start test environment: %v", err)
	}

	// Cleanup on failure - t.Fatalf() triggers deferred functions via runtime.Goexit()
	success := false
	defer func() {
		if !success {
			if stopErr := env.Stop(); stopErr != nil {
				t.Logf("failed to stop environment during cleanup: %v", stopErr)
			}
		}
	}()

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
	kubeconfigPath := filepath.Join(t.TempDir(), "envtest.kubeconfig")
	if err := writeKubeconfig(cfg, kubeconfigPath); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	success = true
	return &testEnv{
		t:              t,
		env:            env,
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
func getCRDPath(t TestingT) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get current file path from runtime.Caller")
	}
	crdPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "crds")
	if _, err := os.Stat(crdPath); os.IsNotExist(err) {
		t.Fatalf("CRD directory does not exist: %s (run 'make download-crds' to fetch CRDs)", crdPath)
	}
	return crdPath
}

// teardown stops the envtest environment and cleans up resources.
// Note: kubeconfig cleanup is handled automatically by t.TempDir().
func (te *testEnv) teardown() {
	te.t.Helper()
	if te.env != nil {
		if err := te.env.Stop(); err != nil {
			te.t.Errorf("failed to stop test environment: %v", err)
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

// createAWSCluster creates a CAPI Cluster resource with AWS infrastructure reference
func (te *testEnv) createAWSCluster(ctx context.Context, namespace, name string) {
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
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
				Kind:       "AWSCluster",
				Name:       name,
				Namespace:  namespace,
			},
			ClusterNetwork: &clusterv1.ClusterNetwork{
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"192.168.0.0/16"},
				},
			},
		},
	}

	if err := te.ctrlClient.Create(ctx, cluster); err != nil {
		te.t.Fatalf("failed to create AWS cluster %s/%s: %v", namespace, name, err)
	}
}

// createAzureCluster creates a CAPI Cluster resource with Azure infrastructure reference
func (te *testEnv) createAzureCluster(ctx context.Context, namespace, name string) {
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
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "AzureCluster",
				Name:       name,
				Namespace:  namespace,
			},
			ClusterNetwork: &clusterv1.ClusterNetwork{
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"192.168.0.0/16"},
				},
			},
		},
	}

	if err := te.ctrlClient.Create(ctx, cluster); err != nil {
		te.t.Fatalf("failed to create Azure cluster %s/%s: %v", namespace, name, err)
	}
}

// createVSphereCluster creates a CAPI Cluster resource with vSphere infrastructure reference
func (te *testEnv) createVSphereCluster(ctx context.Context, namespace, name string) {
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
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "VSphereCluster",
				Name:       name,
				Namespace:  namespace,
			},
			ClusterNetwork: &clusterv1.ClusterNetwork{
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"192.168.0.0/16"},
				},
			},
		},
	}

	if err := te.ctrlClient.Create(ctx, cluster); err != nil {
		te.t.Fatalf("failed to create vSphere cluster %s/%s: %v", namespace, name, err)
	}
}

// createVCDCluster creates a CAPI Cluster resource with VCD infrastructure reference
func (te *testEnv) createVCDCluster(ctx context.Context, namespace, name string) {
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
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "VCDCluster",
				Name:       name,
				Namespace:  namespace,
			},
			ClusterNetwork: &clusterv1.ClusterNetwork{
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"192.168.0.0/16"},
				},
			},
		},
	}

	if err := te.ctrlClient.Create(ctx, cluster); err != nil {
		te.t.Fatalf("failed to create VCD cluster %s/%s: %v", namespace, name, err)
	}
}

// createGCPCluster creates a CAPI Cluster resource with GCP infrastructure reference
func (te *testEnv) createGCPCluster(ctx context.Context, namespace, name string) {
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
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "GCPCluster",
				Name:       name,
				Namespace:  namespace,
			},
			ClusterNetwork: &clusterv1.ClusterNetwork{
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"192.168.0.0/16"},
				},
			},
		},
	}

	if err := te.ctrlClient.Create(ctx, cluster); err != nil {
		te.t.Fatalf("failed to create GCP cluster %s/%s: %v", namespace, name, err)
	}
}

// setClusterPhase updates the phase of an existing cluster.
func (te *testEnv) setClusterPhase(ctx context.Context, namespace, name, phase string) {
	te.t.Helper()
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := te.ctrlClient.Get(ctx, key, cluster); err != nil {
		te.t.Fatalf("failed to get cluster %s/%s: %v", namespace, name, err)
	}
	cluster.Status.Phase = phase
	if err := te.ctrlClient.Status().Update(ctx, cluster); err != nil {
		te.t.Fatalf("failed to set phase on cluster %s/%s: %v", namespace, name, err)
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

// ptr returns a pointer to the given string value.
func ptr(s string) *string {
	return &s
}

// setClusterVersion updates the kubernetes version of an existing cluster.
// Version is stored in Spec.Topology.Version. Since Topology requires a Class field,
// we set a placeholder class value.
func (te *testEnv) setClusterVersion(ctx context.Context, namespace, name, version string) {
	te.t.Helper()
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := te.ctrlClient.Get(ctx, key, cluster); err != nil {
		te.t.Fatalf("failed to get cluster %s/%s: %v", namespace, name, err)
	}
	cluster.Spec.Topology = &clusterv1.Topology{
		Class:   "default",
		Version: version,
	}
	if err := te.ctrlClient.Update(ctx, cluster); err != nil {
		te.t.Fatalf("failed to set version on cluster %s/%s: %v", namespace, name, err)
	}
}

// setClusterConditions updates the conditions on an existing cluster.
func (te *testEnv) setClusterConditions(ctx context.Context, namespace, name string, conditions []clusterv1.Condition) {
	te.t.Helper()
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{Namespace: namespace, Name: name}
	if err := te.ctrlClient.Get(ctx, key, cluster); err != nil {
		te.t.Fatalf("failed to get cluster %s/%s: %v", namespace, name, err)
	}
	cluster.Status.Conditions = conditions
	if err := te.ctrlClient.Status().Update(ctx, cluster); err != nil {
		te.t.Fatalf("failed to set conditions on cluster %s/%s: %v", namespace, name, err)
	}
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
