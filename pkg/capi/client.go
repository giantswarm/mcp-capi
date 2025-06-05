package capi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client provides access to CAPI resources in a Kubernetes cluster
type Client struct {
	// k8sClient is the standard Kubernetes client
	k8sClient kubernetes.Interface

	// ctrlClient is the controller-runtime client for CAPI resources
	ctrlClient client.Client

	// config is the rest config used to connect
	config *rest.Config
}

// NewClient creates a new CAPI client
func NewClient(kubeconfig string) (*Client, error) {
	config, err := loadConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Create standard Kubernetes client
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Create controller-runtime client with CAPI scheme
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add core types to scheme: %w", err)
	}
	if err := clusterv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add CAPI to scheme: %w", err)
	}

	ctrlClient, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create controller client: %w", err)
	}

	return &Client{
		k8sClient:  k8sClient,
		ctrlClient: ctrlClient,
		config:     config,
	}, nil
}

// loadConfig loads the kubeconfig from various sources
func loadConfig(kubeconfig string) (*rest.Config, error) {
	// If kubeconfig is provided, use it
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Try KUBECONFIG env var
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigEnv)
	}

	// Try default location
	if home := homedir.HomeDir(); home != "" {
		defaultPath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(defaultPath); err == nil {
			return clientcmd.BuildConfigFromFlags("", defaultPath)
		}
	}

	return nil, fmt.Errorf("no kubeconfig found")
}

// GetK8sClient returns the standard Kubernetes client
func (c *Client) GetK8sClient() kubernetes.Interface {
	return c.k8sClient
}

// GetCtrlClient returns the controller-runtime client
func (c *Client) GetCtrlClient() client.Client {
	return c.ctrlClient
}

// ListClusters lists all CAPI clusters in the given namespace
func (c *Client) ListClusters(ctx context.Context, namespace string) (*clusterv1.ClusterList, error) {
	clusterList := &clusterv1.ClusterList{}

	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}

	if err := c.ctrlClient.List(ctx, clusterList, opts...); err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	return clusterList, nil
}

// GetCluster retrieves a specific cluster
func (c *Client) GetCluster(ctx context.Context, namespace, name string) (*clusterv1.Cluster, error) {
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.ctrlClient.Get(ctx, key, cluster); err != nil {
		return nil, fmt.Errorf("failed to get cluster %s/%s: %w", namespace, name, err)
	}

	return cluster, nil
}

// ListMachines lists all machines for a given cluster
func (c *Client) ListMachines(ctx context.Context, namespace, clusterName string) (*clusterv1.MachineList, error) {
	machineList := &clusterv1.MachineList{}

	opts := []client.ListOption{
		client.InNamespace(namespace),
	}

	if clusterName != "" {
		opts = append(opts, client.MatchingLabels{
			clusterv1.ClusterNameLabel: clusterName,
		})
	}

	if err := c.ctrlClient.List(ctx, machineList, opts...); err != nil {
		return nil, fmt.Errorf("failed to list machines: %w", err)
	}

	return machineList, nil
}

