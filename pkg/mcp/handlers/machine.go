package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	v1 "k8s.io/api/core/v1"

	"github.com/giantswarm/mcp-capi/pkg/capi"
)

// MachineSummary is one entry of capi_list_machines.
type MachineSummary struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	ClusterName string `json:"clusterName,omitempty"`
	Phase       string `json:"phase,omitempty"`
	NodeName    string `json:"nodeName,omitempty"`
	ProviderID  string `json:"providerID,omitempty"`
	Version     string `json:"version,omitempty"`
	Ready       bool   `json:"ready"`
}

// CreateListMachinesHandler creates a handler for listing CAPI machines as
// {items: [MachineSummary]}.
func CreateListMachinesHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		clusterName, _ := arguments["clusterName"].(string)

		machines, err := capiClient.ListMachines(ctx, namespace, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to list machines: %w", err)
		}

		items := make([]MachineSummary, 0, len(machines.Items))
		for i := range machines.Items {
			machine := &machines.Items[i]
			item := MachineSummary{
				Name:        machine.Name,
				Namespace:   machine.Namespace,
				ClusterName: machine.Spec.ClusterName,
				Phase:       machine.Status.Phase,
				Ready:       machineReady(machine),
			}
			if machine.Status.NodeRef != nil {
				item.NodeName = machine.Status.NodeRef.Name
			}
			if machine.Spec.ProviderID != nil {
				item.ProviderID = *machine.Spec.ProviderID
			}
			if machine.Spec.Version != nil {
				item.Version = *machine.Spec.Version
			}
			items = append(items, item)
		}

		return listResult(items)
	}
}

// MachineDeploymentSummary is one entry of capi_list_machinedeployments.
type MachineDeploymentSummary struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	ClusterName       string `json:"clusterName,omitempty"`
	Phase             string `json:"phase,omitempty"`
	Replicas          int32  `json:"replicas"`
	ReadyReplicas     int32  `json:"readyReplicas"`
	UpdatedReplicas   int32  `json:"updatedReplicas"`
	AvailableReplicas int32  `json:"availableReplicas"`
	Version           string `json:"version,omitempty"`
}

// CreateListMachineDeploymentsHandler creates a handler for listing CAPI
// machine deployments as {items: [MachineDeploymentSummary]}.
func CreateListMachineDeploymentsHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		clusterName, _ := arguments["clusterName"].(string)

		mds, err := capiClient.ListMachineDeployments(ctx, namespace, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to list machine deployments: %w", err)
		}

		items := make([]MachineDeploymentSummary, 0, len(mds.Items))
		for i := range mds.Items {
			md := &mds.Items[i]
			item := MachineDeploymentSummary{
				Name:              md.Name,
				Namespace:         md.Namespace,
				ClusterName:       md.Spec.ClusterName,
				Phase:             md.Status.Phase,
				ReadyReplicas:     md.Status.ReadyReplicas,
				UpdatedReplicas:   md.Status.UpdatedReplicas,
				AvailableReplicas: md.Status.AvailableReplicas,
			}
			if md.Spec.Replicas != nil {
				item.Replicas = *md.Spec.Replicas
			}
			if md.Spec.Template.Spec.Version != nil {
				item.Version = *md.Spec.Template.Spec.Version
			}
			items = append(items, item)
		}

		return listResult(items)
	}
}

// NodeRef names the node a machine is backing.
type NodeRef struct {
	Name string `json:"name"`
	UID  string `json:"uid,omitempty"`
}

// MachineDetail is the result of capi_get_machine.
type MachineDetail struct {
	Name           string      `json:"name"`
	Namespace      string      `json:"namespace"`
	ClusterName    string      `json:"clusterName,omitempty"`
	Phase          string      `json:"phase,omitempty"`
	Version        string      `json:"version,omitempty"`
	ProviderID     string      `json:"providerID,omitempty"`
	Ready          bool        `json:"ready"`
	Node           *NodeRef    `json:"node,omitempty"`
	Bootstrap      *ObjectRef  `json:"bootstrap,omitempty"`
	Infrastructure *ObjectRef  `json:"infrastructure,omitempty"`
	Addresses      []Address   `json:"addresses,omitempty"`
	Conditions     []Condition `json:"conditions,omitempty"`
}

