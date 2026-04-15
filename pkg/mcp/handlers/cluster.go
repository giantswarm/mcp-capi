// Package handlers provides MCP tool handler functions for CAPI cluster operations.
package handlers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
)

// CreateCreateClusterHandler creates a handler for creating new CAPI clusters
func CreateCreateClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()

		// Required parameters
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			return nil, errors.New("name argument is required")
		}
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, errors.New("namespace argument is required")
		}
		provider, ok := arguments["provider"].(string)
		if !ok || provider == "" {
			return nil, errors.New("provider argument is required")
		}

		// Validate provider
		validProviders := []string{"aws", "azure", "gcp", "vsphere"}
		isValidProvider := slices.Contains(validProviders, provider)
		if !isValidProvider {
			return nil, fmt.Errorf(
				"invalid provider %s. Must be one of: %s",
				provider,
				strings.Join(validProviders, ", "),
			)
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
		cluster, err := serverCtx.CAPIClient.CreateCluster(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster: %w", err)
		}

		text := formatCreateClusterDetails(
			name, cluster.Name, cluster.Namespace,
			provider, kubernetesVersion,
			controlPlaneCount, workerCount,
			region, instanceType,
		)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: text,
				},
			},
		}, nil
	}
}

// CreateListClustersHandler creates a handler for listing CAPI clusters
func CreateListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, _ := arguments["namespace"].(string)
		search, _ := arguments["search"].(string)

		// Parse label_selector from arguments
		labelSelector := parseLabelSelector(arguments)

		clusters, err := serverCtx.CAPIClient.ListClusters(ctx, namespace, labelSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		// If a search term is provided, filter clusters by name or label values
		if search != "" {
			clusters.Items = filterClustersBySearch(clusters.Items, search)
		}

		// Bulk fetch all machines in the namespace to avoid N+1 queries
		allMachines, err := serverCtx.CAPIClient.ListMachines(ctx, namespace, "")
		if err != nil {
			return nil, fmt.Errorf("failed to list machines: %w", err)
		}

		// Group machines by cluster name for efficient lookup
		machinesByCluster := groupMachinesByCluster(allMachines.Items)

		var content strings.Builder
		fmt.Fprintf(&content, "Found %d clusters:\n\n", len(clusters.Items))

		for i := range clusters.Items {
			cluster := &clusters.Items[i]
			key := cluster.Namespace + "/" + cluster.Name
			status, _ := serverCtx.CAPIClient.GetClusterStatusFromList(ctx, cluster, machinesByCluster[key])
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

// CreateGetClusterHandler creates a handler for getting a specific cluster
func CreateGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Try exact name match first
		status, err := serverCtx.CAPIClient.GetClusterStatus(ctx, namespace, name)
		if err == nil {
			return textResult(capi.FormatClusterInfo(status)), nil
		}

		// If exact name match failed, fall back to label-value search
		return serverCtx.getClusterByLabelFallback(ctx, namespace, name)
	}
}

// getClusterByLabelFallback searches for clusters whose label values match name
// and returns an appropriate response when no exact name match exists.
func (s *ServerContext) getClusterByLabelFallback(
	ctx context.Context, namespace, name string,
) (*mcp.CallToolResult, error) {
	matched, labelErr := s.CAPIClient.FindClustersByLabelValue(ctx, namespace, name)
	if labelErr != nil || len(matched.Items) == 0 {
		return nil, fmt.Errorf(
			"failed to get cluster %q: no cluster found by name or label value in namespace %s",
			name,
			namespace,
		)
	}

	if len(matched.Items) == 1 {
		cluster := matched.Items[0]
		status, err := s.CAPIClient.GetClusterStatus(ctx, cluster.Namespace, cluster.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster status: %w", err)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "Note: No cluster named %q found. Matched cluster by label value:\n\n", name)
		b.WriteString(capi.FormatClusterInfo(status))
		return textResult(b.String()), nil
	}

	// Multiple matches - list them for the user to disambiguate
	var b strings.Builder
	fmt.Fprintf(&b, "No cluster named %q found, but %d clusters matched the term in their labels:\n\n",
		name, len(matched.Items),
	)
	for i := range matched.Items {
		cluster := &matched.Items[i]
		status, err := s.CAPIClient.GetClusterStatus(ctx, cluster.Namespace, cluster.Name)
		if err == nil {
			b.WriteString(capi.FormatClusterInfo(status))
			b.WriteString("\n---\n\n")
		}
	}
	b.WriteString("Please specify the exact cluster name from the list above.")
	return textResult(b.String()), nil
}

// CreateClusterStatusHandler creates a handler for getting detailed cluster status
func CreateClusterStatusHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		status, err := serverCtx.CAPIClient.GetClusterStatus(ctx, namespace, name)
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

// CreateClusterHealthHandler creates a handler for checking cluster health
func CreateClusterHealthHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		health, err := serverCtx.CAPIClient.GetClusterHealth(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster health: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: formatHealthReport(namespace, name, health),
				},
			},
		}, nil
	}
}

