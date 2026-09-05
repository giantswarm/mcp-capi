package handlers

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
)

// ProviderClusterSummary is one entry of a provider-specific cluster list.
type ProviderClusterSummary struct {
	Name                string `json:"name"`
	Namespace           string `json:"namespace"`
	InfrastructureKind  string `json:"infrastructureKind"`
	Phase               string `json:"phase,omitempty"`
	InfrastructureReady bool   `json:"infrastructureReady"`
}

// ClusterNetwork carries the CIDR configuration of a cluster.
type ClusterNetwork struct {
	PodCIDRs     []string `json:"podCIDRs,omitempty"`
	ServiceCIDRs []string `json:"serviceCIDRs,omitempty"`
}

// ProviderClusterDetail is the result of a provider-specific get_cluster tool.
type ProviderClusterDetail struct {
	Name                string          `json:"name"`
	Namespace           string          `json:"namespace"`
	Phase               string          `json:"phase,omitempty"`
	InfrastructureReady bool            `json:"infrastructureReady"`
	ControlPlaneReady   bool            `json:"controlPlaneReady"`
	Infrastructure      ObjectRef       `json:"infrastructure"`
	Network             *ClusterNetwork `json:"network,omitempty"`
	Conditions          []Condition     `json:"conditions,omitempty"`
	Note                string          `json:"note"`
}

// hasInfrastructureKind reports whether the cluster's infrastructure reference
// is one of the given kinds.
func hasInfrastructureKind(cluster *clusterv1.Cluster, kinds ...string) bool {
	if cluster.Spec.InfrastructureRef == nil {
		return false
	}
	for _, kind := range kinds {
		if cluster.Spec.InfrastructureRef.Kind == kind {
			return true
		}
	}
	return false
}

// createProviderListClustersHandler lists the clusters whose infrastructure
// reference is one of the given kinds, as {items: [ProviderClusterSummary]}.
func createProviderListClustersHandler(serverCtx *ServerContext, kinds ...string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()
		namespace, _ := arguments["namespace"].(string)

		clusters, err := capiClient.ListClusters(ctx, namespace, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		items := make([]ProviderClusterSummary, 0, len(clusters.Items))
		for i := range clusters.Items {
			cluster := &clusters.Items[i]
			if !hasInfrastructureKind(cluster, kinds...) {
				continue
			}
			items = append(items, ProviderClusterSummary{
				Name:                cluster.Name,
				Namespace:           cluster.Namespace,
				InfrastructureKind:  cluster.Spec.InfrastructureRef.Kind,
				Phase:               string(cluster.Status.Phase),
				InfrastructureReady: cluster.Status.InfrastructureReady,
			})
		}

		return listResult(items)
	}
}

// createProviderGetClusterHandler returns one cluster's details after checking
// that its infrastructure reference is one of the given kinds. providerName is
// used in the mismatch error; note tells the caller where provider-specific
// infrastructure details live.
func createProviderGetClusterHandler(serverCtx *ServerContext, providerName, note string, kinds ...string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			return nil, fmt.Errorf("name argument is required")
		}

		cluster, err := capiClient.GetCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster: %w", err)
		}

		if !hasInfrastructureKind(cluster, kinds...) {
			return mcp.NewToolResultError(fmt.Sprintf("Cluster %s/%s is not a%s %s cluster", namespace, name, article(providerName), providerName)), nil
		}

		detail := ProviderClusterDetail{
			Name:                cluster.Name,
			Namespace:           cluster.Namespace,
			Phase:               string(cluster.Status.Phase),
			InfrastructureReady: cluster.Status.InfrastructureReady,
			ControlPlaneReady:   cluster.Status.ControlPlaneReady,
			Infrastructure:      *objectRef(cluster.Spec.InfrastructureRef),
			Conditions:          capiConditions(cluster.Status.Conditions),
			Note:                note,
		}
		if network := cluster.Spec.ClusterNetwork; network != nil {
			n := &ClusterNetwork{}
			if network.Pods != nil {
				n.PodCIDRs = network.Pods.CIDRBlocks
			}
			if network.Services != nil {
				n.ServiceCIDRs = network.Services.CIDRBlocks
			}
			if len(n.PodCIDRs) > 0 || len(n.ServiceCIDRs) > 0 {
				detail.Network = n
			}
		}

		return jsonResult(detail)
	}
}

// article returns "n" for provider names that take "an" (AWS, Azure), "" otherwise,
// so the mismatch error reads "is not an AWS cluster" / "is not a GCP cluster".
func article(providerName string) string {
	switch providerName {
	case "AWS", "Azure":
		return "n"
	default:
		return ""
	}
}
