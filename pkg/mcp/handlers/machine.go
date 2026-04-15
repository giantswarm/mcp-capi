package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	v1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
)

// listResult holds the items and count from a list API call.
type listResult struct {
	count  int
	format func(w *strings.Builder)
}

// newListHandler creates a handler that lists resources with optional cluster name filtering.
func newListHandler(
	serverCtx *ServerContext,
	resourceName string,
	listFn func(ctx context.Context, c *capi.Client, ns, cluster string) (*listResult, error),
) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, errors.New("namespace argument is required")
		}
		clusterName, _ := arguments["clusterName"].(string)

		result, err := listFn(ctx, serverCtx.CAPIClient, namespace, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to list %s: %w", resourceName, err)
		}

		var content strings.Builder
		fmt.Fprintf(&content, "Found %d %s", result.count, resourceName)
		if clusterName != "" {
			content.WriteString(" in cluster " + clusterName)
		}
		content.WriteString(":\n\n")
		result.format(&content)

		return mcp.NewToolResultText(content.String()), nil
	}
}

// newGetHandler creates a handler that gets a single resource by namespace and name.
func newGetHandler(
	serverCtx *ServerContext,
	resourceName string,
	getFn func(ctx context.Context, c *capi.Client, ns, name string) (string, error),
) server.ToolHandlerFunc {
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

		text, err := getFn(ctx, serverCtx.CAPIClient, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get %s: %w", resourceName, err)
		}

		return mcp.NewToolResultText(text), nil
	}
}

// CreateListMachinesHandler creates a handler for listing CAPI machines
func CreateListMachinesHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return newListHandler(serverCtx, "machines",
		func(ctx context.Context, c *capi.Client, ns, cluster string) (*listResult, error) {
			machines, err := c.ListMachines(ctx, ns, cluster)
			if err != nil {
				return nil, err
			}
			return &listResult{
				count: len(machines.Items),
				format: func(w *strings.Builder) {
					for i := range machines.Items {
						formatMachineListItem(w, &machines.Items[i])
					}
				},
			}, nil
		})
}

// CreateListMachineDeploymentsHandler creates a handler for listing CAPI machine deployments
func CreateListMachineDeploymentsHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return newListHandler(serverCtx, "machine deployments",
		func(ctx context.Context, c *capi.Client, ns, cluster string) (*listResult, error) {
			mds, err := c.ListMachineDeployments(ctx, ns, cluster)
			if err != nil {
				return nil, err
			}
			return &listResult{
				count: len(mds.Items),
				format: func(w *strings.Builder) {
					for i := range mds.Items {
						formatMachineDeploymentListItem(w, &mds.Items[i])
					}
				},
			}, nil
		})
}

// CreateGetMachineHandler creates a handler for getting detailed machine information
func CreateGetMachineHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return newGetHandler(serverCtx, "machine",
		func(ctx context.Context, c *capi.Client, ns, name string) (string, error) {
			machine, err := c.GetMachine(ctx, ns, name)
			if err != nil {
				return "", err
			}
			return formatMachineDetails(machine), nil
		})
}

// CreateDeleteMachineHandler creates a handler for deleting CAPI machines
func CreateDeleteMachineHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		force, _ := arguments["force"].(bool)

		// Delete the machine
		err := serverCtx.CAPIClient.DeleteMachine(ctx, capi.DeleteMachineOptions{
			Namespace: namespace,
			Name:      name,
			Force:     force,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete machine: %v", err)), nil
		}

		var content strings.Builder
		fmt.Fprintf(&content, "✅ Successfully initiated deletion of machine %s/%s\n\n", namespace, name)
		content.WriteString("Note: Machine deletion is asynchronous. The machine will be:\n")
		content.WriteString("1. Drained (if it has a node)\n")
		content.WriteString("2. Removed from the cluster\n")
		content.WriteString("3. Infrastructure resources cleaned up\n\n")
		content.WriteString("Monitor deletion progress with:\n")
		fmt.Fprintf(&content, "  capi_get_machine --namespace %s --name %s\n", namespace, name)

		return mcp.NewToolResultText(content.String()), nil
	}
}