// CreateGetMachineHandler creates a handler for getting detailed machine information
func CreateGetMachineHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			return nil, fmt.Errorf("name argument is required")
		}

		machine, err := capiClient.GetMachine(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get machine: %w", err)
		}

		detail := MachineDetail{
			Name:        machine.Name,
			Namespace:   machine.Namespace,
			ClusterName: machine.Spec.ClusterName,
			Phase:       machine.Status.Phase,
			Ready:       machineReady(machine),
			Bootstrap:   objectRef(machine.Spec.Bootstrap.ConfigRef),
			Addresses:   machineAddresses(machine.Status.Addresses),
			Conditions:  capiConditions(machine.Status.Conditions),
		}
		if machine.Spec.Version != nil {
			detail.Version = *machine.Spec.Version
		}
		if machine.Spec.ProviderID != nil {
			detail.ProviderID = *machine.Spec.ProviderID
		}
		if machine.Status.NodeRef != nil {
			detail.Node = &NodeRef{Name: machine.Status.NodeRef.Name, UID: string(machine.Status.NodeRef.UID)}
		}
		if machine.Spec.InfrastructureRef.Kind != "" {
			detail.Infrastructure = objectRef(&machine.Spec.InfrastructureRef)
		}

		return jsonResult(detail)
	}
}

// deleteMachineResult is the result of capi_delete_machine.
type deleteMachineResult struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Force     bool   `json:"force"`
	Message   string `json:"message"`
}

// CreateDeleteMachineHandler creates a handler for deleting CAPI machines
func CreateDeleteMachineHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
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

		err = capiClient.DeleteMachine(ctx, capi.DeleteMachineOptions{
			Namespace: namespace,
			Name:      name,
			Force:     force,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to delete machine: %v", err)), nil
		}

		return jsonResult(deleteMachineResult{
			Namespace: namespace,
			Name:      name,
			Force:     force,
			Message: "Machine deletion initiated. Deletion is asynchronous: the node is drained, the machine is " +
				"removed from the cluster and its infrastructure is cleaned up. Monitor with capi_get_machine.",
		})
	}
}

// remediateMachineResult is the result of capi_remediate_machine; phase and
// nodeName describe the machine before remediation.
type remediateMachineResult struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Phase     string `json:"phase,omitempty"`
	NodeName  string `json:"nodeName,omitempty"`
	Message   string `json:"message"`
}

// CreateRemediateMachineHandler creates a handler for triggering machine remediation
func CreateRemediateMachineHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
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
		machine, err := capiClient.GetMachine(ctx, namespace, name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get machine: %v", err)), nil
		}

		err = capiClient.RemediateMachine(ctx, capi.RemediateMachineOptions{
			Namespace: namespace,
			Name:      name,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to remediate machine: %v", err)), nil
		}

		result := remediateMachineResult{
			Namespace: namespace,
			Name:      name,
			Phase:     machine.Status.Phase,
			Message: "Remediation triggered. The MachineHealthCheck controller processes it according to the " +
				"remediation strategy (machine replacement, reboot or a custom remediation). Monitor with capi_get_machine.",
		}
		if machine.Status.NodeRef != nil {
			result.NodeName = machine.Status.NodeRef.Name
		}

		return jsonResult(result)
	}
}

// createMachineDeploymentResult is the result of capi_create_machinedeployment.
type createMachineDeploymentResult struct {
	Namespace       string    `json:"namespace"`
	Name            string    `json:"name"`
	ClusterName     string    `json:"clusterName"`
	Replicas        int32     `json:"replicas"`
	Version         string    `json:"version"`
	Infrastructure  ObjectRef `json:"infrastructure"`
	Bootstrap       ObjectRef `json:"bootstrap"`
	MinReadySeconds *int32    `json:"minReadySeconds,omitempty"`
	Message         string    `json:"message"`
}