// formatCreateClusterDetails formats the cluster creation success response.
func formatCreateClusterDetails(
	name, clusterName, namespace, provider, kubernetesVersion string,
	controlPlaneCount, workerCount int32,
	region, instanceType string,
) string {
	var b strings.Builder
	fmt.Fprintf(&b, "✅ Cluster '%s' creation initiated successfully!\n\n", name)
	b.WriteString("Cluster Details:\n")
	fmt.Fprintf(&b, "  Name: %s\n", clusterName)
	fmt.Fprintf(&b, "  Namespace: %s\n", namespace)
	fmt.Fprintf(&b, "  Provider: %s\n", provider)
	fmt.Fprintf(&b, "  Kubernetes Version: %s\n", kubernetesVersion)
	fmt.Fprintf(&b, "  Control Plane Nodes: %d\n", controlPlaneCount)
	fmt.Fprintf(&b, "  Worker Nodes: %d\n", workerCount)
	if region != "" {
		fmt.Fprintf(&b, "  Region: %s\n", region)
	}
	if instanceType != "" {
		fmt.Fprintf(&b, "  Instance Type: %s\n", instanceType)
	}
	b.WriteString("\n⚠️  Note: This is a basic implementation that creates only the Cluster resource.\n")
	b.WriteString("In a production setup, you would need to:\n")
	b.WriteString("1. Create the infrastructure-specific cluster resource (e.g., AWSCluster)\n")
	b.WriteString("2. Create the control plane (e.g., KubeadmControlPlane)\n")
	b.WriteString("3. Create machine deployments for worker nodes\n")
	b.WriteString("4. Configure networking, storage, and other cluster settings\n\n")
	b.WriteString("Monitor cluster creation with: capi_cluster_status\n")
	return b.String()
}

// textResult wraps a plain text string in a CallToolResult.
func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}

// parseLabelSelector extracts a string label selector map from the raw MCP arguments map.
func parseLabelSelector(arguments map[string]any) map[string]string {
	ls, ok := arguments["label_selector"].(map[string]any)
	if !ok || len(ls) == 0 {
		return nil
	}
	result := make(map[string]string, len(ls))
	for k, v := range ls {
		if strVal, ok := v.(string); ok {
			result[k] = strVal
		}
	}
	return result
}

// groupMachinesByCluster groups a slice of machines by "namespace/clusterName" key.
func groupMachinesByCluster(machines []clusterv1.Machine) map[string][]clusterv1.Machine {
	result := make(map[string][]clusterv1.Machine)
	for i := range machines {
		m := &machines[i]
		clusterName := m.Labels[clusterv1.ClusterNameLabel]
		key := m.Namespace + "/" + clusterName
		result[key] = append(result[key], *m)
	}
	return result
}

