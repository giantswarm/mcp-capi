package handlers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
)

// vSphere Provider Tools

// CreateVSphereListClustersHandler lists vSphere clusters
func CreateVSphereListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return buildListClustersHandler(serverCtx, providerListConfig{
		header:        "vSphere Clusters:\n\n",
		infraKinds:    []string{"VSphereCluster"},
		provider:      capi.ProviderVSphere,
		providerLabel: "vSphere",
		noneMsg:       "No vSphere clusters found.\n",
		totalFmt:      "Total vSphere clusters: %d\n",
	})
}

// CreateVSphereGetClusterHandler gets details of a vSphere cluster
func CreateVSphereGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, errors.New("namespace argument is required")
		}
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			return nil, errors.New("name argument is required")
		}

		// Get the cluster
		cluster, err := serverCtx.CAPIClient.GetCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster: %w", err)
		}

		// Verify it's a vSphere cluster
		if cluster.Spec.InfrastructureRef == nil ||
			cluster.Spec.InfrastructureRef.Kind != "VSphereCluster" {
			return mcp.NewToolResultError(fmt.Sprintf("Cluster %s/%s is not a vSphere cluster", namespace, name)), nil
		}

		var content strings.Builder
		fmt.Fprintf(&content, "vSphere Cluster: %s/%s\n\n", namespace, name)

		// Basic cluster info
		content.WriteString("Cluster Information:\n")
		fmt.Fprintf(&content, "  Phase: %s\n", cluster.Status.Phase)
		fmt.Fprintf(&content, "  Infrastructure Ready: %v\n", cluster.Status.InfrastructureReady)
		fmt.Fprintf(&content, "  Control Plane Ready: %v\n", cluster.Status.ControlPlaneReady)

		// Infrastructure reference
		content.WriteString("\nInfrastructure:\n")
		fmt.Fprintf(&content, "  Kind: %s\n", cluster.Spec.InfrastructureRef.Kind)
		fmt.Fprintf(&content, "  Name: %s\n", cluster.Spec.InfrastructureRef.Name)

		content.WriteString("\nNote: For detailed vSphere infrastructure information (datacenter, datastore, etc.),\n")
		content.WriteString("you would need to query the VSphereCluster resource directly.\n")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: content.String(),
				},
			},
		}, nil
	}
}

// CreateVSphereManageVMsHandler manages vSphere VMs (placeholder)
func CreateVSphereManageVMsHandler(_ *ServerContext) server.ToolHandlerFunc {
	return buildPlaceholderHandler("vSphere VM Management (Placeholder)\n\n" +
		"This tool would manage vSphere VMs for CAPI clusters.\n" +
		"Operations would include:\n" +
		"- Listing VMs in a cluster\n" +
		"- Power operations (on/off/restart)\n" +
		"- VM cloning from templates\n" +
		"- Resource allocation changes\n" +
		"- Snapshot management\n\n" +
		"vSphere-specific features:\n" +
		"- DRS rules configuration\n" +
		"- Storage vMotion\n" +
		"- VM folder organization\n")
}

// filterClustersByProvider filters clusters by infrastructure provider kinds.
func filterClustersByProvider(clusters *clusterv1.ClusterList, providerKinds []string) []*clusterv1.Cluster {
	filtered := make([]*clusterv1.Cluster, 0, len(clusters.Items))
	for i := range clusters.Items {
		cluster := &clusters.Items[i]
		if cluster.Spec.InfrastructureRef != nil &&
			slices.Contains(providerKinds, cluster.Spec.InfrastructureRef.Kind) {
			filtered = append(filtered, cluster)
		}
	}
	return filtered
}
