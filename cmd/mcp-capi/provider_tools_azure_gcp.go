package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Azure Provider Tools

// createAzureListClustersHandler lists Azure clusters
func createAzureListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, _ := arguments["namespace"].(string)

		// List all clusters
		clusters, err := serverCtx.capiClient.ListClusters(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		var content strings.Builder
		content.WriteString("Azure Clusters:\n\n")

		azureClusterCount := 0
		for _, cluster := range clusters.Items {
			// Check if this is an Azure cluster
			if cluster.Spec.InfrastructureRef != nil &&
				(cluster.Spec.InfrastructureRef.Kind == "AzureCluster" ||
					cluster.Spec.InfrastructureRef.Kind == "AzureManagedCluster") {
				azureClusterCount++

				content.WriteString(fmt.Sprintf("Cluster: %s/%s\n", cluster.Namespace, cluster.Name))
				content.WriteString(fmt.Sprintf("  Infrastructure: %s\n", cluster.Spec.InfrastructureRef.Kind))
				content.WriteString(fmt.Sprintf("  Phase: %s\n", cluster.Status.Phase))
				content.WriteString(fmt.Sprintf("  Ready: %v\n", cluster.Status.InfrastructureReady))

				// Try to get provider information
				provider, _ := serverCtx.capiClient.GetProviderForCluster(ctx, cluster.Namespace, cluster.Name)
				if provider == capi.ProviderAzure {
					content.WriteString("  Provider: Azure (confirmed)\n")
				}

				content.WriteString("\n")
			}
		}

		if azureClusterCount == 0 {
			content.WriteString("No Azure clusters found.\n")
		} else {
			content.WriteString(fmt.Sprintf("Total Azure clusters: %d\n", azureClusterCount))
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

// createAzureGetClusterHandler gets details of an Azure cluster
func createAzureGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Verify it's an Azure cluster
		if cluster.Spec.InfrastructureRef == nil ||
			(cluster.Spec.InfrastructureRef.Kind != "AzureCluster" &&
				cluster.Spec.InfrastructureRef.Kind != "AzureManagedCluster") {
			return mcp.NewToolResultError(fmt.Sprintf("Cluster %s/%s is not an Azure cluster", namespace, name)), nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("Azure Cluster: %s/%s\n\n", namespace, name))

		// Basic cluster info
		content.WriteString("Cluster Information:\n")
		content.WriteString(fmt.Sprintf("  Phase: %s\n", cluster.Status.Phase))
		content.WriteString(fmt.Sprintf("  Infrastructure Ready: %v\n", cluster.Status.InfrastructureReady))
		content.WriteString(fmt.Sprintf("  Control Plane Ready: %v\n", cluster.Status.ControlPlaneReady))

		// Infrastructure reference
		content.WriteString("\nInfrastructure:\n")
		content.WriteString(fmt.Sprintf("  Kind: %s\n", cluster.Spec.InfrastructureRef.Kind))
		content.WriteString(fmt.Sprintf("  Name: %s\n", cluster.Spec.InfrastructureRef.Name))

		content.WriteString("\nNote: For detailed Azure infrastructure information (resource group, vnet, etc.),\n")
		content.WriteString("you would need to query the AzureCluster resource directly.\n")

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

// createAzureManageResourceGroupHandler manages Azure resource groups
func createAzureManageResourceGroupHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var content strings.Builder
		content.WriteString("Azure Resource Group Management (Placeholder)\n\n")
		content.WriteString("This tool would manage Azure resource groups for CAPI clusters.\n")
		content.WriteString("Operations would include:\n")
		content.WriteString("- Creating resource groups\n")
		content.WriteString("- Setting resource group tags\n")
		content.WriteString("- Managing resource group policies\n")
		content.WriteString("- Listing resources in a group\n\n")
		content.WriteString("Note: CAPI typically creates its own resource groups,\n")
		content.WriteString("but this tool would help with custom configurations.\n")

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

// createAzureNetworkConfigHandler configures Azure networking
func createAzureNetworkConfigHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var content strings.Builder
		content.WriteString("Azure Network Configuration (Placeholder)\n\n")
		content.WriteString("This tool would configure Azure networking for CAPI clusters.\n")
		content.WriteString("Operations would include:\n")
		content.WriteString("- Creating/updating VNets\n")
		content.WriteString("- Managing subnets\n")
		content.WriteString("- Configuring Network Security Groups\n")
		content.WriteString("- Setting up VNet peering\n")
		content.WriteString("- Managing load balancers\n\n")
		content.WriteString("Common configurations:\n")
		content.WriteString("- Custom subnet layouts\n")
		content.WriteString("- Private cluster endpoints\n")
		content.WriteString("- Multi-region networking\n")

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

// GCP Provider Tools

// createGCPListClustersHandler lists GCP clusters
func createGCPListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, _ := arguments["namespace"].(string)

		// List all clusters
		clusters, err := serverCtx.capiClient.ListClusters(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		var content strings.Builder
		content.WriteString("GCP Clusters:\n\n")

		gcpClusterCount := 0
		for _, cluster := range clusters.Items {
			// Check if this is a GCP cluster
			if cluster.Spec.InfrastructureRef != nil &&
				(cluster.Spec.InfrastructureRef.Kind == "GCPCluster" ||
					cluster.Spec.InfrastructureRef.Kind == "GCPManagedCluster") {
				gcpClusterCount++

				content.WriteString(fmt.Sprintf("Cluster: %s/%s\n", cluster.Namespace, cluster.Name))
				content.WriteString(fmt.Sprintf("  Infrastructure: %s\n", cluster.Spec.InfrastructureRef.Kind))
				content.WriteString(fmt.Sprintf("  Phase: %s\n", cluster.Status.Phase))
				content.WriteString(fmt.Sprintf("  Ready: %v\n", cluster.Status.InfrastructureReady))

				// Try to get provider information
				provider, _ := serverCtx.capiClient.GetProviderForCluster(ctx, cluster.Namespace, cluster.Name)
				if provider == capi.ProviderGCP {
					content.WriteString("  Provider: GCP (confirmed)\n")
				}

				content.WriteString("\n")
			}
		}

		if gcpClusterCount == 0 {
			content.WriteString("No GCP clusters found.\n")
		} else {
			content.WriteString(fmt.Sprintf("Total GCP clusters: %d\n", gcpClusterCount))
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

// createGCPGetClusterHandler gets details of a GCP cluster
func createGCPGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Verify it's a GCP cluster
		if cluster.Spec.InfrastructureRef == nil ||
			(cluster.Spec.InfrastructureRef.Kind != "GCPCluster" &&
				cluster.Spec.InfrastructureRef.Kind != "GCPManagedCluster") {
			return mcp.NewToolResultError(fmt.Sprintf("Cluster %s/%s is not a GCP cluster", namespace, name)), nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("GCP Cluster: %s/%s\n\n", namespace, name))

		// Basic cluster info
		content.WriteString("Cluster Information:\n")
		content.WriteString(fmt.Sprintf("  Phase: %s\n", cluster.Status.Phase))
		content.WriteString(fmt.Sprintf("  Infrastructure Ready: %v\n", cluster.Status.InfrastructureReady))
		content.WriteString(fmt.Sprintf("  Control Plane Ready: %v\n", cluster.Status.ControlPlaneReady))

		// Infrastructure reference
		content.WriteString("\nInfrastructure:\n")
		content.WriteString(fmt.Sprintf("  Kind: %s\n", cluster.Spec.InfrastructureRef.Kind))
		content.WriteString(fmt.Sprintf("  Name: %s\n", cluster.Spec.InfrastructureRef.Name))

		content.WriteString("\nNote: For detailed GCP infrastructure information (VPC, firewall rules, etc.),\n")
		content.WriteString("you would need to query the GCPCluster resource directly.\n")

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

// createGCPManageNetworkHandler manages GCP networks
func createGCPManageNetworkHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var content strings.Builder
		content.WriteString("GCP Network Management (Placeholder)\n\n")
		content.WriteString("This tool would manage GCP networks for CAPI clusters.\n")
		content.WriteString("Operations would include:\n")
		content.WriteString("- Creating/updating VPC networks\n")
		content.WriteString("- Managing subnets\n")
		content.WriteString("- Configuring firewall rules\n")
		content.WriteString("- Setting up Cloud NAT\n")
		content.WriteString("- Managing load balancers\n\n")
		content.WriteString("GCP-specific features:\n")
		content.WriteString("- Shared VPC support\n")
		content.WriteString("- Private Google Access\n")
		content.WriteString("- Cloud Interconnect integration\n")

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
