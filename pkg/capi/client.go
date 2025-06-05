package capi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
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

// DeleteMachineOptions contains options for deleting a machine
type DeleteMachineOptions struct {
	Namespace string
	Name      string
	Force     bool
}

// DeleteMachine deletes a CAPI machine
func (c *Client) DeleteMachine(ctx context.Context, opts DeleteMachineOptions) error {
	machine := &clusterv1.Machine{}
	key := client.ObjectKey{
		Namespace: opts.Namespace,
		Name:      opts.Name,
	}

	// First, get the machine to check if it exists
	if err := c.ctrlClient.Get(ctx, key, machine); err != nil {
		return fmt.Errorf("failed to get machine: %w", err)
	}

	// If not forcing, check if machine is safe to delete
	if !opts.Force {
		// Check if machine is healthy
		for _, condition := range machine.Status.Conditions {
			if condition.Type == clusterv1.MachineHealthCheckSucceededCondition && condition.Status == corev1.ConditionTrue {
				return fmt.Errorf("machine %s is healthy, use force=true to delete anyway", machine.Name)
			}
		}

		// Check if it's a control plane machine with only one replica
		if util.IsControlPlaneMachine(machine) {
			// This is a simplified check - in production you'd want to check the actual replica count
			return fmt.Errorf("cannot delete control plane machine %s without force=true", machine.Name)
		}
	}

	// Delete the machine
	if err := c.ctrlClient.Delete(ctx, machine); err != nil {
		return fmt.Errorf("failed to delete machine: %w", err)
	}

	return nil
}

// RemediateMachineOptions contains options for remediating a machine
type RemediateMachineOptions struct {
	Namespace string
	Name      string
}

