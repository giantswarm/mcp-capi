package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// createListClustersHandler creates a handler for listing CAPI clusters
func createListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, _ := arguments["namespace"].(string)

		clusters, err := serverCtx.capiClient.ListClusters(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("Found %d clusters:\n\n", len(clusters.Items)))

		for _, cluster := range clusters.Items {
			status, _ := serverCtx.capiClient.GetClusterStatus(ctx, cluster.Namespace, cluster.Name)
			if status != nil {
				content.WriteString(capi.FormatClusterInfo(status))
				content.WriteString("\n---\n\n")
			}
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

// createGetClusterHandler creates a handler for getting a specific cluster
func createGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		status, err := serverCtx.capiClient.GetClusterStatus(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster status: %w", err)
		}

		var content strings.Builder
		content.WriteString(capi.FormatClusterInfo(status))

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

// createClusterStatusHandler creates a handler for getting detailed cluster status
func createClusterStatusHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		status, err := serverCtx.capiClient.GetClusterStatus(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster status: %w", err)
		}

		var content strings.Builder
		content.WriteString(capi.FormatClusterInfo(status))

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

// createScaleClusterHandler creates a handler for scaling clusters
func createScaleClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
		target, ok := arguments["target"].(string)
		if !ok || target == "" {
			return nil, fmt.Errorf("target argument is required")
		}
		replicas, ok := arguments["replicas"].(float64)
		if !ok {
			return nil, fmt.Errorf("replicas argument is required and must be a number")
		}
		machineDeployment, _ := arguments["machineDeployment"].(string)

		err := serverCtx.capiClient.ScaleCluster(ctx, namespace, name, target, int(replicas), machineDeployment)
		if err != nil {
			return nil, fmt.Errorf("failed to scale cluster: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Cluster %s/%s scaled successfully", namespace, name),
				},
			},
		}, nil
	}
}

// createGetKubeconfigHandler creates a handler for retrieving cluster kubeconfig
func createGetKubeconfigHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		kubeconfig, err := serverCtx.capiClient.GetKubeconfig(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("Kubeconfig for cluster %s/%s:\n\n", namespace, name))
		content.WriteString("```yaml\n")
		content.WriteString(kubeconfig)
		content.WriteString("\n```\n\n")
		content.WriteString("To use this kubeconfig:\n")
		content.WriteString("1. Save the content between the ``` markers to a file (e.g., cluster-kubeconfig.yaml)\n")
		content.WriteString("2. Use it with kubectl: kubectl --kubeconfig=cluster-kubeconfig.yaml get nodes\n")

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

// createPauseClusterHandler creates a handler for pausing cluster reconciliation
func createPauseClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		err := serverCtx.capiClient.PauseCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to pause cluster: %w", err)
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("✅ Cluster %s/%s has been paused\n\n", namespace, name))
		content.WriteString("The cluster reconciliation has been stopped. This means:\n")
		content.WriteString("- CAPI controllers will not make any changes to the cluster\n")
		content.WriteString("- The cluster will not be updated or scaled automatically\n")
		content.WriteString("- Manual operations can be performed safely\n\n")
		content.WriteString("To resume normal operations, use the capi_resume_cluster tool.")

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

// createResumeClusterHandler creates a handler for resuming cluster reconciliation
func createResumeClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		err := serverCtx.capiClient.ResumeCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to resume cluster: %w", err)
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("✅ Cluster %s/%s has been resumed\n\n", namespace, name))
		content.WriteString("The cluster reconciliation has been restarted. This means:\n")
		content.WriteString("- CAPI controllers will now reconcile the cluster normally\n")
		content.WriteString("- Any pending updates or changes will be applied\n")
		content.WriteString("- Automatic scaling and updates are re-enabled\n\n")
		content.WriteString("The cluster is now under normal CAPI management.")

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

// createDeleteClusterHandler creates a handler for deleting a cluster
func createDeleteClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
		force, _ := arguments["force"].(bool)

		// Get cluster status first to show what will be deleted
		status, err := serverCtx.capiClient.GetClusterStatus(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster status: %w", err)
		}

		var content strings.Builder

		// Show cluster information
		content.WriteString("⚠️  WARNING: You are about to delete the following cluster:\n\n")
		content.WriteString(capi.FormatClusterInfo(status))
		content.WriteString("\n")

		// Safety checks if not forced
		if !force {
			if status.Ready {
				content.WriteString("❌ SAFETY CHECK FAILED: Cluster is currently in Ready state.\n")
				content.WriteString("   This cluster appears to be healthy and operational.\n")
				content.WriteString("   Use force=true to override this safety check.\n\n")
				content.WriteString("   Recommended actions before deletion:\n")
				content.WriteString("   1. Backup any important data\n")
				content.WriteString("   2. Migrate workloads to another cluster\n")
				content.WriteString("   3. Ensure this is the correct cluster\n")

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

		// Proceed with deletion
		err = serverCtx.capiClient.DeleteCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to delete cluster: %w", err)
		}

		content.WriteString(fmt.Sprintf("\n✅ Cluster %s/%s deletion initiated successfully.\n\n", namespace, name))
		content.WriteString("Note: The actual deletion process may take several minutes as:\n")
		content.WriteString("- All cluster resources are being cleaned up\n")
		content.WriteString("- Infrastructure resources are being deprovisioned\n")
		content.WriteString("- Finalizers are being processed\n\n")
		content.WriteString("You can monitor the deletion progress by listing clusters in this namespace.")

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
