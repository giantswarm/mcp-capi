package capi

import (
	"context"
	"fmt"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PageOptions controls server-side pagination of CAPI list calls. Both
// fields are forwarded to controller-runtime as client.Limit / client.Continue.
type PageOptions struct {
	// Cursor is the continue token returned from a previous page (empty
	// for the first page).
	Cursor string
	// Limit is the maximum number of items the API server may return for
	// this page. Zero leaves the API server's default in place.
	Limit int64
}

// ClusterListItem is the per-cluster digest returned by paginated list
// tools. Smaller than a full Cluster object — fits many entries under
// the toolkit's response cap. The full object is available via
// capi_get_cluster.
type ClusterListItem struct {
	Name               string            `json:"name"`
	Namespace          string            `json:"namespace"`
	Phase              string            `json:"phase,omitempty"`
	Ready              bool              `json:"ready"`
	ControlPlaneReady  bool              `json:"controlPlaneReady"`
	InfraReady         bool              `json:"infraReady"`
	Paused             bool              `json:"paused,omitempty"`
	Version            string            `json:"version,omitempty"`
	Provider           string            `json:"provider,omitempty"`
	InfrastructureKind string            `json:"infrastructureKind,omitempty"`
	TotalMachines      int               `json:"totalMachines"`
	ReadyMachines      int               `json:"readyMachines"`
	Labels             map[string]string `json:"labels,omitempty"`
}

// MachineListItem is the per-machine digest.
type MachineListItem struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	ClusterName string `json:"clusterName,omitempty"`
	Phase       string `json:"phase,omitempty"`
	NodeName    string `json:"nodeName,omitempty"`
	ProviderID  string `json:"providerID,omitempty"`
	Version     string `json:"version,omitempty"`
	Ready       bool   `json:"ready"`
}

// MachineDeploymentListItem is the per-MD digest.
type MachineDeploymentListItem struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	ClusterName       string `json:"clusterName,omitempty"`
	Phase             string `json:"phase,omitempty"`
	Replicas          int32  `json:"replicas"`
	ReadyReplicas     int32  `json:"readyReplicas"`
	UpdatedReplicas   int32  `json:"updatedReplicas"`
	AvailableReplicas int32  `json:"availableReplicas"`
	Version           string `json:"version,omitempty"`
}

// MachineSetListItem is the per-MS digest.
type MachineSetListItem struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	ClusterName       string `json:"clusterName,omitempty"`
	Replicas          int32  `json:"replicas"`
	ReadyReplicas     int32  `json:"readyReplicas"`
	AvailableReplicas int32  `json:"availableReplicas"`
	OwnerKind         string `json:"ownerKind,omitempty"`
	OwnerName         string `json:"ownerName,omitempty"`
	InfraKind         string `json:"infrastructureKind,omitempty"`
	InfraName         string `json:"infrastructureName,omitempty"`
}

// ListClustersPage fetches one page of clusters. Returned cursor is the
// continue token to pass to the next call (empty when no more pages).
func (c *Client) ListClustersPage(ctx context.Context, namespace string, labelSelector map[string]string, page PageOptions) (*clusterv1.ClusterList, string, error) {
	out := &clusterv1.ClusterList{}
	opts := buildListOpts(namespace, page)
	if len(labelSelector) > 0 {
		opts = append(opts, client.MatchingLabels(labelSelector))
	}
	if err := c.ctrlClient.List(ctx, out, opts...); err != nil {
		return nil, "", fmt.Errorf("failed to list clusters: %w", err)
	}
	return out, out.GetContinue(), nil
}

// ListMachinesPage fetches one page of machines.
func (c *Client) ListMachinesPage(ctx context.Context, namespace, clusterName string, page PageOptions) (*clusterv1.MachineList, string, error) {
	out := &clusterv1.MachineList{}
	opts := buildListOpts(namespace, page)
	if clusterName != "" {
		opts = append(opts, client.MatchingLabels{clusterv1.ClusterNameLabel: clusterName})
	}
	if err := c.ctrlClient.List(ctx, out, opts...); err != nil {
		return nil, "", fmt.Errorf("failed to list machines: %w", err)
	}
	return out, out.GetContinue(), nil
}

// ListMachineDeploymentsPage fetches one page of machine deployments.
func (c *Client) ListMachineDeploymentsPage(ctx context.Context, namespace, clusterName string, page PageOptions) (*clusterv1.MachineDeploymentList, string, error) {
	out := &clusterv1.MachineDeploymentList{}
	opts := buildListOpts(namespace, page)
	if clusterName != "" {
		opts = append(opts, client.MatchingLabels{clusterv1.ClusterNameLabel: clusterName})
	}
	if err := c.ctrlClient.List(ctx, out, opts...); err != nil {
		return nil, "", fmt.Errorf("failed to list machine deployments: %w", err)
	}
	return out, out.GetContinue(), nil
}