// RemediateMachine triggers machine health check remediation by annotating the machine
func (c *Client) RemediateMachine(ctx context.Context, opts RemediateMachineOptions) error {
	machine := &clusterv1.Machine{}
	key := client.ObjectKey{
		Namespace: opts.Namespace,
		Name:      opts.Name,
	}

	if err := c.ctrlClient.Get(ctx, key, machine); err != nil {
		return fmt.Errorf("failed to get machine: %w", err)
	}

	// Add remediation annotation
	if machine.Annotations == nil {
		machine.Annotations = make(map[string]string)
	}
	machine.Annotations["cluster.x-k8s.io/remediate-machine"] = fmt.Sprintf("%d", time.Now().Unix())

	// Update the machine
	if err := c.ctrlClient.Update(ctx, machine); err != nil {
		return fmt.Errorf("failed to update machine with remediation annotation: %w", err)
	}

	return nil
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
		return nil, fmt.Errorf("failed to get machine deployment: %w", err)
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

// UpgradeClusterOptions contains options for upgrading a cluster
type UpgradeClusterOptions struct {
	Namespace      string
	Name           string
	TargetVersion  string
	UpgradeWorkers bool
}

// UpgradeCluster upgrades a CAPI cluster to a new Kubernetes version
func (c *Client) UpgradeCluster(ctx context.Context, opts UpgradeClusterOptions) error {
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{
		Namespace: opts.Namespace,
		Name:      opts.Name,
	}

	if err := c.ctrlClient.Get(ctx, key, cluster); err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Update the control plane version
	if cluster.Spec.ControlPlaneRef != nil {
		switch cluster.Spec.ControlPlaneRef.Kind {
		case "KubeadmControlPlane":
			kcp := &controlplanev1.KubeadmControlPlane{}
			cpKey := client.ObjectKey{
				Namespace: cluster.Spec.ControlPlaneRef.Namespace,
				Name:      cluster.Spec.ControlPlaneRef.Name,
			}
			if err := c.ctrlClient.Get(ctx, cpKey, kcp); err != nil {
				return fmt.Errorf("failed to get control plane: %w", err)
			}

			// Update version
			kcp.Spec.Version = opts.TargetVersion
			if err := c.ctrlClient.Update(ctx, kcp); err != nil {
				return fmt.Errorf("failed to update control plane version: %w", err)
			}
		default:
			return fmt.Errorf("unsupported control plane type: %s", cluster.Spec.ControlPlaneRef.Kind)
		}
	}

	// Update worker nodes if requested
	if opts.UpgradeWorkers {
		mdList, err := c.ListMachineDeployments(ctx, opts.Namespace, opts.Name)
		if err != nil {
			return fmt.Errorf("failed to list machine deployments: %w", err)
		}

		for i := range mdList.Items {
			md := &mdList.Items[i]
			if md.Spec.Template.Spec.Version != nil {
				*md.Spec.Template.Spec.Version = opts.TargetVersion
				if err := c.ctrlClient.Update(ctx, md); err != nil {
					return fmt.Errorf("failed to update machine deployment %s: %w", md.Name, err)
				}
			}
		}
	}

	return nil
}

// UpdateClusterOptions contains options for updating a cluster
type UpdateClusterOptions struct {
	Namespace   string
	Name        string
	Labels      map[string]string
	Annotations map[string]string
}

// UpdateCluster updates a CAPI cluster's metadata
func (c *Client) UpdateCluster(ctx context.Context, opts UpdateClusterOptions) (*clusterv1.Cluster, error) {
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{
		Namespace: opts.Namespace,
		Name:      opts.Name,
	}

	if err := c.ctrlClient.Get(ctx, key, cluster); err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Update labels
	if opts.Labels != nil {
		if cluster.Labels == nil {
			cluster.Labels = make(map[string]string)
		}
		for k, v := range opts.Labels {
			if v == "" {
				// Empty value means remove the label
				delete(cluster.Labels, k)
			} else {
				cluster.Labels[k] = v
			}
		}
	}

	// Update annotations
	if opts.Annotations != nil {
		if cluster.Annotations == nil {
			cluster.Annotations = make(map[string]string)
		}
		for k, v := range opts.Annotations {
			if v == "" {
				// Empty value means remove the annotation
				delete(cluster.Annotations, k)
			} else {
				cluster.Annotations[k] = v
			}
		}
	}

	if err := c.ctrlClient.Update(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to update cluster: %w", err)
	}

	return cluster, nil
}

// MoveClusterOptions contains options for moving a cluster
type MoveClusterOptions struct {
	Namespace        string
	Name             string
	TargetKubeconfig string
	TargetNamespace  string
	DryRun           bool
}

// MoveCluster prepares a cluster for migration to another management cluster
// Note: This is a simplified implementation that exports the cluster resources
func (c *Client) MoveCluster(ctx context.Context, opts MoveClusterOptions) (string, error) {
	// Get the cluster
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{
		Namespace: opts.Namespace,
		Name:      opts.Name,
	}

	if err := c.ctrlClient.Get(ctx, key, cluster); err != nil {
		return "", fmt.Errorf("failed to get cluster: %w", err)
	}

	// Prepare target namespace
	targetNs := opts.TargetNamespace
	if targetNs == "" {
		targetNs = opts.Namespace
	}

	// Create a YAML manifest for the move
	var manifest strings.Builder
	manifest.WriteString("# Cluster Move Manifest\n")
	manifest.WriteString(fmt.Sprintf("# Source: %s/%s\n", opts.Namespace, opts.Name))
	manifest.WriteString(fmt.Sprintf("# Target: %s/%s\n", targetNs, opts.Name))
	manifest.WriteString("# Apply this manifest to the target management cluster\n")
	manifest.WriteString("---\n")

	// Note: In a real implementation, you would:
	// 1. Use clusterctl move command or equivalent
	// 2. Export all related resources (Machines, MachineDeployments, etc.)
	// 3. Handle infrastructure-specific resources
	// 4. Pause source cluster before move
	// 5. Update object references

	manifest.WriteString("# This is a placeholder implementation\n")
	manifest.WriteString("# In production, use 'clusterctl move' command\n")
	manifest.WriteString("# Example: clusterctl move --to-kubeconfig=target.kubeconfig\n")

	return manifest.String(), nil
}

// BackupClusterOptions contains options for backing up a cluster
type BackupClusterOptions struct {
	Namespace      string
	Name           string
	IncludeSecrets bool
	OutputFormat   string // yaml or json
}

// BackupCluster creates a backup of cluster resources
func (c *Client) BackupCluster(ctx context.Context, opts BackupClusterOptions) (string, error) {
	// Get the cluster
	cluster := &clusterv1.Cluster{}
	key := client.ObjectKey{
		Namespace: opts.Namespace,
		Name:      opts.Name,
	}

	if err := c.ctrlClient.Get(ctx, key, cluster); err != nil {
		return "", fmt.Errorf("failed to get cluster: %w", err)
	}

	// Create backup manifest
	var backup strings.Builder
	backup.WriteString("# Cluster Backup\n")
	backup.WriteString(fmt.Sprintf("# Cluster: %s/%s\n", opts.Namespace, opts.Name))
	backup.WriteString(fmt.Sprintf("# Date: %s\n", fmt.Sprintf("%v", cluster.CreationTimestamp)))
	backup.WriteString("# Resources included:\n")
	backup.WriteString("# - Cluster\n")
	backup.WriteString("# - Control Plane\n")
	backup.WriteString("# - MachineDeployments\n")
	backup.WriteString("# - Infrastructure Resources\n")
	if opts.IncludeSecrets {
		backup.WriteString("# - Secrets (kubeconfig, certificates)\n")
	}
	backup.WriteString("---\n")

	// Note: In a real implementation, you would:
	// 1. Export the Cluster resource
	// 2. Export ControlPlane resources
	// 3. Export all Machines and MachineDeployments
	// 4. Export infrastructure-specific resources
	// 5. Optionally export secrets (kubeconfig, certs)
	// 6. Add restore instructions

	backup.WriteString("# This is a placeholder implementation\n")
	backup.WriteString("# Use velero or similar tools for complete cluster backup\n")
	backup.WriteString("# Example: velero backup create cluster-backup --include-namespaces=<namespace>\n")

	return backup.String(), nil
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

// CreateMachineDeploymentOptions contains options for creating a machine deployment
type CreateMachineDeploymentOptions struct {
	Namespace          string
	Name               string
	ClusterName        string
	Replicas           int32
	InfrastructureRef  corev1.ObjectReference
	BootstrapConfigRef corev1.ObjectReference
	Version            string
	Labels             map[string]string
	NodeDrainTimeout   *metav1.Duration
	MinReadySeconds    int32
}

// CreateMachineDeployment creates a new CAPI MachineDeployment
func (c *Client) CreateMachineDeployment(ctx context.Context, opts CreateMachineDeploymentOptions) (*clusterv1.MachineDeployment, error) {
	// Create the machine deployment
	md := &clusterv1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
			Labels:    opts.Labels,
		},
		Spec: clusterv1.MachineDeploymentSpec{
			ClusterName: opts.ClusterName,
			Replicas:    &opts.Replicas,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"machinedeployment": opts.Name,
				},
			},
			Template: clusterv1.MachineTemplateSpec{
				ObjectMeta: clusterv1.ObjectMeta{
					Labels: map[string]string{
						"machinedeployment":                  opts.Name,
						clusterv1.ClusterNameLabel:           opts.ClusterName,
						clusterv1.MachineDeploymentNameLabel: opts.Name,
					},
				},
				Spec: clusterv1.MachineSpec{
					ClusterName:       opts.ClusterName,
					Version:           &opts.Version,
					InfrastructureRef: opts.InfrastructureRef,
					Bootstrap: clusterv1.Bootstrap{
						ConfigRef: &opts.BootstrapConfigRef,
					},
				},
			},
			MinReadySeconds: &opts.MinReadySeconds,
		},
	}

	if opts.NodeDrainTimeout != nil {
		md.Spec.Template.Spec.NodeDrainTimeout = opts.NodeDrainTimeout
	}

	// Create the machine deployment
	if err := c.ctrlClient.Create(ctx, md); err != nil {
		return nil, fmt.Errorf("failed to create machine deployment: %w", err)
	}

	return md, nil
}

// ScaleClusterOptions contains options for scaling a cluster
// ... existing code ...