// GetMachine retrieves a specific machine
func (c *Client) GetMachine(ctx context.Context, namespace, name string) (*clusterv1.Machine, error) {
	machine := &clusterv1.Machine{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.ctrlClient.Get(ctx, key, machine); err != nil {
		return nil, fmt.Errorf("failed to get machine %s/%s: %w", namespace, name, err)
	}

	return machine, nil
}

// ListMachineDeployments lists all machine deployments
func (c *Client) ListMachineDeployments(ctx context.Context, namespace, clusterName string) (*clusterv1.MachineDeploymentList, error) {
	mdList := &clusterv1.MachineDeploymentList{}

	opts := []client.ListOption{
		client.InNamespace(namespace),
	}

	if clusterName != "" {
		opts = append(opts, client.MatchingLabels{
			clusterv1.ClusterNameLabel: clusterName,
		})
	}

	if err := c.ctrlClient.List(ctx, mdList, opts...); err != nil {
		return nil, fmt.Errorf("failed to list machine deployments: %w", err)
	}

	return mdList, nil
}

// GetMachineDeployment retrieves a specific machine deployment
func (c *Client) GetMachineDeployment(ctx context.Context, namespace, name string) (*clusterv1.MachineDeployment, error) {
	md := &clusterv1.MachineDeployment{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.ctrlClient.Get(ctx, key, md); err != nil {
		return nil, fmt.Errorf("failed to get machine deployment %s/%s: %w", namespace, name, err)
	}

	return md, nil
}

// GetKubeconfig retrieves the kubeconfig for a workload cluster
func (c *Client) GetKubeconfig(ctx context.Context, namespace, clusterName string) (string, error) {
	// The kubeconfig is typically stored in a secret named {cluster-name}-kubeconfig
	secretName := fmt.Sprintf("%s-kubeconfig", clusterName)

	secret, err := c.k8sClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get kubeconfig secret: %w", err)
	}

	// The kubeconfig is typically stored in the 'value' key
	kubeconfigData, exists := secret.Data["value"]
	if !exists {
		// Try 'data' key as alternative
		kubeconfigData, exists = secret.Data["data"]
		if !exists {
			// List all keys for debugging
			var keys []string
			for k := range secret.Data {
				keys = append(keys, k)
			}
			return "", fmt.Errorf("kubeconfig not found in secret, available keys: %v", keys)
		}
	}

	return string(kubeconfigData), nil
}

// PauseCluster pauses reconciliation for a cluster by adding the cluster.x-k8s.io/paused annotation
func (c *Client) PauseCluster(ctx context.Context, namespace, name string) error {
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.ctrlClient.Get(ctx, key, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Add paused annotation
	if cluster.Annotations == nil {
		cluster.Annotations = make(map[string]string)
	}
	cluster.Annotations[clusterv1.PausedAnnotation] = "true"

	if err := c.ctrlClient.Update(ctx, cluster); err != nil {
		return fmt.Errorf("failed to pause cluster: %w", err)
	}

	return nil
}

// ResumeCluster resumes reconciliation for a cluster by removing the cluster.x-k8s.io/paused annotation
func (c *Client) ResumeCluster(ctx context.Context, namespace, name string) error {
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.ctrlClient.Get(ctx, key, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Remove paused annotation
	if cluster.Annotations != nil {
		delete(cluster.Annotations, clusterv1.PausedAnnotation)
	}

	if err := c.ctrlClient.Update(ctx, cluster); err != nil {
		return fmt.Errorf("failed to resume cluster: %w", err)
	}

	return nil
}

// DeleteCluster deletes a CAPI cluster
func (c *Client) DeleteCluster(ctx context.Context, namespace, name string) error {
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.ctrlClient.Get(ctx, key, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Delete the cluster
	if err := c.ctrlClient.Delete(ctx, cluster); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	return nil
}

// CreateClusterOptions contains options for creating a new cluster
type CreateClusterOptions struct {
	Name              string
	Namespace         string
	InfraProvider     string
	KubernetesVersion string
	ControlPlaneCount int32
	WorkerCount       int32
	Region            string
	InstanceType      string
}

// CreateCluster creates a new CAPI cluster with basic configuration
func (c *Client) CreateCluster(ctx context.Context, opts CreateClusterOptions) (*clusterv1.Cluster, error) {
	// For now, we'll create a basic cluster object
	// In a real implementation, this would create all the necessary resources
	// (Cluster, KubeadmControlPlane, MachineDeployment, etc.)

	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/provider": opts.InfraProvider,
			},
		},
		Spec: clusterv1.ClusterSpec{
			ClusterNetwork: &clusterv1.ClusterNetwork{
				Pods: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"192.168.0.0/16"},
				},
				Services: &clusterv1.NetworkRanges{
					CIDRBlocks: []string{"10.96.0.0/12"},
				},
			},
			ControlPlaneRef: &corev1.ObjectReference{
				APIVersion: "controlplane.cluster.x-k8s.io/v1beta1",
				Kind:       "KubeadmControlPlane",
				Name:       opts.Name + "-control-plane",
			},
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: getInfraAPIVersion(opts.InfraProvider),
				Kind:       getInfraKind(opts.InfraProvider),
				Name:       opts.Name,
			},
		},
	}

	// Create the cluster
	if err := c.ctrlClient.Create(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	return cluster, nil
}