// CreateRemediateMachineHandler creates a handler for triggering machine remediation
func CreateRemediateMachineHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Get current machine status first
		machine, err := serverCtx.CAPIClient.GetMachine(ctx, namespace, name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get machine: %v", err)), nil
		}

		// Trigger remediation
		err = serverCtx.CAPIClient.RemediateMachine(ctx, capi.RemediateMachineOptions{
			Namespace: namespace,
			Name:      name,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to remediate machine: %v", err)), nil
		}

		var content strings.Builder
		fmt.Fprintf(&content, "🔧 Triggered remediation for machine %s/%s\n\n", namespace, name)
		content.WriteString("Current Machine Status:\n")
		fmt.Fprintf(&content, "  • Phase: %s\n", machine.Status.Phase)
		if machine.Status.NodeRef != nil {
			fmt.Fprintf(&content, "  • Node: %s\n", machine.Status.NodeRef.Name)
		}
		content.WriteString("\nRemediation Process:\n")
		content.WriteString("1. Machine will be marked for remediation\n")
		content.WriteString("2. MachineHealthCheck controller will process the remediation\n")
		content.WriteString("3. Depending on remediation strategy:\n")
		content.WriteString("   - Machine may be deleted and recreated\n")
		content.WriteString("   - Node may be rebooted\n")
		content.WriteString("   - Custom remediation may be applied\n\n")
		content.WriteString("Monitor remediation progress with:\n")
		fmt.Fprintf(&content, "  capi_get_machine --namespace %s --name %s\n", namespace, name)

		return mcp.NewToolResultText(content.String()), nil
	}
}

// CreateCreateMachineDeploymentHandler creates a handler for creating new machine deployments
func CreateCreateMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		opts, names, errResult := parseMachineDeploymentCreateArgs(arguments)
		if errResult != nil {
			return errResult, nil
		}

		// Create the machine deployment
		md, err := serverCtx.CAPIClient.CreateMachineDeployment(ctx, *opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create machine deployment: %v", err)), nil
		}

		text := formatCreateMachineDeploymentResult(
			opts.Namespace, opts.Name, opts.ClusterName, opts.Version,
			names.infraKind, names.infraName, names.bootstrapKind, names.bootstrapName,
			opts.Replicas, md,
		)
		return mcp.NewToolResultText(text), nil
	}
}

// CreateScaleMachineDeploymentHandler creates a handler for scaling machine deployments
func CreateScaleMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		replicasFloat, ok := arguments["replicas"].(float64)
		if !ok {
			return nil, errors.New("replicas argument is required")
		}
		replicas := int32(replicasFloat)

		// Get current state
		list, err := serverCtx.CAPIClient.ListMachineDeployments(ctx, namespace, "")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get machine deployment: %v", err)), nil
		}

		currentReplicas, found := findMachineDeploymentReplicas(list.Items, name)
		if !found {
			return mcp.NewToolResultError(fmt.Sprintf("Machine deployment %s/%s not found", namespace, name)), nil
		}

		// Scale the machine deployment
		err = serverCtx.CAPIClient.ScaleMachineDeployment(ctx, namespace, name, replicas)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to scale machine deployment: %v", err)), nil
		}

		text := formatScaleMachineDeploymentResult(namespace, name, currentReplicas, replicas)
		return mcp.NewToolResultText(text), nil
	}
}

// CreateUpdateMachineDeploymentHandler creates a handler for updating machine deployment configuration
func CreateUpdateMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		opts := parseMachineDeploymentUpdateOpts(namespace, name, arguments)

		// Update the machine deployment
		md, err := serverCtx.CAPIClient.UpdateMachineDeployment(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update machine deployment: %v", err)), nil
		}

		return mcp.NewToolResultText(formatUpdateMachineDeploymentResult(namespace, name, opts, md)), nil
	}
}

// CreateRolloutMachineDeploymentHandler creates a handler for triggering machine deployment rollout
func CreateRolloutMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		reason, _ := arguments["reason"].(string)

		// Trigger the rollout
		err := serverCtx.CAPIClient.RolloutMachineDeployment(ctx, capi.RolloutMachineDeploymentOptions{
			Namespace: namespace,
			Name:      name,
			Reason:    reason,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to trigger rollout: %v", err)), nil
		}

		var content strings.Builder
		fmt.Fprintf(&content, "🔄 Successfully triggered rollout for machine deployment %s/%s\n\n", namespace, name)

		if reason != "" {
			fmt.Fprintf(&content, "Reason: %s\n\n", reason)
		}

		content.WriteString("Rollout Process:\n")
		content.WriteString("1. New machines will be created with updated configuration\n")
		content.WriteString("2. Old machines will be gradually replaced\n")
		content.WriteString("3. The rollout respects the deployment's update strategy\n")
		content.WriteString("4. Health checks ensure machines are ready before proceeding\n\n")

		content.WriteString("Monitor rollout progress with:\n")
		fmt.Fprintf(&content, "  capi_list_machines --namespace %s --cluster <cluster-name>\n", namespace)
		fmt.Fprintf(&content, "  capi_list_machinedeployments --namespace %s\n", namespace)

		return mcp.NewToolResultText(content.String()), nil
	}
}