// filterClustersBySearch returns a filtered slice of clusters whose name or label
// values contain the given search term (case-insensitive).
func filterClustersBySearch(items []clusterv1.Cluster, search string) []clusterv1.Cluster {
	searchLower := strings.ToLower(search)
	var filtered []clusterv1.Cluster
	for i := range items {
		c := &items[i]
		if strings.Contains(strings.ToLower(c.Name), searchLower) {
			filtered = append(filtered, *c)
			continue
		}
		for _, v := range c.Labels {
			if strings.Contains(strings.ToLower(v), searchLower) {
				filtered = append(filtered, *c)
				break
			}
		}
	}
	return filtered
}

// formatHealthReport formats the health check response for a cluster.
func formatHealthReport(namespace, name string, health *capi.ClusterHealthStatus) string {
	var b strings.Builder
	if health.Healthy {
		fmt.Fprintf(&b, "✅ Cluster %s/%s is HEALTHY\n\n", namespace, name)
	} else {
		fmt.Fprintf(&b, "❌ Cluster %s/%s is UNHEALTHY\n\n", namespace, name)
	}
	b.WriteString("Component Status:\n")
	fmt.Fprintf(&b, "  • Control Plane: %s\n", formatHealthStatus(health.ControlPlaneReady))
	fmt.Fprintf(&b, "  • Infrastructure: %s\n", formatHealthStatus(health.InfraReady))
	fmt.Fprintf(&b, "  • Worker Nodes: %s\n", formatHealthStatus(health.WorkersReady))
	if len(health.Issues) > 0 {
		b.WriteString("\n🔴 Issues:\n")
		for _, issue := range health.Issues {
			fmt.Fprintf(&b, "  • %s\n", issue)
		}
	}
	if len(health.Warnings) > 0 {
		b.WriteString("\n⚠️  Warnings:\n")
		for _, warning := range health.Warnings {
			fmt.Fprintf(&b, "  • %s\n", warning)
		}
	}
	if !health.Healthy {
		b.WriteString("\n📋 Recommendations:\n")
		if !health.ControlPlaneReady {
			b.WriteString("  • Check control plane pods and logs\n")
			b.WriteString("  • Verify API server connectivity\n")
		}
		if !health.InfraReady {
			b.WriteString("  • Check infrastructure provider status\n")
			b.WriteString("  • Verify cloud resources are provisioned\n")
		}
		if !health.WorkersReady {
			b.WriteString("  • Check machine status with 'capi_list_machines'\n")
			b.WriteString("  • Review machine deployment events\n")
		}
	}
	return b.String()
}

// formatMetadataMap formats a label or annotation map as a string, one entry per line.
func formatMetadataMap(b *strings.Builder, entries map[string]string) {
	if len(entries) > 0 {
		for k, v := range entries {
			fmt.Fprintf(b, "  %s: %s\n", k, v)
		}
	} else {
		b.WriteString("  (none)\n")
	}
}

// formatMetadataChanges formats what labels or annotations were added/removed.
func formatMetadataChanges(b *strings.Builder, section string, changes map[string]string) {
	if len(changes) == 0 {
		return
	}
	fmt.Fprintf(b, "%s updated:\n", section)
	for k, v := range changes {
		if v == "" {
			fmt.Fprintf(b, "  ✗ Removed: %s\n", k)
		} else {
			fmt.Fprintf(b, "  ✓ Set: %s=%s\n", k, v)
		}
	}
	b.WriteString("\n")
}

// formatHealthStatus returns a formatted string for component health status
func formatHealthStatus(ready bool) string {
	if ready {
		return "✅ Ready"
	}
	return "❌ Not Ready"
}

// CreateScaleClusterHandler creates a handler for scaling clusters
func CreateScaleClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
		target, ok := arguments["target"].(string)
		if !ok || target == "" {
			return nil, errors.New("target argument is required")
		}
		replicas, ok := arguments["replicas"].(float64)
		if !ok {
			return nil, errors.New("replicas argument is required and must be a number")
		}
		machineDeployment, _ := arguments["machineDeployment"].(string)

		err := serverCtx.CAPIClient.ScaleCluster(ctx, namespace, name, target, int(replicas), machineDeployment)
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

