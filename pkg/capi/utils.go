package capi

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterStatus represents the status of a CAPI cluster
type ClusterStatus struct {
	Name              string
	Namespace         string
	Phase             string
	Ready             bool
	ControlPlaneReady bool
	InfraReady        bool
	Version           string
	Provider          Provider
	TotalMachines     int
	ReadyMachines     int
	Conditions        clusterv1.Conditions
}

// GetClusterStatus retrieves comprehensive status information for a cluster
func (c *Client) GetClusterStatus(ctx context.Context, namespace, name string) (*ClusterStatus, error) {
	cluster, err := c.GetCluster(ctx, namespace, name)
	if err != nil {
		return nil, err
	}

	status := &ClusterStatus{
		Name:              cluster.Name,
		Namespace:         cluster.Namespace,
		Phase:             string(cluster.Status.Phase),
		Ready:             conditions.IsTrue(cluster, clusterv1.ReadyCondition),
		ControlPlaneReady: cluster.Status.ControlPlaneReady,
		InfraReady:        cluster.Status.InfrastructureReady,
		Conditions:        cluster.Status.Conditions,
	}

	// Get version from cluster spec
	if cluster.Spec.Topology != nil && cluster.Spec.Topology.Version != "" {
		status.Version = cluster.Spec.Topology.Version
	}

	// Get provider information
	provider, _ := c.GetProviderForCluster(ctx, namespace, name)
	status.Provider = provider

	// Get machine counts
	machines, err := c.ListMachines(ctx, namespace, name)
	if err == nil {
		status.TotalMachines = len(machines.Items)
		for _, machine := range machines.Items {
			if machine.Status.NodeRef != nil {
				status.ReadyMachines++
			}
		}
	}

	// Get control plane version if available
	if cluster.Spec.ControlPlaneRef != nil && status.Version == "" {
		if cluster.Spec.ControlPlaneRef.Kind == "KubeadmControlPlane" {
			kcp, err := c.GetKubeadmControlPlane(ctx, namespace, cluster.Spec.ControlPlaneRef.Name)
			if err == nil && kcp.Spec.Version != "" {
				status.Version = kcp.Spec.Version
			}
		}
	}

	return status, nil
}

// GetKubeconfig retrieves the kubeconfig for a workload cluster
func (c *Client) GetKubeconfig(ctx context.Context, namespace, clusterName string) (string, error) {
	secretName := fmt.Sprintf("%s-kubeconfig", clusterName)

	secret := &corev1.Secret{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      secretName,
	}

	if err := c.ctrlClient.Get(ctx, key, secret); err != nil {
		return "", fmt.Errorf("failed to get kubeconfig secret: %w", err)
	}

	kubeconfig, ok := secret.Data["value"]
	if !ok {
		return "", fmt.Errorf("kubeconfig secret does not contain 'value' key")
	}

	return string(kubeconfig), nil
}

// IsClusterReady checks if a cluster is fully ready
func (c *Client) IsClusterReady(ctx context.Context, namespace, name string) (bool, error) {
	cluster, err := c.GetCluster(ctx, namespace, name)
	if err != nil {
		return false, err
	}

	return conditions.IsTrue(cluster, clusterv1.ReadyCondition), nil
}

// WaitForClusterReady waits for a cluster to become ready
// This is a simplified version - in production you'd want proper timeout handling
func (c *Client) WaitForClusterReady(ctx context.Context, namespace, name string) error {
	// This would typically use a wait.Poll or watch mechanism
	// For now, just check once
	ready, err := c.IsClusterReady(ctx, namespace, name)
	if err != nil {
		return err
	}
	if !ready {
		return fmt.Errorf("cluster %s/%s is not ready", namespace, name)
	}
	return nil
}

// GetMachinePhase returns a human-readable phase for a machine
func GetMachinePhase(machine *clusterv1.Machine) string {
	if machine.Status.Phase != "" {
		return string(machine.Status.Phase)
	}

	// Check conditions
	if conditions.IsTrue(machine, clusterv1.ReadyCondition) {
		return "Running"
	}

	return "Unknown"
}

// GetControlPlaneStatus returns the status of a KubeadmControlPlane
func GetControlPlaneStatus(kcp *controlplanev1.KubeadmControlPlane) string {
	if kcp.Status.Ready {
		return "Ready"
	}

	if kcp.Status.UnavailableReplicas > 0 {
		return fmt.Sprintf("Degraded (%d unavailable)", kcp.Status.UnavailableReplicas)
	}

	if kcp.Status.Replicas == 0 {
		return "Not Initialized"
	}

	return "Updating"
}

// FormatClusterInfo formats cluster information for display
func FormatClusterInfo(status *ClusterStatus) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Cluster: %s/%s\n", status.Namespace, status.Name))
	sb.WriteString(fmt.Sprintf("Phase: %s\n", status.Phase))
	sb.WriteString(fmt.Sprintf("Ready: %v\n", status.Ready))
	sb.WriteString(fmt.Sprintf("Provider: %s\n", status.Provider))
	sb.WriteString(fmt.Sprintf("Version: %s\n", status.Version))
	sb.WriteString(fmt.Sprintf("Machines: %d/%d ready\n", status.ReadyMachines, status.TotalMachines))

	if len(status.Conditions) > 0 {
		sb.WriteString("\nConditions:\n")
		for _, cond := range status.Conditions {
			sb.WriteString(fmt.Sprintf("  %s: %s", cond.Type, cond.Status))
			if cond.Reason != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", cond.Reason))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