// CreateCreateMachineDeploymentHandler creates a handler for creating new machine deployments
func CreateCreateMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
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

		replicas := int32(1)
		if r, ok := arguments["replicas"].(float64); ok {
			replicas = int32(r)
		}

		infraKind, _ := arguments["infra_kind"].(string)
		infraName, _ := arguments["infra_name"].(string)
		infraAPIVersion, _ := arguments["infra_api_version"].(string)

		if infraKind == "" || infraName == "" {
			return mcp.NewToolResultError("infra_kind and infra_name are required"), nil
		}

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

		md, err := capiClient.CreateMachineDeployment(ctx, capi.CreateMachineDeploymentOptions{
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

		return jsonResult(createMachineDeploymentResult{
			Namespace:       namespace,
			Name:            name,
			ClusterName:     clusterName,
			Replicas:        replicas,
			Version:         version,
			Infrastructure:  ObjectRef{Kind: infraKind, Name: infraName, APIVersion: infraAPIVersion},
			Bootstrap:       ObjectRef{Kind: bootstrapKind, Name: bootstrapName, APIVersion: bootstrapAPIVersion},
			MinReadySeconds: md.Spec.MinReadySeconds,
			Message: "MachineDeployment created. It requires an existing infrastructure template (e.g. AWSMachineTemplate) " +
				"and bootstrap config template (e.g. KubeadmConfigTemplate). Monitor with capi_list_machines; " +
				"scale with capi_scale_machinedeployment.",
		})
	}
}

// scaleMachineDeploymentResult is the result of capi_scale_machinedeployment.
type scaleMachineDeploymentResult struct {
	Namespace        string `json:"namespace"`
	Name             string `json:"name"`
	PreviousReplicas int32  `json:"previousReplicas"`
	Replicas         int32  `json:"replicas"`
	Message          string `json:"message"`
}

// CreateScaleMachineDeploymentHandler creates a handler for scaling machine deployments
func CreateScaleMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
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
		list, err := capiClient.ListMachineDeployments(ctx, namespace, "")
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

		err = capiClient.ScaleMachineDeployment(ctx, namespace, name, replicas)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to scale machine deployment: %v", err)), nil
		}

		var message string
		switch {
		case replicas > currentReplicas:
			message = fmt.Sprintf("Scaling up by %d machines: new machines are provisioned, bootstrapped and joined to the cluster. Monitor with capi_list_machines.", replicas-currentReplicas)
		case replicas < currentReplicas:
			message = fmt.Sprintf("Scaling down by %d machines: nodes are cordoned, drained, removed from the cluster and their infrastructure cleaned up. Monitor with capi_list_machines.", currentReplicas-replicas)
		default:
			message = "No change: the replica count is unchanged"
		}

		return jsonResult(scaleMachineDeploymentResult{
			Namespace:        namespace,
			Name:             name,
			PreviousReplicas: currentReplicas,
			Replicas:         replicas,
			Message:          message,
		})
	}
}

// updateMachineDeploymentResult is the result of capi_update_machinedeployment;
// it reflects the MachineDeployment after the update.
type updateMachineDeploymentResult struct {
	Namespace         string `json:"namespace"`
	Name              string `json:"name"`
	Replicas          *int32 `json:"replicas,omitempty"`
	Version           string `json:"version,omitempty"`
	MinReadySeconds   *int32 `json:"minReadySeconds,omitempty"`
	ReadyReplicas     int32  `json:"readyReplicas"`
	UpdatedReplicas   int32  `json:"updatedReplicas"`
	AvailableReplicas int32  `json:"availableReplicas"`
	Message           string `json:"message"`
}

// CreateUpdateMachineDeploymentHandler creates a handler for updating machine deployment configuration
func CreateUpdateMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			return nil, fmt.Errorf("name argument is required")
		}

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

		if labels, ok := arguments["labels"].(map[string]interface{}); ok {
			opts.Labels = make(map[string]string)
			for k, v := range labels {
				if strVal, ok := v.(string); ok {
					opts.Labels[k] = strVal
				}
			}
		}

		if annotations, ok := arguments["annotations"].(map[string]interface{}); ok {
			opts.Annotations = make(map[string]string)
			for k, v := range annotations {
				if strVal, ok := v.(string); ok {
					opts.Annotations[k] = strVal
				}
			}
		}

		md, err := capiClient.UpdateMachineDeployment(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update machine deployment: %v", err)), nil
		}

		result := updateMachineDeploymentResult{
			Namespace:         namespace,
			Name:              name,
			Replicas:          md.Spec.Replicas,
			MinReadySeconds:   md.Spec.MinReadySeconds,
			ReadyReplicas:     md.Status.ReadyReplicas,
			UpdatedReplicas:   md.Status.UpdatedReplicas,
			AvailableReplicas: md.Status.AvailableReplicas,
			Message:           "MachineDeployment updated",
		}
		if md.Spec.Template.Spec.Version != nil {
			result.Version = *md.Spec.Template.Spec.Version
		}

		return jsonResult(result)
	}
}

