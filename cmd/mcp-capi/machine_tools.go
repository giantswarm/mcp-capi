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

// createUpdateMachineDeploymentHandler creates a handler for updating machine deployment configuration
func createUpdateMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Parse optional parameters
		opts := capi.UpdateMachineDeploymentOptions{
			Namespace: namespace,
			Name:      name,
		}

		// Version update
		if version, ok := arguments["version"].(string); ok && version != "" {
			opts.Version = &version
		}

		// Replicas update
		if replicasFloat, ok := arguments["replicas"].(float64); ok {
			replicas := int32(replicasFloat)
			opts.Replicas = &replicas
		}

		// MinReadySeconds update
		if minReadyFloat, ok := arguments["min_ready_seconds"].(float64); ok {
			minReady := int32(minReadyFloat)
			opts.MinReadySeconds = &minReady
		}

		// Labels update
		if labels, ok := arguments["labels"].(map[string]interface{}); ok {
			opts.Labels = make(map[string]string)
			for k, v := range labels {
				if strVal, ok := v.(string); ok {
					opts.Labels[k] = strVal
				}
			}
		}

		// Annotations update
		if annotations, ok := arguments["annotations"].(map[string]interface{}); ok {
			opts.Annotations = make(map[string]string)
			for k, v := range annotations {
				if strVal, ok := v.(string); ok {
					opts.Annotations[k] = strVal
				}
			}
		}

		// Update the machine deployment
		md, err := serverCtx.capiClient.UpdateMachineDeployment(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update machine deployment: %v", err)), nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("✅ Successfully updated machine deployment %s/%s\n\n", namespace, name))
		content.WriteString("Updated Configuration:\n")

		if opts.Version != nil {
			content.WriteString(fmt.Sprintf("  • Version: %s\n", *opts.Version))
		}
		if opts.Replicas != nil {
			content.WriteString(fmt.Sprintf("  • Replicas: %d\n", *opts.Replicas))
		}
		if opts.MinReadySeconds != nil {
			content.WriteString(fmt.Sprintf("  • Min Ready Seconds: %d\n", *opts.MinReadySeconds))
		}
		if len(opts.Labels) > 0 {
			content.WriteString("  • Labels updated\n")
		}
		if len(opts.Annotations) > 0 {
			content.WriteString("  • Annotations updated\n")
		}

		content.WriteString("\nCurrent Status:\n")
		content.WriteString(fmt.Sprintf("  • Ready Replicas: %d\n", md.Status.ReadyReplicas))
		content.WriteString(fmt.Sprintf("  • Updated Replicas: %d\n", md.Status.UpdatedReplicas))
		content.WriteString(fmt.Sprintf("  • Available Replicas: %d\n", md.Status.AvailableReplicas))

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

// createRolloutMachineDeploymentHandler creates a handler for triggering machine deployment rollout
func createRolloutMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		reason, _ := arguments["reason"].(string)

		// Trigger the rollout
		err := serverCtx.capiClient.RolloutMachineDeployment(ctx, capi.RolloutMachineDeploymentOptions{
			Namespace: namespace,
			Name:      name,
			Reason:    reason,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to trigger rollout: %v", err)), nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("🔄 Successfully triggered rollout for machine deployment %s/%s\n\n", namespace, name))

		if reason != "" {
			content.WriteString(fmt.Sprintf("Reason: %s\n\n", reason))
		}

		content.WriteString("Rollout Process:\n")
		content.WriteString("1. New machines will be created with updated configuration\n")
		content.WriteString("2. Old machines will be gradually replaced\n")
		content.WriteString("3. The rollout respects the deployment's update strategy\n")
		content.WriteString("4. Health checks ensure machines are ready before proceeding\n\n")

		content.WriteString("Monitor rollout progress with:\n")
		content.WriteString(fmt.Sprintf("  capi_list_machines --namespace %s --cluster <cluster-name>\n", namespace))
		content.WriteString(fmt.Sprintf("  capi_list_machinedeployments --namespace %s\n", namespace))

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

// createListMachineSetsHandler creates a handler for listing machine sets
func createListMachineSetsHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		clusterName, _ := arguments["clusterName"].(string)

		machineSets, err := serverCtx.capiClient.ListMachineSets(ctx, namespace, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to list machine sets: %w", err)
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("Found %d machine sets", len(machineSets.Items)))
		if clusterName != "" {
			content.WriteString(fmt.Sprintf(" in cluster %s", clusterName))
		}
		content.WriteString(":\n\n")

		for _, ms := range machineSets.Items {
			content.WriteString(fmt.Sprintf("MachineSet: %s/%s\n", ms.Namespace, ms.Name))
			content.WriteString(fmt.Sprintf("  Cluster: %s\n", ms.Spec.ClusterName))
			if ms.Spec.Replicas != nil {
				content.WriteString(fmt.Sprintf("  Replicas: %d\n", *ms.Spec.Replicas))
			}
			content.WriteString(fmt.Sprintf("  Ready: %d/%d\n", ms.Status.ReadyReplicas, ms.Status.Replicas))
			content.WriteString(fmt.Sprintf("  Available: %d\n", ms.Status.AvailableReplicas))

			// Show owner reference (usually MachineDeployment)
			for _, owner := range ms.OwnerReferences {
				if owner.Kind == "MachineDeployment" {
					content.WriteString(fmt.Sprintf("  Owner: MachineDeployment/%s\n", owner.Name))
				}
			}

			// Show machine template
			if ms.Spec.Template.Spec.InfrastructureRef.Name != "" {
				content.WriteString(fmt.Sprintf("  Infrastructure: %s/%s\n",
					ms.Spec.Template.Spec.InfrastructureRef.Kind,
					ms.Spec.Template.Spec.InfrastructureRef.Name))
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

// createGetMachineSetHandler creates a handler for getting machine set details
func createGetMachineSetHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		ms, err := serverCtx.capiClient.GetMachineSet(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get machine set: %w", err)
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("MachineSet: %s/%s\n\n", ms.Namespace, ms.Name))

		// Basic information
		content.WriteString("Basic Information:\n")
		content.WriteString(fmt.Sprintf("  Cluster: %s\n", ms.Spec.ClusterName))
		if ms.Spec.Replicas != nil {
			content.WriteString(fmt.Sprintf("  Desired Replicas: %d\n", *ms.Spec.Replicas))
		}

		// Status
		content.WriteString("\nStatus:\n")
		content.WriteString(fmt.Sprintf("  Total Replicas: %d\n", ms.Status.Replicas))
		content.WriteString(fmt.Sprintf("  Ready Replicas: %d\n", ms.Status.ReadyReplicas))
		content.WriteString(fmt.Sprintf("  Available Replicas: %d\n", ms.Status.AvailableReplicas))
		if ms.Status.FailureReason != nil {
			content.WriteString(fmt.Sprintf("  Failure Reason: %s\n", *ms.Status.FailureReason))
		}
		if ms.Status.FailureMessage != nil {
			content.WriteString(fmt.Sprintf("  Failure Message: %s\n", *ms.Status.FailureMessage))
		}

		// Machine template
		content.WriteString("\nMachine Template:\n")
		if ms.Spec.Template.Spec.Version != nil {
			content.WriteString(fmt.Sprintf("  Kubernetes Version: %s\n", *ms.Spec.Template.Spec.Version))
		}
		if ms.Spec.Template.Spec.InfrastructureRef.Name != "" {
			content.WriteString(fmt.Sprintf("  Infrastructure: %s/%s\n",
				ms.Spec.Template.Spec.InfrastructureRef.Kind,
				ms.Spec.Template.Spec.InfrastructureRef.Name))
		}
		if ms.Spec.Template.Spec.Bootstrap.ConfigRef != nil {
			content.WriteString(fmt.Sprintf("  Bootstrap: %s/%s\n",
				ms.Spec.Template.Spec.Bootstrap.ConfigRef.Kind,
				ms.Spec.Template.Spec.Bootstrap.ConfigRef.Name))
		}

		// Owner references
		if len(ms.OwnerReferences) > 0 {
			content.WriteString("\nOwners:\n")
			for _, owner := range ms.OwnerReferences {
				content.WriteString(fmt.Sprintf("  - %s: %s\n", owner.Kind, owner.Name))
			}
		}

		// Conditions
		if len(ms.Status.Conditions) > 0 {
			content.WriteString("\nConditions:\n")
			for _, condition := range ms.Status.Conditions {
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

// createDrainNodeHandler creates a handler for draining nodes
func createDrainNodeHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		// Build options
		opts := capi.NodeOperationOptions{}

		// Either namespace+machineName or nodeName is required
		namespace, _ := arguments["namespace"].(string)
		machineName, _ := arguments["machine_name"].(string)
		nodeName, _ := arguments["node_name"].(string)

		if nodeName == "" && (namespace == "" || machineName == "") {
			return nil, fmt.Errorf("either node_name or (namespace and machine_name) must be provided")
		}

		opts.Namespace = namespace
		opts.MachineName = machineName
		opts.NodeName = nodeName

		// Optional parameters
		opts.IgnoreDaemonSets, _ = arguments["ignore_daemonsets"].(bool)
		opts.DeleteLocalData, _ = arguments["delete_local_data"].(bool)
		opts.Force, _ = arguments["force"].(bool)

		if gracePeriodFloat, ok := arguments["grace_period_seconds"].(float64); ok {
			gracePeriod := int32(gracePeriodFloat)
			opts.GracePeriodSeconds = &gracePeriod
		}

		// Drain the node
		err := serverCtx.capiClient.DrainNode(ctx, opts)
		if err != nil {
			// Check if it's our placeholder error
			if strings.Contains(err.Error(), "has been cordoned") {
				var content strings.Builder
				content.WriteString("⚠️  Node drain partially implemented\n\n")
				content.WriteString(fmt.Sprintf("Node has been cordoned (marked as unschedulable)\n"))
				content.WriteString("\nFull drain implementation would:\n")
				content.WriteString("1. List all pods on the node\n")
				content.WriteString("2. Filter out DaemonSet pods if requested\n")
				content.WriteString("3. Create pod evictions respecting PodDisruptionBudgets\n")
				content.WriteString("4. Wait for pods to terminate gracefully\n")
				content.WriteString("5. Force delete pods that exceed grace period\n\n")
				content.WriteString("For now, you can manually drain using kubectl:\n")
				if nodeName != "" {
					content.WriteString(fmt.Sprintf("  kubectl drain %s --ignore-daemonsets --delete-emptydir-data\n", nodeName))
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
			return mcp.NewToolResultError(fmt.Sprintf("Failed to drain node: %v", err)), nil
		}

		var content strings.Builder
		content.WriteString("✅ Successfully drained node\n\n")
		content.WriteString("Drain Options Applied:\n")
		content.WriteString(fmt.Sprintf("  • Ignore DaemonSets: %v\n", opts.IgnoreDaemonSets))
		content.WriteString(fmt.Sprintf("  • Delete Local Data: %v\n", opts.DeleteLocalData))
		content.WriteString(fmt.Sprintf("  • Force: %v\n", opts.Force))
		if opts.GracePeriodSeconds != nil {
			content.WriteString(fmt.Sprintf("  • Grace Period: %d seconds\n", *opts.GracePeriodSeconds))
		}
		content.WriteString("\nThe node is now:\n")
		content.WriteString("• Cordoned (no new pods will be scheduled)\n")
		content.WriteString("• Drained (existing pods have been evicted)\n")

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

// createCordonNodeHandler creates a handler for cordoning/uncordoning nodes
func createCordonNodeHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		// Build options
		opts := capi.NodeOperationOptions{}

		// Either namespace+machineName or nodeName is required
		namespace, _ := arguments["namespace"].(string)
		machineName, _ := arguments["machine_name"].(string)
		nodeName, _ := arguments["node_name"].(string)

		if nodeName == "" && (namespace == "" || machineName == "") {
			return nil, fmt.Errorf("either node_name or (namespace and machine_name) must be provided")
		}

		opts.Namespace = namespace
		opts.MachineName = machineName
		opts.NodeName = nodeName
		opts.Uncordon, _ = arguments["uncordon"].(bool)

		// Cordon/uncordon the node
		err := serverCtx.capiClient.CordonNode(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update node: %v", err)), nil
		}

		var content strings.Builder
		action := "cordoned"
		if opts.Uncordon {
			action = "uncordoned"
		}

		content.WriteString(fmt.Sprintf("✅ Successfully %s node\n\n", action))

		if opts.Uncordon {
			content.WriteString("The node is now:\n")
			content.WriteString("• Schedulable (new pods can be scheduled on this node)\n")
			content.WriteString("• Ready to accept workloads\n")
		} else {
			content.WriteString("The node is now:\n")
			content.WriteString("• Unschedulable (no new pods will be scheduled)\n")
			content.WriteString("• Existing pods will continue running\n\n")
			content.WriteString("To drain the node and evict pods, use:\n")
			content.WriteString("  capi_drain_node\n")
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

// createNodeStatusHandler creates a handler for getting node status
func createNodeStatusHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		// Build options
		opts := capi.NodeOperationOptions{}

		// Either namespace+machineName or nodeName is required
		namespace, _ := arguments["namespace"].(string)
		machineName, _ := arguments["machine_name"].(string)
		nodeName, _ := arguments["node_name"].(string)

		if nodeName == "" && (namespace == "" || machineName == "") {
			return nil, fmt.Errorf("either node_name or (namespace and machine_name) must be provided")
		}

		opts.Namespace = namespace
		opts.MachineName = machineName
		opts.NodeName = nodeName

		// Get node status
		node, err := serverCtx.capiClient.GetNodeStatus(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get node status: %v", err)), nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("Node: %s\n\n", node.Name))

		// Basic information
		content.WriteString("Basic Information:\n")
		content.WriteString(fmt.Sprintf("  UID: %s\n", node.UID))
		content.WriteString(fmt.Sprintf("  Created: %s\n", node.CreationTimestamp))
		content.WriteString(fmt.Sprintf("  Schedulable: %v\n", !node.Spec.Unschedulable))
		if node.Spec.ProviderID != "" {
			content.WriteString(fmt.Sprintf("  Provider ID: %s\n", node.Spec.ProviderID))
		}

		// Node info
		info := node.Status.NodeInfo
		content.WriteString("\nNode Info:\n")
		content.WriteString(fmt.Sprintf("  OS: %s (%s)\n", info.OperatingSystem, info.OSImage))
		content.WriteString(fmt.Sprintf("  Kernel: %s\n", info.KernelVersion))
		content.WriteString(fmt.Sprintf("  Container Runtime: %s\n", info.ContainerRuntimeVersion))
		content.WriteString(fmt.Sprintf("  Kubelet: %s\n", info.KubeletVersion))
		content.WriteString(fmt.Sprintf("  Architecture: %s\n", info.Architecture))

		// Capacity and allocatable resources
		content.WriteString("\nResources:\n")
		content.WriteString("  Capacity:\n")
		if cpu := node.Status.Capacity[v1.ResourceCPU]; !cpu.IsZero() {
			content.WriteString(fmt.Sprintf("    CPU: %s\n", cpu.String()))
		}
		if memory := node.Status.Capacity[v1.ResourceMemory]; !memory.IsZero() {
			content.WriteString(fmt.Sprintf("    Memory: %s\n", memory.String()))
		}
		if pods := node.Status.Capacity[v1.ResourcePods]; !pods.IsZero() {
			content.WriteString(fmt.Sprintf("    Pods: %s\n", pods.String()))
		}

		content.WriteString("  Allocatable:\n")
		if cpu := node.Status.Allocatable[v1.ResourceCPU]; !cpu.IsZero() {
			content.WriteString(fmt.Sprintf("    CPU: %s\n", cpu.String()))
		}
		if memory := node.Status.Allocatable[v1.ResourceMemory]; !memory.IsZero() {
			content.WriteString(fmt.Sprintf("    Memory: %s\n", memory.String()))
		}
		if pods := node.Status.Allocatable[v1.ResourcePods]; !pods.IsZero() {
			content.WriteString(fmt.Sprintf("    Pods: %s\n", pods.String()))
		}

		// Conditions
		content.WriteString("\nConditions:\n")
		for _, condition := range node.Status.Conditions {
			content.WriteString(fmt.Sprintf("  - Type: %s\n", condition.Type))
			content.WriteString(fmt.Sprintf("    Status: %s\n", condition.Status))
			if condition.Reason != "" {
				content.WriteString(fmt.Sprintf("    Reason: %s\n", condition.Reason))
			}
			if condition.Message != "" {
				content.WriteString(fmt.Sprintf("    Message: %s\n", condition.Message))
			}
		}

		// Addresses
		if len(node.Status.Addresses) > 0 {
			content.WriteString("\nAddresses:\n")
			for _, addr := range node.Status.Addresses {
				content.WriteString(fmt.Sprintf("  - %s: %s\n", addr.Type, addr.Address))
			}
		}

		// Taints
		if len(node.Spec.Taints) > 0 {
			content.WriteString("\nTaints:\n")
			for _, taint := range node.Spec.Taints {
				content.WriteString(fmt.Sprintf("  - Key: %s\n", taint.Key))
				if taint.Value != "" {
					content.WriteString(fmt.Sprintf("    Value: %s\n", taint.Value))
				}
				content.WriteString(fmt.Sprintf("    Effect: %s\n", taint.Effect))
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
