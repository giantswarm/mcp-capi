package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// createCreateClusterHandler creates a handler for creating new CAPI clusters
func createCreateClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		// Required parameters
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			return nil, fmt.Errorf("name argument is required")
		}
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		provider, ok := arguments["provider"].(string)
		if !ok || provider == "" {
			return nil, fmt.Errorf("provider argument is required")
		}

		// Validate provider
		validProviders := []string{"aws", "azure", "gcp", "vsphere"}
		isValidProvider := false
		for _, vp := range validProviders {
			if provider == vp {
				isValidProvider = true
				break
			}
		}
		if !isValidProvider {
			return nil, fmt.Errorf("invalid provider %s. Must be one of: %s", provider, strings.Join(validProviders, ", "))
		}

		// Optional parameters with defaults
		kubernetesVersion, _ := arguments["kubernetes_version"].(string)
		if kubernetesVersion == "" {
			kubernetesVersion = "v1.29.0"
		}

		controlPlaneCount := int32(3)
		if cpCount, ok := arguments["control_plane_count"].(float64); ok {
			controlPlaneCount = int32(cpCount)
		}

		workerCount := int32(3)
		if wCount, ok := arguments["worker_count"].(float64); ok {
			workerCount = int32(wCount)
		}

		region, _ := arguments["region"].(string)
		instanceType, _ := arguments["instance_type"].(string)

		// Create cluster options
		opts := capi.CreateClusterOptions{
			Name:              name,
			Namespace:         namespace,
			InfraProvider:     provider,
			KubernetesVersion: kubernetesVersion,
			ControlPlaneCount: controlPlaneCount,
			WorkerCount:       workerCount,
			Region:            region,
			InstanceType:      instanceType,
		}

		// Create the cluster
		cluster, err := serverCtx.capiClient.CreateCluster(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster: %w", err)
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("✅ Cluster '%s' creation initiated successfully!\n\n", name))
		content.WriteString("Cluster Details:\n")
		content.WriteString(fmt.Sprintf("  Name: %s\n", cluster.Name))
		content.WriteString(fmt.Sprintf("  Namespace: %s\n", cluster.Namespace))
		content.WriteString(fmt.Sprintf("  Provider: %s\n", provider))
		content.WriteString(fmt.Sprintf("  Kubernetes Version: %s\n", kubernetesVersion))
		content.WriteString(fmt.Sprintf("  Control Plane Nodes: %d\n", controlPlaneCount))
		content.WriteString(fmt.Sprintf("  Worker Nodes: %d\n", workerCount))
		if region != "" {
			content.WriteString(fmt.Sprintf("  Region: %s\n", region))
		}
		if instanceType != "" {
			content.WriteString(fmt.Sprintf("  Instance Type: %s\n", instanceType))
		}
		content.WriteString("\n⚠️  Note: This is a basic implementation that creates only the Cluster resource.\n")
		content.WriteString("In a production setup, you would need to:\n")
		content.WriteString("1. Create the infrastructure-specific cluster resource (e.g., AWSCluster)\n")
		content.WriteString("2. Create the control plane (e.g., KubeadmControlPlane)\n")
		content.WriteString("3. Create machine deployments for worker nodes\n")
		content.WriteString("4. Configure networking, storage, and other cluster settings\n\n")
		content.WriteString("Monitor cluster creation with: capi_cluster_status\n")

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

// createClusterHealthHandler creates a handler for checking cluster health
func createClusterHealthHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		health, err := serverCtx.capiClient.GetClusterHealth(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster health: %w", err)
		}

		var content strings.Builder
		
		// Overall status
		if health.Healthy {
			content.WriteString(fmt.Sprintf("✅ Cluster %s/%s is HEALTHY\n\n", namespace, name))
		} else {
			content.WriteString(fmt.Sprintf("❌ Cluster %s/%s is UNHEALTHY\n\n", namespace, name))
		}
		
		// Component status
		content.WriteString("Component Status:\n")
		content.WriteString(fmt.Sprintf("  • Control Plane: %s\n", formatHealthStatus(health.ControlPlaneReady)))
		content.WriteString(fmt.Sprintf("  • Infrastructure: %s\n", formatHealthStatus(health.InfraReady)))
		content.WriteString(fmt.Sprintf("  • Worker Nodes: %s\n", formatHealthStatus(health.WorkersReady)))
		
		// Issues
		if len(health.Issues) > 0 {
			content.WriteString("\n🔴 Issues:\n")
			for _, issue := range health.Issues {
				content.WriteString(fmt.Sprintf("  • %s\n", issue))
			}
		}
		
		// Warnings
		if len(health.Warnings) > 0 {
			content.WriteString("\n⚠️  Warnings:\n")
			for _, warning := range health.Warnings {
				content.WriteString(fmt.Sprintf("  • %s\n", warning))
			}
		}
		
		// Recommendations
		if !health.Healthy {
			content.WriteString("\n📋 Recommendations:\n")
			if !health.ControlPlaneReady {
				content.WriteString("  • Check control plane pods and logs\n")
				content.WriteString("  • Verify API server connectivity\n")
			}
			if !health.InfraReady {
				content.WriteString("  • Check infrastructure provider status\n")
				content.WriteString("  • Verify cloud resources are provisioned\n")
			}
			if !health.WorkersReady {
				content.WriteString("  • Check machine status with 'capi_list_machines'\n")
				content.WriteString("  • Review machine deployment events\n")
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

// formatHealthStatus returns a formatted string for component health status
func formatHealthStatus(ready bool) string {
	if ready {
		return "✅ Ready"
	}
	return "❌ Not Ready"
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
