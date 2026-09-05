package capi

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	controlplanev1 "sigs.k8s.io/cluster-api/api/controlplane/kubeadm/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"                      //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
)

const kindKubeadmControlPlane = "KubeadmControlPlane"

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
	Paused            bool
	Labels            map[string]string
	Annotations       map[string]string
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
		Ready:             isConditionTrue(cluster.Status.Conditions, clusterv1.ReadyCondition),
		ControlPlaneReady: cluster.Status.ControlPlaneReady,
		InfraReady:        cluster.Status.InfrastructureReady,
		Conditions:        cluster.Status.Conditions,
		Paused:            cluster.Spec.Paused,
		Labels:            cluster.Labels,
		Annotations:       cluster.Annotations,
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
		if cluster.Spec.ControlPlaneRef.Kind == kindKubeadmControlPlane {
			kcp, err := c.GetKubeadmControlPlane(ctx, namespace, cluster.Spec.ControlPlaneRef.Name)
			if err == nil && kcp.Spec.Version != "" {
				status.Version = kcp.Spec.Version
			}
		}
	}

	return status, nil
}

// GetClusterStatusFromList computes status for an already-fetched cluster using
// pre-fetched machines. This avoids the per-cluster Get and ListMachines calls
// that GetClusterStatus performs, making it suitable for bulk listing.
func (c *Client) GetClusterStatusFromList(ctx context.Context, cluster *clusterv1.Cluster, machines []clusterv1.Machine) (*ClusterStatus, error) {
	status := &ClusterStatus{
		Name:              cluster.Name,
		Namespace:         cluster.Namespace,
		Phase:             string(cluster.Status.Phase),
		Ready:             isConditionTrue(cluster.Status.Conditions, clusterv1.ReadyCondition),
		ControlPlaneReady: cluster.Status.ControlPlaneReady,
		InfraReady:        cluster.Status.InfrastructureReady,
		Conditions:        cluster.Status.Conditions,
		Paused:            cluster.Spec.Paused,
		Labels:            cluster.Labels,
		Annotations:       cluster.Annotations,
	}

	// Get version from cluster spec
	if cluster.Spec.Topology != nil && cluster.Spec.Topology.Version != "" {
		status.Version = cluster.Spec.Topology.Version
	}

	// Get provider information (pure function, no API call)
	status.Provider = DetermineProvider(cluster)

	// Count machines from pre-fetched slice
	status.TotalMachines = len(machines)
	for _, machine := range machines {
		if machine.Status.NodeRef != nil {
			status.ReadyMachines++
		}
	}

	// Get control plane version if available
	if cluster.Spec.ControlPlaneRef != nil && status.Version == "" {
		if cluster.Spec.ControlPlaneRef.Kind == kindKubeadmControlPlane {
			kcp, err := c.GetKubeadmControlPlane(ctx, cluster.Namespace, cluster.Spec.ControlPlaneRef.Name)
			if err == nil && kcp.Spec.Version != "" {
				status.Version = kcp.Spec.Version
			}
		}
	}

	return status, nil
}

// IsClusterReady checks if a cluster is fully ready
func (c *Client) IsClusterReady(ctx context.Context, namespace, name string) (bool, error) {
	cluster, err := c.GetCluster(ctx, namespace, name)
	if err != nil {
		return false, err
	}

	return isConditionTrue(cluster.Status.Conditions, clusterv1.ReadyCondition), nil
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
	if isConditionTrue(machine.Status.Conditions, clusterv1.ReadyCondition) {
		return "Running"
	}

	return "Unknown"
}

// GetControlPlaneStatus returns the status of a KubeadmControlPlane
func GetControlPlaneStatus(kcp *controlplanev1.KubeadmControlPlane) string {
	if kcp.Status.Ready {
		return "Ready"
	}

	if kcp.Status.UnavailableReplicas > 0 { //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
		return fmt.Sprintf("Degraded (%d unavailable)", kcp.Status.UnavailableReplicas) //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
	}

	if kcp.Status.Replicas == 0 {
		return "Not Initialized"
	}

	return "Updating"
}

// isConditionTrue checks if a condition with the given type has status True.
// This is a v1beta1-compatible helper that replaces the conditions.IsTrue utility
// which now requires v1beta2 types.
func isConditionTrue(conditions clusterv1.Conditions, conditionType clusterv1.ConditionType) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// UserLabels returns a copy of labels with internal CAPI labels removed.
// Internal labels (those under the cluster.x-k8s.io domain) are set by the
// system and are not useful for user-facing display.
func UserLabels(labels map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range labels {
		if !strings.Contains(k, "cluster.x-k8s.io") {
			result[k] = v
		}
	}
	return result
}