// CreateGetKubeconfigHandler creates a handler for retrieving cluster kubeconfig
func CreateGetKubeconfigHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		kubeconfig, err := serverCtx.CAPIClient.GetKubeconfig(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
		}

		var content strings.Builder
		fmt.Fprintf(&content, "Kubeconfig for cluster %s/%s:\n\n", namespace, name)
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

// pauseResumeAction is a function that pauses or resumes a cluster.
type pauseResumeAction func(ctx context.Context, namespace, name string) error

// clusterPauseResumeHandler is a shared helper for pause and resume handlers.
// actionFn performs the actual pause or resume operation. statusVerb is the past-tense
// verb for the success line (e.g. "paused" or "resumed"). details are the body lines
// printed after the status line.
func clusterPauseResumeHandler(
	actionName string,
	actionFn pauseResumeAction,
	statusVerb string,
	details []string,
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

		if err := actionFn(ctx, namespace, name); err != nil {
			return nil, fmt.Errorf("failed to %s cluster: %w", actionName, err)
		}

		var content strings.Builder
		fmt.Fprintf(&content, "✅ Cluster %s/%s has been %s\n\n", namespace, name, statusVerb)
		for _, line := range details {
			content.WriteString(line)
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

// CreatePauseClusterHandler creates a handler for pausing cluster reconciliation
func CreatePauseClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return clusterPauseResumeHandler(
		"pause",
		serverCtx.CAPIClient.PauseCluster,
		"paused",
		[]string{
			"The cluster reconciliation has been stopped. This means:\n",
			"- CAPI controllers will not make any changes to the cluster\n",
			"- The cluster will not be updated or scaled automatically\n",
			"- Manual operations can be performed safely\n\n",
			"To resume normal operations, use the capi_resume_cluster tool.",
		},
	)
}

// CreateResumeClusterHandler creates a handler for resuming cluster reconciliation
func CreateResumeClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return clusterPauseResumeHandler(
		"resume",
		serverCtx.CAPIClient.ResumeCluster,
		"resumed",
		[]string{
			"The cluster reconciliation has been restarted. This means:\n",
			"- CAPI controllers will now reconcile the cluster normally\n",
			"- Any pending updates or changes will be applied\n",
			"- Automatic scaling and updates are re-enabled\n\n",
			"The cluster is now under normal CAPI management.",
		},
	)
}

// CreateDeleteClusterHandler creates a handler for deleting a cluster
func CreateDeleteClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Get cluster status first to show what will be deleted
		status, err := serverCtx.CAPIClient.GetClusterStatus(ctx, namespace, name)
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
		err = serverCtx.CAPIClient.DeleteCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to delete cluster: %w", err)
		}

		fmt.Fprintf(&content, "\n✅ Cluster %s/%s deletion initiated successfully.\n\n", namespace, name)
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