// CreateListMachineSetsHandler creates a handler for listing machine sets
func CreateListMachineSetsHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return newListHandler(serverCtx, "machine sets",
		func(ctx context.Context, c *capi.Client, ns, cluster string) (*listResult, error) {
			machineSets, err := c.ListMachineSets(ctx, ns, cluster)
			if err != nil {
				return nil, err
			}
			return &listResult{
				count: len(machineSets.Items),
				format: func(w *strings.Builder) {
					for i := range machineSets.Items {
						formatMachineSetListItem(w, &machineSets.Items[i])
					}
				},
			}, nil
		})
}

// CreateGetMachineSetHandler creates a handler for getting machine set details
func CreateGetMachineSetHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return newGetHandler(serverCtx, "machine set",
		func(ctx context.Context, c *capi.Client, ns, name string) (string, error) {
			ms, err := c.GetMachineSet(ctx, ns, name)
			if err != nil {
				return "", err
			}
			return formatMachineSetDetails(ms), nil
		})
}

// CreateDrainNodeHandler creates a handler for draining nodes
func CreateDrainNodeHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		// Build options
		opts := capi.NodeOperationOptions{}

		// Either namespace+machineName or nodeName is required
		namespace, _ := arguments["namespace"].(string)
		machineName, _ := arguments["machine_name"].(string)
		nodeName, _ := arguments["node_name"].(string)

		if nodeName == "" && (namespace == "" || machineName == "") {
			return nil, errors.New("either node_name or (namespace and machine_name) must be provided")
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
		err := serverCtx.CAPIClient.DrainNode(ctx, opts)
		if err != nil {
			// Check if it's our placeholder error
			if strings.Contains(err.Error(), "has been cordoned") {
				return mcp.NewToolResultText(formatPartialDrainResult(nodeName)), nil
			}
			return mcp.NewToolResultError(fmt.Sprintf("Failed to drain node: %v", err)), nil
		}

		var content strings.Builder
		content.WriteString("✅ Successfully drained node\n\n")
		content.WriteString("Drain Options Applied:\n")
		fmt.Fprintf(&content, "  • Ignore DaemonSets: %v\n", opts.IgnoreDaemonSets)
		fmt.Fprintf(&content, "  • Delete Local Data: %v\n", opts.DeleteLocalData)
		fmt.Fprintf(&content, "  • Force: %v\n", opts.Force)
		if opts.GracePeriodSeconds != nil {
			fmt.Fprintf(&content, "  • Grace Period: %d seconds\n", *opts.GracePeriodSeconds)
		}
		content.WriteString("\nThe node is now:\n")
		content.WriteString("• Cordoned (no new pods will be scheduled)\n")
		content.WriteString("• Drained (existing pods have been evicted)\n")

		return mcp.NewToolResultText(content.String()), nil
	}
}

// CreateCordonNodeHandler creates a handler for cordoning/uncordoning nodes
func CreateCordonNodeHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		// Build options
		opts := capi.NodeOperationOptions{}

		// Either namespace+machineName or nodeName is required
		namespace, _ := arguments["namespace"].(string)
		machineName, _ := arguments["machine_name"].(string)
		nodeName, _ := arguments["node_name"].(string)

		if nodeName == "" && (namespace == "" || machineName == "") {
			return nil, errors.New("either node_name or (namespace and machine_name) must be provided")
		}

		opts.Namespace = namespace
		opts.MachineName = machineName
		opts.NodeName = nodeName
		opts.Uncordon, _ = arguments["uncordon"].(bool)

		// Cordon/uncordon the node
		err := serverCtx.CAPIClient.CordonNode(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update node: %v", err)), nil
		}

		var content strings.Builder
		action := "cordoned"
		if opts.Uncordon {
			action = "uncordoned"
		}

		fmt.Fprintf(&content, "✅ Successfully %s node\n\n", action)

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

		return mcp.NewToolResultText(content.String()), nil
	}
}

// CreateNodeStatusHandler creates a handler for getting node status
func CreateNodeStatusHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		// Build options
		opts := capi.NodeOperationOptions{}

		// Either namespace+machineName or nodeName is required
		namespace, _ := arguments["namespace"].(string)
		machineName, _ := arguments["machine_name"].(string)
		nodeName, _ := arguments["node_name"].(string)

		if nodeName == "" && (namespace == "" || machineName == "") {
			return nil, errors.New("either node_name or (namespace and machine_name) must be provided")
		}

		opts.Namespace = namespace
		opts.MachineName = machineName
		opts.NodeName = nodeName

		// Get node status
		node, err := serverCtx.CAPIClient.GetNodeStatus(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get node status: %v", err)), nil
		}

		return mcp.NewToolResultText(formatNodeStatus(node)), nil
	}
}