// ListMachineSetsPage fetches one page of machine sets.
func (c *Client) ListMachineSetsPage(ctx context.Context, namespace, clusterName string, page PageOptions) (*clusterv1.MachineSetList, string, error) {
	out := &clusterv1.MachineSetList{}
	opts := buildListOpts(namespace, page)
	if clusterName != "" {
		opts = append(opts, client.MatchingLabels{clusterv1.ClusterNameLabel: clusterName})
	}
	if err := c.ctrlClient.List(ctx, out, opts...); err != nil {
		return nil, "", fmt.Errorf("failed to list machine sets: %w", err)
	}
	return out, out.GetContinue(), nil
}

func buildListOpts(namespace string, page PageOptions) []client.ListOption {
	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if page.Limit > 0 {
		opts = append(opts, client.Limit(page.Limit))
	}
	if page.Cursor != "" {
		opts = append(opts, client.Continue(page.Cursor))
	}
	return opts
}

// SummarizeCluster builds a ClusterListItem from a Cluster + its machines.
// machines may be nil; counts default to zero.
func SummarizeCluster(cluster *clusterv1.Cluster, machines []clusterv1.Machine, provider Provider, version string) ClusterListItem {
	item := ClusterListItem{
		Name:              cluster.Name,
		Namespace:         cluster.Namespace,
		Phase:             string(cluster.Status.Phase),
		Ready:             isConditionTrue(cluster.Status.Conditions, clusterv1.ReadyCondition),
		ControlPlaneReady: cluster.Status.ControlPlaneReady,
		InfraReady:        cluster.Status.InfrastructureReady,
		Paused:            cluster.Spec.Paused,
		Version:           version,
		Provider:          string(provider),
		Labels:            filterUserLabels(cluster.Labels),
	}
	if cluster.Spec.InfrastructureRef != nil {
		item.InfrastructureKind = cluster.Spec.InfrastructureRef.Kind
	}
	for _, m := range machines {
		item.TotalMachines++
		if m.Status.NodeRef != nil {
			item.ReadyMachines++
		}
	}
	return item
}

// SummarizeMachine builds a MachineListItem from a Machine.
func SummarizeMachine(m *clusterv1.Machine) MachineListItem {
	item := MachineListItem{
		Name:        m.Name,
		Namespace:   m.Namespace,
		ClusterName: m.Spec.ClusterName,
		Phase:       m.Status.Phase,
	}
	if m.Status.NodeRef != nil {
		item.NodeName = m.Status.NodeRef.Name
	}
	if m.Spec.ProviderID != nil {
		item.ProviderID = *m.Spec.ProviderID
	}
	if m.Spec.Version != nil {
		item.Version = *m.Spec.Version
	}
	for _, cond := range m.Status.Conditions {
		if cond.Type == clusterv1.ReadyCondition && cond.Status == "True" {
			item.Ready = true
			break
		}
	}
	return item
}

// SummarizeMachineDeployment builds a MachineDeploymentListItem.
func SummarizeMachineDeployment(md *clusterv1.MachineDeployment) MachineDeploymentListItem {
	item := MachineDeploymentListItem{
		Name:              md.Name,
		Namespace:         md.Namespace,
		ClusterName:       md.Spec.ClusterName,
		Phase:             md.Status.Phase,
		ReadyReplicas:     md.Status.ReadyReplicas,
		UpdatedReplicas:   md.Status.UpdatedReplicas,
		AvailableReplicas: md.Status.AvailableReplicas,
	}
	if md.Spec.Replicas != nil {
		item.Replicas = *md.Spec.Replicas
	}
	if md.Spec.Template.Spec.Version != nil {
		item.Version = *md.Spec.Template.Spec.Version
	}
	return item
}

// SummarizeMachineSet builds a MachineSetListItem.
func SummarizeMachineSet(ms *clusterv1.MachineSet) MachineSetListItem {
	item := MachineSetListItem{
		Name:              ms.Name,
		Namespace:         ms.Namespace,
		ClusterName:       ms.Spec.ClusterName,
		ReadyReplicas:     ms.Status.ReadyReplicas,
		AvailableReplicas: ms.Status.AvailableReplicas,
	}
	if ms.Spec.Replicas != nil {
		item.Replicas = *ms.Spec.Replicas
	}
	for _, owner := range ms.OwnerReferences {
		if owner.Kind == "MachineDeployment" {
			item.OwnerKind = owner.Kind
			item.OwnerName = owner.Name
			break
		}
	}
	if ms.Spec.Template.Spec.InfrastructureRef.Name != "" {
		item.InfraKind = ms.Spec.Template.Spec.InfrastructureRef.Kind
		item.InfraName = ms.Spec.Template.Spec.InfrastructureRef.Name
	}
	return item
}