// rolloutMachineDeploymentResult is the result of capi_rollout_machinedeployment.
type rolloutMachineDeploymentResult struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Reason    string `json:"reason,omitempty"`
	Message   string `json:"message"`
}

// CreateRolloutMachineDeploymentHandler creates a handler for triggering machine deployment rollout
func CreateRolloutMachineDeploymentHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
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

		err = capiClient.RolloutMachineDeployment(ctx, capi.RolloutMachineDeploymentOptions{
			Namespace: namespace,
			Name:      name,
			Reason:    reason,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to trigger rollout: %v", err)), nil
		}

		return jsonResult(rolloutMachineDeploymentResult{
			Namespace: namespace,
			Name:      name,
			Reason:    reason,
			Message: "Rollout triggered. New machines are created with the updated configuration and old machines " +
				"are replaced according to the deployment's update strategy, with health checks between steps. " +
				"Monitor with capi_list_machines and capi_list_machinedeployments.",
		})
	}
}

// MachineSetSummary is one entry of capi_list_machinesets.
type MachineSetSummary struct {
	Name              string     `json:"name"`
	Namespace         string     `json:"namespace"`
	ClusterName       string     `json:"clusterName,omitempty"`
	Replicas          int32      `json:"replicas"`
	CurrentReplicas   int32      `json:"currentReplicas"`
	ReadyReplicas     int32      `json:"readyReplicas"`
	AvailableReplicas int32      `json:"availableReplicas"`
	Owner             *ObjectRef `json:"owner,omitempty"`
	Infrastructure    *ObjectRef `json:"infrastructure,omitempty"`
}

// CreateListMachineSetsHandler creates a handler for listing machine sets as
// {items: [MachineSetSummary]}.
func CreateListMachineSetsHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		clusterName, _ := arguments["clusterName"].(string)

		machineSets, err := capiClient.ListMachineSets(ctx, namespace, clusterName)
		if err != nil {
			return nil, fmt.Errorf("failed to list machine sets: %w", err)
		}

		items := make([]MachineSetSummary, 0, len(machineSets.Items))
		for i := range machineSets.Items {
			ms := &machineSets.Items[i]
			item := MachineSetSummary{
				Name:              ms.Name,
				Namespace:         ms.Namespace,
				ClusterName:       ms.Spec.ClusterName,
				CurrentReplicas:   ms.Status.Replicas,
				ReadyReplicas:     ms.Status.ReadyReplicas,
				AvailableReplicas: ms.Status.AvailableReplicas,
			}
			if ms.Spec.Replicas != nil {
				item.Replicas = *ms.Spec.Replicas
			}
			for _, owner := range ms.OwnerReferences {
				if owner.Kind == "MachineDeployment" {
					item.Owner = &ObjectRef{Kind: owner.Kind, Name: owner.Name}
					break
				}
			}
			if ms.Spec.Template.Spec.InfrastructureRef.Name != "" {
				item.Infrastructure = objectRef(&ms.Spec.Template.Spec.InfrastructureRef)
			}
			items = append(items, item)
		}

		return listResult(items)
	}
}

// MachineTemplate describes the machine template of a MachineSet.
type MachineTemplate struct {
	Version        string     `json:"version,omitempty"`
	Infrastructure *ObjectRef `json:"infrastructure,omitempty"`
	Bootstrap      *ObjectRef `json:"bootstrap,omitempty"`
}

