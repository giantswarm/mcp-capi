package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	v1 "k8s.io/api/core/v1"
)

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

// createDeleteMachineHandler creates a handler for deleting CAPI machines
func createDeleteMachineHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Delete the machine
		err := serverCtx.capiClient.DeleteMachine(ctx, capi.DeleteMachineOptions{
			Namespace: namespace,
			Name:      name,
			Force:     force,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete machine: %v", err)), nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("✅ Successfully initiated deletion of machine %s/%s\n\n", namespace, name))
		content.WriteString("Note: Machine deletion is asynchronous. The machine will be:\n")
		content.WriteString("1. Drained (if it has a node)\n")
		content.WriteString("2. Removed from the cluster\n")
		content.WriteString("3. Infrastructure resources cleaned up\n\n")
		content.WriteString("Monitor deletion progress with:\n")
		content.WriteString(fmt.Sprintf("  capi_get_machine --namespace %s --name %s\n", namespace, name))

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

// createRemediateMachineHandler creates a handler for triggering machine remediation
func createRemediateMachineHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Get current machine status first
		machine, err := serverCtx.capiClient.GetMachine(ctx, namespace, name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get machine: %v", err)), nil
		}

		// Trigger remediation
		err = serverCtx.capiClient.RemediateMachine(ctx, capi.RemediateMachineOptions{
			Namespace: namespace,
			Name:      name,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to remediate machine: %v", err)), nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("🔧 Triggered remediation for machine %s/%s\n\n", namespace, name))
		content.WriteString("Current Machine Status:\n")
		content.WriteString(fmt.Sprintf("  • Phase: %s\n", machine.Status.Phase))
		if machine.Status.NodeRef != nil {
			content.WriteString(fmt.Sprintf("  • Node: %s\n", machine.Status.NodeRef.Name))
		}
		content.WriteString("\nRemediation Process:\n")
		content.WriteString("1. Machine will be marked for remediation\n")
		content.WriteString("2. MachineHealthCheck controller will process the remediation\n")
		content.WriteString("3. Depending on remediation strategy:\n")
		content.WriteString("   - Machine may be deleted and recreated\n")
		content.WriteString("   - Node may be rebooted\n")
		content.WriteString("   - Custom remediation may be applied\n\n")
		content.WriteString("Monitor remediation progress with:\n")
		content.WriteString(fmt.Sprintf("  capi_get_machine --namespace %s --name %s\n", namespace, name))

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

// createCreateMachineDeploymentHandler creates a handler for creating new machine deployments
func createCreateMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
		clusterName, ok := arguments["cluster_name"].(string)
		if !ok || clusterName == "" {
			return nil, fmt.Errorf("cluster_name argument is required")
		}

		// Get replicas
		replicas := int32(1)
		if r, ok := arguments["replicas"].(float64); ok {
			replicas = int32(r)
		}

		// Get infrastructure reference
		infraKind, _ := arguments["infra_kind"].(string)
		infraName, _ := arguments["infra_name"].(string)
		infraAPIVersion, _ := arguments["infra_api_version"].(string)

		if infraKind == "" || infraName == "" {
			return mcp.NewToolResultError("infra_kind and infra_name are required"), nil
		}

		// Get bootstrap reference
		bootstrapKind, _ := arguments["bootstrap_kind"].(string)
		bootstrapName, _ := arguments["bootstrap_name"].(string)
		bootstrapAPIVersion, _ := arguments["bootstrap_api_version"].(string)

		if bootstrapKind == "" || bootstrapName == "" {
			return mcp.NewToolResultError("bootstrap_kind and bootstrap_name are required"), nil
		}

		version, _ := arguments["version"].(string)
		if version == "" {
			version = "v1.29.0" // Default version
		}

		// Create the machine deployment
		md, err := serverCtx.capiClient.CreateMachineDeployment(ctx, capi.CreateMachineDeploymentOptions{
			Namespace:   namespace,
			Name:        name,
			ClusterName: clusterName,
			Replicas:    replicas,
			InfrastructureRef: v1.ObjectReference{
				Kind:       infraKind,
				Name:       infraName,
				APIVersion: infraAPIVersion,
			},
			BootstrapConfigRef: v1.ObjectReference{
				Kind:       bootstrapKind,
				Name:       bootstrapName,
				APIVersion: bootstrapAPIVersion,
			},
			Version: version,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create machine deployment: %v", err)), nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("✅ Successfully created machine deployment %s/%s\n\n", namespace, name))
		content.WriteString("Configuration:\n")
		content.WriteString(fmt.Sprintf("  • Cluster: %s\n", clusterName))
		content.WriteString(fmt.Sprintf("  • Replicas: %d\n", replicas))
		content.WriteString(fmt.Sprintf("  • Version: %s\n", version))
		content.WriteString(fmt.Sprintf("  • Infrastructure: %s/%s\n", infraKind, infraName))
		content.WriteString(fmt.Sprintf("  • Bootstrap: %s/%s\n", bootstrapKind, bootstrapName))
		if md.Spec.MinReadySeconds != nil {
			content.WriteString(fmt.Sprintf("  • Min Ready Seconds: %d\n", *md.Spec.MinReadySeconds))
		}
		content.WriteString("\nNote: Before creating a MachineDeployment, ensure you have:\n")
		content.WriteString("1. Created the infrastructure template (e.g., AWSMachineTemplate)\n")
		content.WriteString("2. Created the bootstrap config template (e.g., KubeadmConfigTemplate)\n\n")
		content.WriteString("Monitor the deployment with:\n")
		content.WriteString(fmt.Sprintf("  capi_list_machines --cluster %s\n", clusterName))
		content.WriteString("\nScale the deployment with:\n")
		content.WriteString(fmt.Sprintf("  capi_scale_machinedeployment --namespace %s --name %s --replicas <count>\n", namespace, name))

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

// createScaleMachineDeploymentHandler creates a handler for scaling machine deployments
func createScaleMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		replicasFloat, ok := arguments["replicas"].(float64)
		if !ok {
			return nil, fmt.Errorf("replicas argument is required")
		}
		replicas := int32(replicasFloat)

		// Get current state
		list, err := serverCtx.capiClient.ListMachineDeployments(ctx, namespace, "")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get machine deployment: %v", err)), nil
		}

		var currentReplicas int32
		found := false
		for _, md := range list.Items {
			if md.Name == name {
				if md.Spec.Replicas != nil {
					currentReplicas = *md.Spec.Replicas
				}
				found = true
				break
			}
		}

		if !found {
			return mcp.NewToolResultError(fmt.Sprintf("Machine deployment %s/%s not found", namespace, name)), nil
		}

		// Scale the machine deployment
		err = serverCtx.capiClient.ScaleMachineDeployment(ctx, namespace, name, replicas)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to scale machine deployment: %v", err)), nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("✅ Successfully scaled machine deployment %s/%s\n\n", namespace, name))
		content.WriteString("Scaling Operation:\n")
		content.WriteString(fmt.Sprintf("  • Previous Replicas: %d\n", currentReplicas))
		content.WriteString(fmt.Sprintf("  • New Replicas: %d\n", replicas))

		if replicas > currentReplicas {
			content.WriteString(fmt.Sprintf("  • Action: Scaling UP by %d nodes\n", replicas-currentReplicas))
			content.WriteString("\nNew nodes will be:\n")
			content.WriteString("1. Provisioned by the infrastructure provider\n")
			content.WriteString("2. Bootstrapped with Kubernetes\n")
			content.WriteString("3. Joined to the cluster\n")
		} else if replicas < currentReplicas {
			content.WriteString(fmt.Sprintf("  • Action: Scaling DOWN by %d nodes\n", currentReplicas-replicas))
			content.WriteString("\nNodes will be:\n")
			content.WriteString("1. Cordoned to prevent new workloads\n")
			content.WriteString("2. Drained to move existing workloads\n")
			content.WriteString("3. Removed from the cluster\n")
			content.WriteString("4. Infrastructure resources cleaned up\n")
		} else {
			content.WriteString("  • Action: No change (same replica count)\n")
		}

		content.WriteString("\nMonitor scaling progress with:\n")
		content.WriteString(fmt.Sprintf("  capi_list_machines --namespace %s\n", namespace))

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