// formatMachineListItem writes a single machine summary entry into the builder.
func formatMachineListItem(w *strings.Builder, machine *clusterv1.Machine) {
	fmt.Fprintf(w, "Machine: %s/%s\n", machine.Namespace, machine.Name)
	fmt.Fprintf(w, "  Cluster: %s\n", machine.Spec.ClusterName)
	if machine.Status.Phase != "" {
		fmt.Fprintf(w, "  Phase: %s\n", machine.Status.Phase)
	}
	if machine.Status.NodeRef != nil {
		fmt.Fprintf(w, "  Node: %s\n", machine.Status.NodeRef.Name)
	}
	if machine.Spec.ProviderID != nil {
		fmt.Fprintf(w, "  Provider ID: %s\n", *machine.Spec.ProviderID)
	}
	// Check if machine has Ready condition
	ready := false
	for _, condition := range machine.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == "True" {
			ready = true
			break
		}
	}
	fmt.Fprintf(w, "  Ready: %v\n", ready)
	w.WriteString("\n")
}

// formatMachineDetails returns a detailed text description of a machine.
func formatMachineDetails(machine *clusterv1.Machine) string {
	var w strings.Builder
	fmt.Fprintf(&w, "Machine: %s/%s\n\n", machine.Namespace, machine.Name)

	// Basic information
	w.WriteString("Basic Information:\n")
	fmt.Fprintf(&w, "  Cluster: %s\n", machine.Spec.ClusterName)
	if machine.Status.Phase != "" {
		fmt.Fprintf(&w, "  Phase: %s\n", machine.Status.Phase)
	}
	if machine.Spec.Version != nil {
		fmt.Fprintf(&w, "  Kubernetes Version: %s\n", *machine.Spec.Version)
	}
	if machine.Spec.ProviderID != nil {
		fmt.Fprintf(&w, "  Provider ID: %s\n", *machine.Spec.ProviderID)
	}

	// Node information
	if machine.Status.NodeRef != nil {
		w.WriteString("\nNode Information:\n")
		fmt.Fprintf(&w, "  Node Name: %s\n", machine.Status.NodeRef.Name)
		fmt.Fprintf(&w, "  Node UID: %s\n", machine.Status.NodeRef.UID)
	}

	// Bootstrap information
	if machine.Spec.Bootstrap.ConfigRef != nil {
		w.WriteString("\nBootstrap:\n")
		fmt.Fprintf(&w, "  Kind: %s\n", machine.Spec.Bootstrap.ConfigRef.Kind)
		fmt.Fprintf(&w, "  Name: %s\n", machine.Spec.Bootstrap.ConfigRef.Name)
	}

	// Infrastructure information
	if machine.Spec.InfrastructureRef.Kind != "" {
		w.WriteString("\nInfrastructure:\n")
		fmt.Fprintf(&w, "  Kind: %s\n", machine.Spec.InfrastructureRef.Kind)
		fmt.Fprintf(&w, "  Name: %s\n", machine.Spec.InfrastructureRef.Name)
	}

	// Conditions
	if len(machine.Status.Conditions) > 0 {
		w.WriteString("\nConditions:\n")
		for _, condition := range machine.Status.Conditions {
			fmt.Fprintf(&w, "  - Type: %s\n", condition.Type)
			fmt.Fprintf(&w, "    Status: %s\n", condition.Status)
			if condition.Reason != "" {
				fmt.Fprintf(&w, "    Reason: %s\n", condition.Reason)
			}
			if condition.Message != "" {
				fmt.Fprintf(&w, "    Message: %s\n", condition.Message)
			}
		}
	}

	// Addresses
	if len(machine.Status.Addresses) > 0 {
		w.WriteString("\nAddresses:\n")
		for _, addr := range machine.Status.Addresses {
			fmt.Fprintf(&w, "  - Type: %s, Address: %s\n", addr.Type, addr.Address)
		}
	}

	return w.String()
}

// formatMachineDeploymentListItem writes a single machine deployment summary entry into the builder.
func formatMachineDeploymentListItem(w *strings.Builder, md *clusterv1.MachineDeployment) {
	fmt.Fprintf(w, "MachineDeployment: %s/%s\n", md.Namespace, md.Name)
	fmt.Fprintf(w, "  Cluster: %s\n", md.Spec.ClusterName)
	if md.Spec.Replicas != nil {
		fmt.Fprintf(w, "  Replicas: %d\n", *md.Spec.Replicas)
	}
	if md.Status.Replicas > 0 {
		fmt.Fprintf(w, "  Status: %d ready / %d updated / %d available\n",
			md.Status.ReadyReplicas,
			md.Status.UpdatedReplicas,
			md.Status.AvailableReplicas)
	}
	if md.Status.Phase != "" {
		fmt.Fprintf(w, "  Phase: %s\n", md.Status.Phase)
	}
	if md.Spec.Template.Spec.Version != nil {
		fmt.Fprintf(w, "  Kubernetes Version: %s\n", *md.Spec.Template.Spec.Version)
	}
	w.WriteString("\n")
}