// MachineSetDetail is the result of capi_get_machineset.
type MachineSetDetail struct {
	Name              string          `json:"name"`
	Namespace         string          `json:"namespace"`
	ClusterName       string          `json:"clusterName,omitempty"`
	Replicas          int32           `json:"replicas"`
	CurrentReplicas   int32           `json:"currentReplicas"`
	ReadyReplicas     int32           `json:"readyReplicas"`
	AvailableReplicas int32           `json:"availableReplicas"`
	FailureReason     string          `json:"failureReason,omitempty"`
	FailureMessage    string          `json:"failureMessage,omitempty"`
	Template          MachineTemplate `json:"template"`
	Owners            []ObjectRef     `json:"owners,omitempty"`
	Conditions        []Condition     `json:"conditions,omitempty"`
}

// CreateGetMachineSetHandler creates a handler for getting machine set details
func CreateGetMachineSetHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			return nil, fmt.Errorf("name argument is required")
		}

		ms, err := capiClient.GetMachineSet(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get machine set: %w", err)
		}

		detail := MachineSetDetail{
			Name:              ms.Name,
			Namespace:         ms.Namespace,
			ClusterName:       ms.Spec.ClusterName,
			CurrentReplicas:   ms.Status.Replicas,
			ReadyReplicas:     ms.Status.ReadyReplicas,
			AvailableReplicas: ms.Status.AvailableReplicas,
			Conditions:        capiConditions(ms.Status.Conditions),
		}
		if ms.Spec.Replicas != nil {
			detail.Replicas = *ms.Spec.Replicas
		}
		if ms.Status.FailureReason != nil { //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
			detail.FailureReason = string(*ms.Status.FailureReason) //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
		}
		if ms.Status.FailureMessage != nil { //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
			detail.FailureMessage = *ms.Status.FailureMessage //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
		}
		if ms.Spec.Template.Spec.Version != nil {
			detail.Template.Version = *ms.Spec.Template.Spec.Version
		}
		if ms.Spec.Template.Spec.InfrastructureRef.Name != "" {
			detail.Template.Infrastructure = objectRef(&ms.Spec.Template.Spec.InfrastructureRef)
		}
		detail.Template.Bootstrap = objectRef(ms.Spec.Template.Spec.Bootstrap.ConfigRef)
		for _, owner := range ms.OwnerReferences {
			detail.Owners = append(detail.Owners, ObjectRef{Kind: owner.Kind, Name: owner.Name})
		}

		return jsonResult(detail)
	}
}

// drainNodeResult is the result of capi_drain_node.
type drainNodeResult struct {
	NodeName      string `json:"nodeName,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	MachineName   string `json:"machineName,omitempty"`
	Cordoned      bool   `json:"cordoned"`
	Drained       bool   `json:"drained"`
	Message       string `json:"message"`
	ManualCommand string `json:"manualCommand,omitempty"`
}

// CreateDrainNodeHandler creates a handler for draining nodes
func CreateDrainNodeHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()

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

		opts.IgnoreDaemonSets, _ = arguments["ignore_daemonsets"].(bool)
		opts.DeleteLocalData, _ = arguments["delete_local_data"].(bool)
		opts.Force, _ = arguments["force"].(bool)

		if gracePeriodFloat, ok := arguments["grace_period_seconds"].(float64); ok {
			gracePeriod := int32(gracePeriodFloat)
			opts.GracePeriodSeconds = &gracePeriod
		}

		// DrainNode currently only cordons the node; pod eviction is not
		// implemented yet, so report the partial result.
		if err := capiClient.DrainNode(ctx, opts); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to drain node: %v", err)), nil
		}

		result := drainNodeResult{
			NodeName:    nodeName,
			Namespace:   namespace,
			MachineName: machineName,
			Cordoned:    true,
			Drained:     false,
			Message: "Drain is partially implemented: the node was cordoned (marked unschedulable) but pods were " +
				"not evicted. Evict them with kubectl drain, respecting PodDisruptionBudgets.",
		}
		if nodeName != "" {
			result.ManualCommand = fmt.Sprintf("kubectl drain %s --ignore-daemonsets --delete-emptydir-data", nodeName)
		}

		return jsonResult(result)
	}
}

// cordonNodeResult is the result of capi_cordon_node.
type cordonNodeResult struct {
	NodeName      string `json:"nodeName,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	MachineName   string `json:"machineName,omitempty"`
	Unschedulable bool   `json:"unschedulable"`
	Message       string `json:"message"`
}

