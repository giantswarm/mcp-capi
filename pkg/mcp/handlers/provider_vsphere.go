package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-capi/pkg/capi"
)

// vSphere Provider Tools

// createVSphereListClustersHandler lists vSphere clusters
func CreateVSphereListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, _ := arguments["namespace"].(string)

		clusters, err := serverCtx.CAPIClient.ListClusters(ctx, namespace, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		items := make([]capi.ClusterListItem, 0, len(clusters.Items))
		for i := range clusters.Items {
			cl := &clusters.Items[i]
			if cl.Spec.InfrastructureRef == nil || cl.Spec.InfrastructureRef.Kind != "VSphereCluster" {
				continue
			}
			provider, _ := serverCtx.CAPIClient.GetProviderForCluster(ctx, cl.Namespace, cl.Name)
			items = append(items, capi.SummarizeCluster(cl, nil, provider, ""))
		}
		return paginatedResult(items, "")
	}
}

// createVSphereGetClusterHandler gets details of a vSphere cluster
func CreateVSphereGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			return nil, fmt.Errorf("name argument is required")
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
					Type: textContentType,
					Text: content.String(),
				},
			},
		}, nil
	}
}

// createVSphereManageVMsHandler manages vSphere VMs
func CreateVSphereManageVMsHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var content strings.Builder
		content.WriteString("vSphere VM Management (Placeholder)\n\n")
		content.WriteString("This tool would manage vSphere VMs for CAPI clusters.\n")
		content.WriteString("Operations would include:\n")
		content.WriteString("- Listing VMs in a cluster\n")
		content.WriteString("- Power operations (on/off/restart)\n")
		content.WriteString("- VM cloning from templates\n")
		content.WriteString("- Resource allocation changes\n")
		content.WriteString("- Snapshot management\n\n")
		content.WriteString("vSphere-specific features:\n")
		content.WriteString("- DRS rules configuration\n")
		content.WriteString("- Storage vMotion\n")
		content.WriteString("- VM folder organization\n")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: textContentType,
					Text: content.String(),
				},
			},
		}, nil
	}
}