// Helper functions to map provider to API versions and kinds
func getInfraAPIVersion(provider string) string {
	switch provider {
	case "aws":
		return "infrastructure.cluster.x-k8s.io/v1beta2"
	case "azure":
		return "infrastructure.cluster.x-k8s.io/v1beta1"
	case "gcp":
		return "infrastructure.cluster.x-k8s.io/v1beta1"
	case "vsphere":
		return "infrastructure.cluster.x-k8s.io/v1beta1"
	default:
		return "infrastructure.cluster.x-k8s.io/v1beta1"
	}
}

func getInfraKind(provider string) string {
	switch provider {
	case "aws":
		return "AWSCluster"
	case "azure":
		return "AzureCluster"
	case "gcp":
		return "GCPCluster"
	case "vsphere":
		return "VSphereCluster"
	default:
		return "Cluster"
	}
}

// ClusterHealthStatus represents the health status of a cluster
type ClusterHealthStatus struct {
	Healthy           bool
	ControlPlaneReady bool
	WorkersReady      bool
	InfraReady        bool
	Issues            []string
	Warnings          []string
}

// GetClusterHealth checks the health of a cluster
func (c *Client) GetClusterHealth(ctx context.Context, namespace, name string) (*ClusterHealthStatus, error) {
	status, err := c.GetClusterStatus(ctx, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster status: %w", err)
	}
	
	health := &ClusterHealthStatus{
		Healthy:           true,
		ControlPlaneReady: status.ControlPlaneReady,
		InfraReady:        status.InfraReady,
		Issues:            []string{},
		Warnings:          []string{},
	}
	
	// Check control plane
	if !status.ControlPlaneReady {
		health.Healthy = false
		health.Issues = append(health.Issues, "Control plane is not ready")
	}
	
	// Check infrastructure
	if !status.InfraReady {
		health.Healthy = false
		health.Issues = append(health.Issues, "Infrastructure is not ready")
	}
	
	// Check workers
	machines, err := c.ListMachines(ctx, namespace, name)
	if err == nil {
		readyMachines := 0
		totalMachines := len(machines.Items)
		
		for _, machine := range machines.Items {
			for _, condition := range machine.Status.Conditions {
				if condition.Type == "Ready" && condition.Status == "True" {
					readyMachines++
					break
				}
			}
		}
		
		health.WorkersReady = readyMachines == totalMachines && totalMachines > 0
		if !health.WorkersReady {
			health.Healthy = false
			health.Issues = append(health.Issues, fmt.Sprintf("Only %d/%d machines are ready", readyMachines, totalMachines))
		}
	}
	
	// Check conditions for issues
	for _, condition := range status.Conditions {
		if condition.Status != "True" && condition.Severity == "Error" {
			health.Healthy = false
			health.Issues = append(health.Issues, fmt.Sprintf("%s: %s", condition.Type, condition.Message))
		} else if condition.Status != "True" && condition.Severity == "Warning" {
			health.Warnings = append(health.Warnings, fmt.Sprintf("%s: %s", condition.Type, condition.Message))
		}
	}
	
	// Check phase
	if status.Phase != "Provisioned" && status.Phase != "" {
		health.Warnings = append(health.Warnings, fmt.Sprintf("Cluster phase is '%s', expected 'Provisioned'", status.Phase))
	}
	
	return health, nil
}