// CreateCordonNodeHandler creates a handler for cordoning/uncordoning nodes
func CreateCordonNodeHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()

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

		err = capiClient.CordonNode(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update node: %v", err)), nil
		}

		message := "Node cordoned: no new pods are scheduled on it while existing pods keep running. Use capi_drain_node to evict pods."
		if opts.Uncordon {
			message = "Node uncordoned: it is schedulable again and accepts new workloads."
		}

		return jsonResult(cordonNodeResult{
			NodeName:      nodeName,
			Namespace:     namespace,
			MachineName:   machineName,
			Unschedulable: !opts.Uncordon,
			Message:       message,
		})
	}
}

// NodeInfo is the digest of a node's system information.
type NodeInfo struct {
	OperatingSystem         string `json:"operatingSystem,omitempty"`
	OSImage                 string `json:"osImage,omitempty"`
	KernelVersion           string `json:"kernelVersion,omitempty"`
	ContainerRuntimeVersion string `json:"containerRuntimeVersion,omitempty"`
	KubeletVersion          string `json:"kubeletVersion,omitempty"`
	Architecture            string `json:"architecture,omitempty"`
}

// NodeResources lists the CPU, memory and pod quantities of a node.
type NodeResources struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
	Pods   string `json:"pods,omitempty"`
}

// Taint is a node taint.
type Taint struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Effect string `json:"effect"`
}

// NodeStatus is the result of capi_node_status.
type NodeStatus struct {
	Name        string        `json:"name"`
	UID         string        `json:"uid"`
	CreatedAt   string        `json:"createdAt"`
	Schedulable bool          `json:"schedulable"`
	ProviderID  string        `json:"providerID,omitempty"`
	NodeInfo    NodeInfo      `json:"nodeInfo"`
	Capacity    NodeResources `json:"capacity"`
	Allocatable NodeResources `json:"allocatable"`
	Conditions  []Condition   `json:"conditions,omitempty"`
	Addresses   []Address     `json:"addresses,omitempty"`
	Taints      []Taint       `json:"taints,omitempty"`
}

// nodeResources converts a resource list into its digest; zero quantities are omitted.
func nodeResources(list v1.ResourceList) NodeResources {
	var out NodeResources
	if cpu := list[v1.ResourceCPU]; !cpu.IsZero() {
		out.CPU = cpu.String()
	}
	if memory := list[v1.ResourceMemory]; !memory.IsZero() {
		out.Memory = memory.String()
	}
	if pods := list[v1.ResourcePods]; !pods.IsZero() {
		out.Pods = pods.String()
	}
	return out
}

// CreateNodeStatusHandler creates a handler for getting node status
func CreateNodeStatusHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()

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

		node, err := capiClient.GetNodeStatus(ctx, opts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get node status: %v", err)), nil
		}

		info := node.Status.NodeInfo
		status := NodeStatus{
			Name:        node.Name,
			UID:         string(node.UID),
			CreatedAt:   node.CreationTimestamp.UTC().Format(time.RFC3339),
			Schedulable: !node.Spec.Unschedulable,
			ProviderID:  node.Spec.ProviderID,
			NodeInfo: NodeInfo{
				OperatingSystem:         info.OperatingSystem,
				OSImage:                 info.OSImage,
				KernelVersion:           info.KernelVersion,
				ContainerRuntimeVersion: info.ContainerRuntimeVersion,
				KubeletVersion:          info.KubeletVersion,
				Architecture:            info.Architecture,
			},
			Capacity:    nodeResources(node.Status.Capacity),
			Allocatable: nodeResources(node.Status.Allocatable),
			Conditions:  nodeConditions(node.Status.Conditions),
			Addresses:   nodeAddresses(node.Status.Addresses),
		}
		for _, taint := range node.Spec.Taints {
			status.Taints = append(status.Taints, Taint{Key: taint.Key, Value: taint.Value, Effect: string(taint.Effect)})
		}

		return jsonResult(status)
	}
}