// formatCreateMachineDeploymentResult returns the success text for a created machine deployment.
func formatCreateMachineDeploymentResult(
	namespace, name, clusterName, version string,
	infraKind, infraName, bootstrapKind, bootstrapName string,
	replicas int32,
	md *clusterv1.MachineDeployment,
) string {
	var w strings.Builder
	fmt.Fprintf(&w, "✅ Successfully created machine deployment %s/%s\n\n", namespace, name)
	w.WriteString("Configuration:\n")
	fmt.Fprintf(&w, "  • Cluster: %s\n", clusterName)
	fmt.Fprintf(&w, "  • Replicas: %d\n", replicas)
	fmt.Fprintf(&w, "  • Version: %s\n", version)
	fmt.Fprintf(&w, "  • Infrastructure: %s/%s\n", infraKind, infraName)
	fmt.Fprintf(&w, "  • Bootstrap: %s/%s\n", bootstrapKind, bootstrapName)
	if md.Spec.MinReadySeconds != nil {
		fmt.Fprintf(&w, "  • Min Ready Seconds: %d\n", *md.Spec.MinReadySeconds)
	}
	w.WriteString("\nNote: Before creating a MachineDeployment, ensure you have:\n")
	w.WriteString("1. Created the infrastructure template (e.g., AWSMachineTemplate)\n")
	w.WriteString("2. Created the bootstrap config template (e.g., KubeadmConfigTemplate)\n\n")
	w.WriteString("Monitor the deployment with:\n")
	fmt.Fprintf(&w, "  capi_list_machines --cluster %s\n", clusterName)
	w.WriteString("\nScale the deployment with:\n")
	fmt.Fprintf(&w, "  capi_scale_machinedeployment --namespace %s --name %s --replicas <count>\n", namespace, name)
	return w.String()
}

// formatScaleMachineDeploymentResult returns the success text for a scaled machine deployment.
func formatScaleMachineDeploymentResult(namespace, name string, currentReplicas, replicas int32) string {
	var w strings.Builder
	fmt.Fprintf(&w, "✅ Successfully scaled machine deployment %s/%s\n\n", namespace, name)
	w.WriteString("Scaling Operation:\n")
	fmt.Fprintf(&w, "  • Previous Replicas: %d\n", currentReplicas)
	fmt.Fprintf(&w, "  • New Replicas: %d\n", replicas)

	switch {
	case replicas > currentReplicas:
		fmt.Fprintf(&w, "  • Action: Scaling UP by %d nodes\n", replicas-currentReplicas)
		w.WriteString("\nNew nodes will be:\n")
		w.WriteString("1. Provisioned by the infrastructure provider\n")
		w.WriteString("2. Bootstrapped with Kubernetes\n")
		w.WriteString("3. Joined to the cluster\n")
	case replicas < currentReplicas:
		fmt.Fprintf(&w, "  • Action: Scaling DOWN by %d nodes\n", currentReplicas-replicas)
		w.WriteString("\nNodes will be:\n")
		w.WriteString("1. Cordoned to prevent new workloads\n")
		w.WriteString("2. Drained to move existing workloads\n")
		w.WriteString("3. Removed from the cluster\n")
		w.WriteString("4. Infrastructure resources cleaned up\n")
	default:
		w.WriteString("  • Action: No change (same replica count)\n")
	}

	fmt.Fprintf(&w, "\nMonitor scaling progress with:\n  capi_list_machines --namespace %s\n", namespace)
	return w.String()
}

// formatUpdateMachineDeploymentResult returns the success text for an updated machine deployment.
func formatUpdateMachineDeploymentResult(
	namespace, name string,
	opts capi.UpdateMachineDeploymentOptions,
	md *clusterv1.MachineDeployment,
) string {
	var w strings.Builder
	fmt.Fprintf(&w, "✅ Successfully updated machine deployment %s/%s\n\n", namespace, name)
	w.WriteString("Updated Configuration:\n")

	if opts.Version != nil {
		fmt.Fprintf(&w, "  • Version: %s\n", *opts.Version)
	}
	if opts.Replicas != nil {
		fmt.Fprintf(&w, "  • Replicas: %d\n", *opts.Replicas)
	}
	if opts.MinReadySeconds != nil {
		fmt.Fprintf(&w, "  • Min Ready Seconds: %d\n", *opts.MinReadySeconds)
	}
	if len(opts.Labels) > 0 {
		w.WriteString("  • Labels updated\n")
	}
	if len(opts.Annotations) > 0 {
		w.WriteString("  • Annotations updated\n")
	}

	w.WriteString("\nCurrent Status:\n")
	fmt.Fprintf(&w, "  • Ready Replicas: %d\n", md.Status.ReadyReplicas)
	fmt.Fprintf(&w, "  • Updated Replicas: %d\n", md.Status.UpdatedReplicas)
	fmt.Fprintf(&w, "  • Available Replicas: %d\n", md.Status.AvailableReplicas)
	return w.String()
}

