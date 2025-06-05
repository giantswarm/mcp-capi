package capi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