// CreateUpgradeClusterHandler creates a handler for upgrading cluster Kubernetes version
func CreateUpgradeClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
		targetVersion, ok := arguments["target_version"].(string)
		if !ok || targetVersion == "" {
			return nil, errors.New("target_version argument is required")
		}

		// Default to upgrading workers
		upgradeWorkers := true
		if uw, ok := arguments["upgrade_workers"].(bool); ok {
			upgradeWorkers = uw
		}

		// Get current cluster status
		status, err := serverCtx.CAPIClient.GetClusterStatus(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster status: %w", err)
		}

		var content strings.Builder
		fmt.Fprintf(&content, "🚀 Initiating cluster upgrade for %s/%s\n\n", namespace, name)
		content.WriteString("Current State:\n")
		fmt.Fprintf(&content, "  • Current Version: %s\n", status.Version)
		fmt.Fprintf(&content, "  • Target Version: %s\n", targetVersion)
		fmt.Fprintf(&content, "  • Upgrade Workers: %v\n\n", upgradeWorkers)

		// Perform the upgrade
		opts := capi.UpgradeClusterOptions{
			Namespace:      namespace,
			Name:           name,
			TargetVersion:  targetVersion,
			UpgradeWorkers: upgradeWorkers,
		}

		if err := serverCtx.CAPIClient.UpgradeCluster(ctx, opts); err != nil {
			return nil, fmt.Errorf("failed to upgrade cluster: %w", err)
		}

		content.WriteString("✅ Upgrade initiated successfully!\n\n")
		content.WriteString("Upgrade Process:\n")
		content.WriteString("1. Control plane nodes will be upgraded first (one by one)\n")
		if upgradeWorkers {
			content.WriteString("2. Worker nodes will be upgraded after control plane is ready\n")
		} else {
			content.WriteString("2. Worker nodes will NOT be upgraded (upgrade_workers=false)\n")
		}
		content.WriteString("\n⚠️  Important Notes:\n")
		content.WriteString("• The upgrade process can take 30-60 minutes depending on cluster size\n")
		content.WriteString("• Control plane will remain available during rolling upgrade\n")
		content.WriteString("• Workloads may be rescheduled during worker node upgrades\n")
		content.WriteString("• Monitor progress with: capi_cluster_status\n")
		content.WriteString("\n📋 Recommended Actions:\n")
		content.WriteString("1. Monitor cluster health: capi_cluster_health\n")
		content.WriteString("2. Watch control plane: capi_list_machines\n")
		content.WriteString("3. Check events for any issues\n")
		content.WriteString("4. Verify workloads after upgrade completes\n")

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

// CreateUpdateClusterHandler creates a handler for updating cluster metadata
func CreateUpdateClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Get labels and annotations from arguments
		labels, _ := arguments["labels"].(map[string]any)
		annotations, _ := arguments["annotations"].(map[string]any)

		// Convert interface{} maps to string maps
		labelMap := make(map[string]string)
		for k, v := range labels {
			if strVal, ok := v.(string); ok {
				labelMap[k] = strVal
			}
		}

		annotationMap := make(map[string]string)
		for k, v := range annotations {
			if strVal, ok := v.(string); ok {
				annotationMap[k] = strVal
			}
		}

		// Update the cluster
		opts := capi.UpdateClusterOptions{
			Namespace:   namespace,
			Name:        name,
			Labels:      labelMap,
			Annotations: annotationMap,
		}

		cluster, err := serverCtx.CAPIClient.UpdateCluster(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to update cluster: %w", err)
		}

		var content strings.Builder
		fmt.Fprintf(&content, "✅ Cluster %s/%s updated successfully!\n\n", namespace, name)
		formatMetadataChanges(&content, "Labels", labelMap)
		formatMetadataChanges(&content, "Annotations", annotationMap)

		// Show current metadata
		content.WriteString("Current metadata:\n")
		content.WriteString("Labels:\n")
		formatMetadataMap(&content, cluster.Labels)
		content.WriteString("\nAnnotations:\n")
		formatMetadataMap(&content, cluster.Annotations)

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

