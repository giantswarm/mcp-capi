package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// vSphere Provider Tools

// createVSphereListClustersHandler lists vSphere clusters
func createVSphereListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, _ := arguments["namespace"].(string)

		// List all clusters
		clusters, err := serverCtx.capiClient.ListClusters(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		var content strings.Builder
		content.WriteString("vSphere Clusters:\n\n")

		vsphereClusterCount := 0
		for _, cluster := range clusters.Items {
			// Check if this is a vSphere cluster
			if cluster.Spec.InfrastructureRef != nil &&
				cluster.Spec.InfrastructureRef.Kind == "VSphereCluster" {
				vsphereClusterCount++

				content.WriteString(fmt.Sprintf("Cluster: %s/%s\n", cluster.Namespace, cluster.Name))
				content.WriteString(fmt.Sprintf("  Infrastructure: %s\n", cluster.Spec.InfrastructureRef.Kind))
				content.WriteString(fmt.Sprintf("  Phase: %s\n", cluster.Status.Phase))
				content.WriteString(fmt.Sprintf("  Ready: %v\n", cluster.Status.InfrastructureReady))

				// Try to get provider information
				provider, _ := serverCtx.capiClient.GetProviderForCluster(ctx, cluster.Namespace, cluster.Name)
				if provider == capi.ProviderVSphere {
					content.WriteString("  Provider: vSphere (confirmed)\n")
				}

				content.WriteString("\n")
			}
		}

		if vsphereClusterCount == 0 {
			content.WriteString("No vSphere clusters found.\n")
		} else {
			content.WriteString(fmt.Sprintf("Total vSphere clusters: %d\n", vsphereClusterCount))
		}

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

// createVSphereGetClusterHandler gets details of a vSphere cluster
func createVSphereGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
		cluster, err := serverCtx.capiClient.GetCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster: %w", err)
		}

		// Verify it's a vSphere cluster
		if cluster.Spec.InfrastructureRef == nil ||
			cluster.Spec.InfrastructureRef.Kind != "VSphereCluster" {
			return mcp.NewToolResultError(fmt.Sprintf("Cluster %s/%s is not a vSphere cluster", namespace, name)), nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("vSphere Cluster: %s/%s\n\n", namespace, name))

		// Basic cluster info
		content.WriteString("Cluster Information:\n")
		content.WriteString(fmt.Sprintf("  Phase: %s\n", cluster.Status.Phase))
		content.WriteString(fmt.Sprintf("  Infrastructure Ready: %v\n", cluster.Status.InfrastructureReady))
		content.WriteString(fmt.Sprintf("  Control Plane Ready: %v\n", cluster.Status.ControlPlaneReady))

		// Infrastructure reference
		content.WriteString("\nInfrastructure:\n")
		content.WriteString(fmt.Sprintf("  Kind: %s\n", cluster.Spec.InfrastructureRef.Kind))
		content.WriteString(fmt.Sprintf("  Name: %s\n", cluster.Spec.InfrastructureRef.Name))

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

// createVSphereManageVMsHandler manages vSphere VMs
func createVSphereManageVMsHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
					Type: "text",
					Text: content.String(),
				},
			},
		}, nil
	}
}

// Helper function to filter clusters by provider
func filterClustersByProvider(clusters *clusterv1.ClusterList, providerKinds []string) []*clusterv1.Cluster {
	var filtered []*clusterv1.Cluster
	for i := range clusters.Items {
		cluster := &clusters.Items[i]
		if cluster.Spec.InfrastructureRef != nil {
			for _, kind := range providerKinds {
				if cluster.Spec.InfrastructureRef.Kind == kind {
					filtered = append(filtered, cluster)
					break
				}
			}
		}
	}
	return filtered
}