// formatMachineSetListItem writes a single machine set summary entry into the builder.
func formatMachineSetListItem(w *strings.Builder, ms *clusterv1.MachineSet) {
	fmt.Fprintf(w, "MachineSet: %s/%s\n", ms.Namespace, ms.Name)
	fmt.Fprintf(w, "  Cluster: %s\n", ms.Spec.ClusterName)
	if ms.Spec.Replicas != nil {
		fmt.Fprintf(w, "  Replicas: %d\n", *ms.Spec.Replicas)
	}
	fmt.Fprintf(w, "  Ready: %d/%d\n", ms.Status.ReadyReplicas, ms.Status.Replicas)
	fmt.Fprintf(w, "  Available: %d\n", ms.Status.AvailableReplicas)

	// Show owner reference (usually MachineDeployment)
	for _, owner := range ms.OwnerReferences {
		if owner.Kind == "MachineDeployment" {
			fmt.Fprintf(w, "  Owner: MachineDeployment/%s\n", owner.Name)
		}
	}

	// Show machine template
	if ms.Spec.Template.Spec.InfrastructureRef.Name != "" {
		fmt.Fprintf(w, "  Infrastructure: %s/%s\n",
			ms.Spec.Template.Spec.InfrastructureRef.Kind,
			ms.Spec.Template.Spec.InfrastructureRef.Name)
	}
	w.WriteString("\n")
}

// formatMachineSetDetails returns a detailed text description of a machine set.
func formatMachineSetDetails(ms *clusterv1.MachineSet) string {
	var w strings.Builder
	fmt.Fprintf(&w, "MachineSet: %s/%s\n\n", ms.Namespace, ms.Name)

	// Basic information
	w.WriteString("Basic Information:\n")
	fmt.Fprintf(&w, "  Cluster: %s\n", ms.Spec.ClusterName)
	if ms.Spec.Replicas != nil {
		fmt.Fprintf(&w, "  Desired Replicas: %d\n", *ms.Spec.Replicas)
	}

	// Status
	w.WriteString("\nStatus:\n")
	fmt.Fprintf(&w, "  Total Replicas: %d\n", ms.Status.Replicas)
	fmt.Fprintf(&w, "  Ready Replicas: %d\n", ms.Status.ReadyReplicas)
	fmt.Fprintf(&w, "  Available Replicas: %d\n", ms.Status.AvailableReplicas)
	if reason := ms.Status.FailureReason; reason != nil { //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
		fmt.Fprintf(&w, "  Failure Reason: %s\n", *reason)
	}
	if msg := ms.Status.FailureMessage; msg != nil { //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
		fmt.Fprintf(&w, "  Failure Message: %s\n", *msg)
	}

	// Machine template
	w.WriteString("\nMachine Template:\n")
	if ms.Spec.Template.Spec.Version != nil {
		fmt.Fprintf(&w, "  Kubernetes Version: %s\n", *ms.Spec.Template.Spec.Version)
	}
	if ms.Spec.Template.Spec.InfrastructureRef.Name != "" {
		fmt.Fprintf(&w, "  Infrastructure: %s/%s\n",
			ms.Spec.Template.Spec.InfrastructureRef.Kind,
			ms.Spec.Template.Spec.InfrastructureRef.Name)
	}
	if ms.Spec.Template.Spec.Bootstrap.ConfigRef != nil {
		fmt.Fprintf(&w, "  Bootstrap: %s/%s\n",
			ms.Spec.Template.Spec.Bootstrap.ConfigRef.Kind,
			ms.Spec.Template.Spec.Bootstrap.ConfigRef.Name)
	}

	// Owner references
	if len(ms.OwnerReferences) > 0 {
		w.WriteString("\nOwners:\n")
		for _, owner := range ms.OwnerReferences {
			fmt.Fprintf(&w, "  - %s: %s\n", owner.Kind, owner.Name)
		}
	}

	// Conditions
	if len(ms.Status.Conditions) > 0 {
		w.WriteString("\nConditions:\n")
		for _, condition := range ms.Status.Conditions {
			fmt.Fprintf(&w, "  - Type: %s\n", condition.Type)
			fmt.Fprintf(&w, "    Status: %s\n", condition.Status)
			if condition.Reason != "" {
				fmt.Fprintf(&w, "    Reason: %s\n", condition.Reason)
			}
			if condition.Message != "" {
				fmt.Fprintf(&w, "    Message: %s\n", condition.Message)
			}
		}
	}

	return w.String()
}

