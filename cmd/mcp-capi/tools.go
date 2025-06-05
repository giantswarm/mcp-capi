package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// testToolHandler handles the test tool
func testToolHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	message, ok := arguments["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message argument is required and must be a string")
	}

	response := fmt.Sprintf("Echo from CAPI MCP Server: %s", message)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: response,
			},
		},
	}, nil
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

// createListMachinesHandler creates a handler for listing CAPI machines
func createListMachinesHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		clusterName, _ := arguments["clusterName"].(string)

		machines, err := serverCtx.capiClient.ListMachines(ctx, namespace, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to list machines: %w", err)
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("Found %d machines", len(machines.Items)))
		if clusterName != "" {
			content.WriteString(fmt.Sprintf(" in cluster %s", clusterName))
		}
		content.WriteString(":\n\n")

		for _, machine := range machines.Items {
			content.WriteString(fmt.Sprintf("Machine: %s/%s\n", machine.Namespace, machine.Name))
			content.WriteString(fmt.Sprintf("  Cluster: %s\n", machine.Spec.ClusterName))
			if machine.Status.Phase != "" {
				content.WriteString(fmt.Sprintf("  Phase: %s\n", machine.Status.Phase))
			}
			if machine.Status.NodeRef != nil {
				content.WriteString(fmt.Sprintf("  Node: %s\n", machine.Status.NodeRef.Name))
			}
			if machine.Spec.ProviderID != nil {
				content.WriteString(fmt.Sprintf("  Provider ID: %s\n", *machine.Spec.ProviderID))
			}
			// Check if machine has Ready condition
			ready := false
			for _, condition := range machine.Status.Conditions {
				if condition.Type == "Ready" && condition.Status == "True" {
					ready = true
					break
				}
			}
			content.WriteString(fmt.Sprintf("  Ready: %v\n", ready))
			content.WriteString("\n")
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

// createListMachineDeploymentsHandler creates a handler for listing CAPI machine deployments
func createListMachineDeploymentsHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		clusterName, _ := arguments["clusterName"].(string)

		mds, err := serverCtx.capiClient.ListMachineDeployments(ctx, namespace, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to list machine deployments: %w", err)
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("Found %d machine deployments", len(mds.Items)))
		if clusterName != "" {
			content.WriteString(fmt.Sprintf(" in cluster %s", clusterName))
		}
		content.WriteString(":\n\n")

		for _, md := range mds.Items {
			content.WriteString(fmt.Sprintf("MachineDeployment: %s/%s\n", md.Namespace, md.Name))
			content.WriteString(fmt.Sprintf("  Cluster: %s\n", md.Spec.ClusterName))
			content.WriteString(fmt.Sprintf("  Replicas: %d\n", *md.Spec.Replicas))
			if md.Status.Replicas > 0 {
				content.WriteString(fmt.Sprintf("  Status: %d ready / %d updated / %d available\n",
					md.Status.ReadyReplicas,
					md.Status.UpdatedReplicas,
					md.Status.AvailableReplicas))
			}
			if md.Status.Phase != "" {
				content.WriteString(fmt.Sprintf("  Phase: %s\n", md.Status.Phase))
			}
			if md.Spec.Template.Spec.Version != nil {
				content.WriteString(fmt.Sprintf("  Kubernetes Version: %s\n", *md.Spec.Template.Spec.Version))
			}
			content.WriteString("\n")
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

// createGetMachineHandler creates a handler for getting detailed machine information
func createGetMachineHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		machine, err := serverCtx.capiClient.GetMachine(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get machine: %w", err)
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("Machine: %s/%s\n\n", machine.Namespace, machine.Name))

		// Basic information
		content.WriteString("Basic Information:\n")
		content.WriteString(fmt.Sprintf("  Cluster: %s\n", machine.Spec.ClusterName))
		if machine.Status.Phase != "" {
			content.WriteString(fmt.Sprintf("  Phase: %s\n", machine.Status.Phase))
		}
		if machine.Spec.Version != nil {
			content.WriteString(fmt.Sprintf("  Kubernetes Version: %s\n", *machine.Spec.Version))
		}
		if machine.Spec.ProviderID != nil {
			content.WriteString(fmt.Sprintf("  Provider ID: %s\n", *machine.Spec.ProviderID))
		}

		// Node information
		if machine.Status.NodeRef != nil {
			content.WriteString(fmt.Sprintf("\nNode Information:\n"))
			content.WriteString(fmt.Sprintf("  Node Name: %s\n", machine.Status.NodeRef.Name))
			content.WriteString(fmt.Sprintf("  Node UID: %s\n", machine.Status.NodeRef.UID))
		}

		// Bootstrap information
		if machine.Spec.Bootstrap.ConfigRef != nil {
			content.WriteString(fmt.Sprintf("\nBootstrap:\n"))
			content.WriteString(fmt.Sprintf("  Kind: %s\n", machine.Spec.Bootstrap.ConfigRef.Kind))
			content.WriteString(fmt.Sprintf("  Name: %s\n", machine.Spec.Bootstrap.ConfigRef.Name))
		}

		// Infrastructure information
		if machine.Spec.InfrastructureRef.Kind != "" {
			content.WriteString(fmt.Sprintf("\nInfrastructure:\n"))
			content.WriteString(fmt.Sprintf("  Kind: %s\n", machine.Spec.InfrastructureRef.Kind))
			content.WriteString(fmt.Sprintf("  Name: %s\n", machine.Spec.InfrastructureRef.Name))
		}

		// Conditions
		if len(machine.Status.Conditions) > 0 {
			content.WriteString("\nConditions:\n")
			for _, condition := range machine.Status.Conditions {
				content.WriteString(fmt.Sprintf("  - Type: %s\n", condition.Type))
				content.WriteString(fmt.Sprintf("    Status: %s\n", condition.Status))
				if condition.Reason != "" {
					content.WriteString(fmt.Sprintf("    Reason: %s\n", condition.Reason))
				}
				if condition.Message != "" {
					content.WriteString(fmt.Sprintf("    Message: %s\n", condition.Message))
				}
			}
		}

		// Addresses
		if len(machine.Status.Addresses) > 0 {
			content.WriteString("\nAddresses:\n")
			for _, addr := range machine.Status.Addresses {
				content.WriteString(fmt.Sprintf("  - Type: %s, Address: %s\n", addr.Type, addr.Address))
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