// CreateMoveClusterHandler creates a handler for moving clusters between management clusters
func CreateMoveClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		targetKubeconfig, _ := arguments["target_kubeconfig"].(string)
		targetNamespace, _ := arguments["target_namespace"].(string)
		dryRun, _ := arguments["dry_run"].(bool)

		// Prepare move options
		opts := capi.MoveClusterOptions{
			Namespace:        namespace,
			Name:             name,
			TargetKubeconfig: targetKubeconfig,
			TargetNamespace:  targetNamespace,
			DryRun:           dryRun,
		}

		// Get move instructions/manifest
		manifest, err := serverCtx.CAPIClient.MoveCluster(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare cluster move: %w", err)
		}

		var content strings.Builder
		fmt.Fprintf(&content, "🚀 Cluster Move Preparation for %s/%s\n\n", namespace, name)

		if dryRun {
			content.WriteString("⚠️  DRY RUN MODE - No actual changes will be made\n\n")
		}

		content.WriteString("📋 Move Instructions:\n")
		content.WriteString("1. Ensure target management cluster is ready\n")
		content.WriteString("2. Install required providers on target cluster\n")
		content.WriteString("3. Create target namespace if needed\n")
		content.WriteString("4. Use clusterctl to perform the move:\n\n")

		content.WriteString("```bash\n")
		content.WriteString("# Pause the cluster first\n")
		fmt.Fprintf(&content,
			"kubectl patch cluster %s -n %s --type merge -p '{\"spec\":{\"paused\":true}}'\n\n",
			name,
			namespace,
		)

		content.WriteString("# Move the cluster\n")
		if targetKubeconfig != "" {
			content.WriteString("clusterctl move --to-kubeconfig=" + targetKubeconfig)
		} else {
			content.WriteString("clusterctl move --to-kubeconfig=<target-kubeconfig>")
		}
		if targetNamespace != "" && targetNamespace != namespace {
			fmt.Fprintf(&content, " --namespace %s --to-namespace %s", namespace, targetNamespace)
		} else {
			content.WriteString(" --namespace " + namespace)
		}
		content.WriteString("\n")
		content.WriteString("```\n\n")

		content.WriteString("⚠️  Important Notes:\n")
		content.WriteString("• The source cluster will be paused during move\n")
		content.WriteString("• All cluster resources will be migrated\n")
		content.WriteString("• Ensure network connectivity between clusters\n")
		content.WriteString("• Verify provider versions match\n\n")

		content.WriteString("📝 Move Manifest Preview:\n")
		content.WriteString("```yaml\n")
		content.WriteString(manifest)
		content.WriteString("\n```\n")

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

// CreateBackupClusterHandler creates a handler for backing up cluster configurations
func CreateBackupClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		includeSecrets, _ := arguments["include_secrets"].(bool)
		outputFormat, _ := arguments["output_format"].(string)
		if outputFormat == "" {
			outputFormat = "yaml"
		}

		// Create backup
		opts := capi.BackupClusterOptions{
			Namespace:      namespace,
			Name:           name,
			IncludeSecrets: includeSecrets,
			OutputFormat:   outputFormat,
		}

		backup, err := serverCtx.CAPIClient.BackupCluster(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster backup: %w", err)
		}

		var content strings.Builder
		fmt.Fprintf(&content, "📦 Cluster Backup for %s/%s\n\n", namespace, name)

		content.WriteString("Backup Configuration:\n")
		fmt.Fprintf(&content, "  • Format: %s\n", outputFormat)
		fmt.Fprintf(&content, "  • Include Secrets: %v\n\n", includeSecrets)

		content.WriteString("📋 Backup Instructions:\n")
		content.WriteString("1. Save the backup content below to a file\n")
		content.WriteString("2. Store in a secure location (git, S3, etc.)\n")
		content.WriteString("3. Test restore procedure in a non-production environment\n\n")

		content.WriteString("🔧 Recommended Backup Tools:\n")
		content.WriteString("• Velero - Complete cluster backup solution\n")
		content.WriteString("  velero backup create <backup-name> --include-namespaces=<namespace>\n")
		content.WriteString("• etcd snapshot - For control plane state\n")
		content.WriteString("• Git repositories - For GitOps managed clusters\n\n")

		content.WriteString("⚠️  Important Notes:\n")
		content.WriteString("• This backup includes CAPI resources only\n")
		content.WriteString("• Workload data is NOT included\n")
		content.WriteString("• Infrastructure provider resources may need separate backup\n")
		if includeSecrets {
			content.WriteString("• ⚠️  Secrets are included - handle with care!\n")
		}
		content.WriteString("\n")

		content.WriteString("📄 Backup Content:\n")
		content.WriteString("```" + outputFormat + "\n")
		content.WriteString(backup)
		content.WriteString("\n```\n\n")

		content.WriteString("💾 To save this backup:\n")
		content.WriteString("1. Copy the content between the ``` markers\n")
		fmt.Fprintf(&content, "2. Save to a file: cluster-%s-%s-backup.%s\n", namespace, name, outputFormat)
		content.WriteString("3. Encrypt if it contains secrets\n")

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