// formatPartialDrainResult returns the text for a node drain that only cordoned the node.
func formatPartialDrainResult(nodeName string) string {
	var w strings.Builder
	w.WriteString("⚠️  Node drain partially implemented\n\n")
	w.WriteString("Node has been cordoned (marked as unschedulable)\n")
	w.WriteString("\nFull drain implementation would:\n")
	w.WriteString("1. List all pods on the node\n")
	w.WriteString("2. Filter out DaemonSet pods if requested\n")
	w.WriteString("3. Create pod evictions respecting PodDisruptionBudgets\n")
	w.WriteString("4. Wait for pods to terminate gracefully\n")
	w.WriteString("5. Force delete pods that exceed grace period\n\n")
	w.WriteString("For now, you can manually drain using kubectl:\n")
	if nodeName != "" {
		fmt.Fprintf(&w, "  kubectl drain %s --ignore-daemonsets --delete-emptydir-data\n", nodeName)
	}
	return w.String()
}

// formatNodeStatus returns a detailed text description of a node's status.
func formatNodeStatus(node *v1.Node) string {
	var w strings.Builder
	fmt.Fprintf(&w, "Node: %s\n\n", node.Name)

	// Basic information
	w.WriteString("Basic Information:\n")
	fmt.Fprintf(&w, "  UID: %s\n", node.UID)
	fmt.Fprintf(&w, "  Created: %s\n", node.CreationTimestamp)
	fmt.Fprintf(&w, "  Schedulable: %v\n", !node.Spec.Unschedulable)
	if node.Spec.ProviderID != "" {
		fmt.Fprintf(&w, "  Provider ID: %s\n", node.Spec.ProviderID)
	}

	// Node info
	info := node.Status.NodeInfo
	w.WriteString("\nNode Info:\n")
	fmt.Fprintf(&w, "  OS: %s (%s)\n", info.OperatingSystem, info.OSImage)
	fmt.Fprintf(&w, "  Kernel: %s\n", info.KernelVersion)
	fmt.Fprintf(&w, "  Container Runtime: %s\n", info.ContainerRuntimeVersion)
	fmt.Fprintf(&w, "  Kubelet: %s\n", info.KubeletVersion)
	fmt.Fprintf(&w, "  Architecture: %s\n", info.Architecture)

	formatNodeResources(&w, node)
	formatNodeConditions(&w, node)
	formatNodeAddressesAndTaints(&w, node)

	return w.String()
}

// formatNodeResources writes capacity and allocatable resource sections.
func formatNodeResources(w *strings.Builder, node *v1.Node) {
	w.WriteString("\nResources:\n")
	w.WriteString("  Capacity:\n")
	if cpu := node.Status.Capacity[v1.ResourceCPU]; !cpu.IsZero() {
		fmt.Fprintf(w, "    CPU: %s\n", cpu.String())
	}
	if memory := node.Status.Capacity[v1.ResourceMemory]; !memory.IsZero() {
		fmt.Fprintf(w, "    Memory: %s\n", memory.String())
	}
	if pods := node.Status.Capacity[v1.ResourcePods]; !pods.IsZero() {
		fmt.Fprintf(w, "    Pods: %s\n", pods.String())
	}

	w.WriteString("  Allocatable:\n")
	if cpu := node.Status.Allocatable[v1.ResourceCPU]; !cpu.IsZero() {
		fmt.Fprintf(w, "    CPU: %s\n", cpu.String())
	}
	if memory := node.Status.Allocatable[v1.ResourceMemory]; !memory.IsZero() {
		fmt.Fprintf(w, "    Memory: %s\n", memory.String())
	}
	if pods := node.Status.Allocatable[v1.ResourcePods]; !pods.IsZero() {
		fmt.Fprintf(w, "    Pods: %s\n", pods.String())
	}
}

// formatNodeConditions writes the conditions section.
func formatNodeConditions(w *strings.Builder, node *v1.Node) {
	w.WriteString("\nConditions:\n")
	for _, condition := range node.Status.Conditions {
		fmt.Fprintf(w, "  - Type: %s\n", condition.Type)
		fmt.Fprintf(w, "    Status: %s\n", condition.Status)
		if condition.Reason != "" {
			fmt.Fprintf(w, "    Reason: %s\n", condition.Reason)
		}
		if condition.Message != "" {
			fmt.Fprintf(w, "    Message: %s\n", condition.Message)
		}
	}
}

// formatNodeAddressesAndTaints writes the addresses and taints sections.
func formatNodeAddressesAndTaints(w *strings.Builder, node *v1.Node) {
	if len(node.Status.Addresses) > 0 {
		w.WriteString("\nAddresses:\n")
		for _, addr := range node.Status.Addresses {
			fmt.Fprintf(w, "  - %s: %s\n", addr.Type, addr.Address)
		}
	}

	if len(node.Spec.Taints) > 0 {
		w.WriteString("\nTaints:\n")
		for _, taint := range node.Spec.Taints {
			fmt.Fprintf(w, "  - Key: %s\n", taint.Key)
			if taint.Value != "" {
				fmt.Fprintf(w, "    Value: %s\n", taint.Value)
			}
			fmt.Fprintf(w, "    Effect: %s\n", taint.Effect)
		}
	}
}

// machineDeploymentRefNames holds the kind/name strings for infra and bootstrap refs,
// used only for building human-readable output after a successful create.
type machineDeploymentRefNames struct {
	infraKind, infraName         string
	bootstrapKind, bootstrapName string
}

// parseMachineDeploymentCreateArgs parses all arguments for CreateMachineDeployment.
// On validation failure, errResult is non-nil and should be returned directly to the MCP caller.
func parseMachineDeploymentCreateArgs(arguments map[string]any) (
	opts *capi.CreateMachineDeploymentOptions,
	names machineDeploymentRefNames,
	errResult *mcp.CallToolResult,
) {
	namespace, ok := arguments["namespace"].(string)
	if !ok || namespace == "" {
		return nil, names, mcp.NewToolResultError("namespace argument is required")
	}
	name, nameOK := arguments["name"].(string)
	if !nameOK || name == "" {
		return nil, names, mcp.NewToolResultError("name argument is required")
	}
	clusterName, clusterOK := arguments["cluster_name"].(string)
	if !clusterOK || clusterName == "" {
		return nil, names, mcp.NewToolResultError("cluster_name argument is required")
	}

	replicas := int32(1)
	if r, ok := arguments["replicas"].(float64); ok {
		replicas = int32(r)
	}

	names.infraKind, _ = arguments["infra_kind"].(string)
	names.infraName, _ = arguments["infra_name"].(string)
	infraAPIVersion, _ := arguments["infra_api_version"].(string)

	if names.infraKind == "" || names.infraName == "" {
		return nil, names, mcp.NewToolResultError("infra_kind and infra_name are required")
	}

	names.bootstrapKind, _ = arguments["bootstrap_kind"].(string)
	names.bootstrapName, _ = arguments["bootstrap_name"].(string)
	bootstrapAPIVersion, _ := arguments["bootstrap_api_version"].(string)

	if names.bootstrapKind == "" || names.bootstrapName == "" {
		return nil, names, mcp.NewToolResultError("bootstrap_kind and bootstrap_name are required")
	}

	version, _ := arguments["version"].(string)
	if version == "" {
		version = "v1.29.0" // Default version
	}

	return &capi.CreateMachineDeploymentOptions{
		Namespace:   namespace,
		Name:        name,
		ClusterName: clusterName,
		Replicas:    replicas,
		InfrastructureRef: v1.ObjectReference{
			Kind:       names.infraKind,
			Name:       names.infraName,
			APIVersion: infraAPIVersion,
		},
		BootstrapConfigRef: v1.ObjectReference{
			Kind:       names.bootstrapKind,
			Name:       names.bootstrapName,
			APIVersion: bootstrapAPIVersion,
		},
		Version: version,
	}, names, nil
}

// findMachineDeploymentReplicas searches for a machine deployment by name in a list and returns
// its current replica count. Returns (0, false) when the deployment is not found.
func findMachineDeploymentReplicas(items []clusterv1.MachineDeployment, name string) (int32, bool) {
	for i := range items {
		if items[i].Name == name {
			if items[i].Spec.Replicas != nil {
				return *items[i].Spec.Replicas, true
			}
			return 0, true
		}
	}
	return 0, false
}

// parseMachineDeploymentUpdateOpts builds UpdateMachineDeploymentOptions from the handler arguments map.
func parseMachineDeploymentUpdateOpts(
	namespace, name string,
	arguments map[string]any,
) capi.UpdateMachineDeploymentOptions {
	opts := capi.UpdateMachineDeploymentOptions{
		Namespace: namespace,
		Name:      name,
	}

	if version, ok := arguments["version"].(string); ok && version != "" {
		opts.Version = &version
	}
	if replicasFloat, ok := arguments["replicas"].(float64); ok {
		replicas := int32(replicasFloat)
		opts.Replicas = &replicas
	}
	if minReadyFloat, ok := arguments["min_ready_seconds"].(float64); ok {
		minReady := int32(minReadyFloat)
		opts.MinReadySeconds = &minReady
	}

	if labels, ok := arguments["labels"].(map[string]any); ok {
		opts.Labels = convertStringMap(labels)
	}
	if annotations, ok := arguments["annotations"].(map[string]any); ok {
		opts.Annotations = convertStringMap(annotations)
	}

	return opts
}

// convertStringMap converts a map[string]any to map[string]string, dropping non-string values.
func convertStringMap(src map[string]any) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		if strVal, ok := v.(string); ok {
			out[k] = strVal
		}
	}
	return out
}
