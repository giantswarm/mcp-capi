package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration

	"github.com/giantswarm/mcp-capi/pkg/capi"
)

// createClusterResult is the result of capi_create_cluster.
type createClusterResult struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	Provider          string `json:"provider"`
	KubernetesVersion string `json:"kubernetesVersion"`
	ControlPlaneCount int32  `json:"controlPlaneCount"`
	WorkerCount       int32  `json:"workerCount"`
	Region            string `json:"region,omitempty"`
	InstanceType      string `json:"instanceType,omitempty"`
	Message           string `json:"message"`
}

// CreateCreateClusterHandler creates a handler for creating new CAPI clusters
func CreateCreateClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
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
		validProviders := []string{string(capi.ProviderAWS), string(capi.ProviderAzure), string(capi.ProviderGCP), string(capi.ProviderVSphere)}
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

		cluster, err := capiClient.CreateCluster(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster: %w", err)
		}

		return jsonResult(createClusterResult{
			Name:              cluster.Name,
			Namespace:         cluster.Namespace,
			Provider:          provider,
			KubernetesVersion: kubernetesVersion,
			ControlPlaneCount: controlPlaneCount,
			WorkerCount:       workerCount,
			Region:            region,
			InstanceType:      instanceType,
			Message: "Cluster resource created. This basic implementation creates only the Cluster object; " +
				"the infrastructure cluster (e.g. AWSCluster), the control plane (e.g. KubeadmControlPlane) and " +
				"MachineDeployments for workers must be created separately. Monitor with capi_cluster_status.",
		})
	}
}

// CreateListClustersHandler creates a handler for listing CAPI clusters as
// {items: [ClusterSummary]}.
func CreateListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()
		namespace, _ := arguments["namespace"].(string)
		search, _ := arguments["search"].(string)

		// Parse label_selector from arguments
		var labelSelector map[string]string
		if ls, ok := arguments["label_selector"].(map[string]interface{}); ok && len(ls) > 0 {
			labelSelector = make(map[string]string)
			for k, v := range ls {
				if strVal, ok := v.(string); ok {
					labelSelector[k] = strVal
				}
			}
		}

		clusters, err := capiClient.ListClusters(ctx, namespace, labelSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		// If a search term is provided, filter clusters by name or label values
		if search != "" {
			searchLower := strings.ToLower(search)
			var filtered []clusterv1.Cluster
			for _, cluster := range clusters.Items {
				if strings.Contains(strings.ToLower(cluster.Name), searchLower) {
					filtered = append(filtered, cluster)
					continue
				}
				matched := false
				for _, v := range cluster.Labels {
					if strings.Contains(strings.ToLower(v), searchLower) {
						matched = true
						break
					}
				}
				if matched {
					filtered = append(filtered, cluster)
				}
			}
			clusters.Items = filtered
		}

		// Bulk fetch all machines in the namespace to avoid N+1 queries
		allMachines, err := capiClient.ListMachines(ctx, namespace, "")
		if err != nil {
			return nil, fmt.Errorf("failed to list machines: %w", err)
		}

		// Group machines by cluster name
		machinesByCluster := make(map[string][]clusterv1.Machine)
		for _, m := range allMachines.Items {
			clusterName := m.Labels[clusterv1.ClusterNameLabel]
			key := m.Namespace + "/" + clusterName
			machinesByCluster[key] = append(machinesByCluster[key], m)
		}

		items := make([]ClusterSummary, 0, len(clusters.Items))
		for i := range clusters.Items {
			cluster := &clusters.Items[i]
			key := cluster.Namespace + "/" + cluster.Name
			status, _ := capiClient.GetClusterStatusFromList(ctx, cluster, machinesByCluster[key])
			if status != nil {
				items = append(items, clusterSummary(status))
			}
		}

		return listResult(items)
	}
}

// getClusterResult is the result of capi_get_cluster. When one cluster
// resolves, its summary is inlined; when the name only matched label values
// on several clusters, they are listed as candidates instead.
type getClusterResult struct {
	*ClusterSummary
	MatchedBy  string           `json:"matchedBy,omitempty"`
	Note       string           `json:"note,omitempty"`
	Candidates []ClusterSummary `json:"candidates,omitempty"`
}

// CreateGetClusterHandler creates a handler for getting a specific cluster
func CreateGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Try exact name match first
		status, err := capiClient.GetClusterStatus(ctx, namespace, name)
		if err == nil {
			summary := clusterSummary(status)
			return jsonResult(getClusterResult{ClusterSummary: &summary})
		}

		// If exact name match failed, try matching against label values
		matched, labelErr := capiClient.FindClustersByLabelValue(ctx, namespace, name)
		if labelErr != nil || len(matched.Items) == 0 {
			// Return the original error if label search also fails
			return nil, fmt.Errorf("failed to get cluster %q: no cluster found by name or label value in namespace %s", name, namespace)
		}

		if len(matched.Items) == 1 {
			// Single match found via labels - return its status
			cluster := matched.Items[0]
			status, err := capiClient.GetClusterStatus(ctx, cluster.Namespace, cluster.Name)
			if err != nil {
				return nil, fmt.Errorf("failed to get cluster status: %w", err)
			}
			summary := clusterSummary(status)
			return jsonResult(getClusterResult{
				ClusterSummary: &summary,
				MatchedBy:      "labelValue",
				Note:           fmt.Sprintf("No cluster named %q found; matched cluster %s by label value.", name, cluster.Name),
			})
		}

		// Multiple matches - list them for the user to disambiguate
		candidates := make([]ClusterSummary, 0, len(matched.Items))
		for _, cluster := range matched.Items {
			status, err := capiClient.GetClusterStatus(ctx, cluster.Namespace, cluster.Name)
			if err == nil {
				candidates = append(candidates, clusterSummary(status))
			}
		}

		return jsonResult(getClusterResult{
			MatchedBy:  "labelValue",
			Note:       fmt.Sprintf("No cluster named %q found; %d clusters matched the term in their labels. Specify the exact cluster name.", name, len(matched.Items)),
			Candidates: candidates,
		})
	}
}

// CreateClusterStatusHandler creates a handler for getting detailed cluster status
func CreateClusterStatusHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		status, err := capiClient.GetClusterStatus(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster status: %w", err)
		}

		return jsonResult(clusterSummary(status))
	}
}

// clusterHealthResult is the result of capi_cluster_health.
type clusterHealthResult struct {
	Namespace           string   `json:"namespace"`
	Name                string   `json:"name"`
	Healthy             bool     `json:"healthy"`
	ControlPlaneReady   bool     `json:"controlPlaneReady"`
	InfrastructureReady bool     `json:"infrastructureReady"`
	WorkersReady        bool     `json:"workersReady"`
	Issues              []string `json:"issues"`
	Warnings            []string `json:"warnings"`
	Recommendations     []string `json:"recommendations,omitempty"`
}

// CreateClusterHealthHandler creates a handler for checking cluster health
func CreateClusterHealthHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		health, err := capiClient.GetClusterHealth(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster health: %w", err)
		}

		result := clusterHealthResult{
			Namespace:           namespace,
			Name:                name,
			Healthy:             health.Healthy,
			ControlPlaneReady:   health.ControlPlaneReady,
			InfrastructureReady: health.InfraReady,
			WorkersReady:        health.WorkersReady,
			Issues:              nonNilStrings(health.Issues),
			Warnings:            nonNilStrings(health.Warnings),
		}

		if !health.Healthy {
			if !health.ControlPlaneReady {
				result.Recommendations = append(result.Recommendations,
					"Check control plane pods and logs",
					"Verify API server connectivity")
			}
			if !health.InfraReady {
				result.Recommendations = append(result.Recommendations,
					"Check infrastructure provider status",
					"Verify cloud resources are provisioned")
			}
			if !health.WorkersReady {
				result.Recommendations = append(result.Recommendations,
					"Check machine status with capi_list_machines",
					"Review machine deployment events")
			}
		}

		return jsonResult(result)
	}
}

// nonNilStrings returns s, or an empty slice when s is nil, so the field
// encodes as [] rather than null.
func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// scaleClusterResult is the result of capi_scale_cluster.
type scaleClusterResult struct {
	Namespace         string `json:"namespace"`
	Name              string `json:"name"`
	Target            string `json:"target"`
	Replicas          int    `json:"replicas"`
	MachineDeployment string `json:"machineDeployment,omitempty"`
	Message           string `json:"message"`
}

// CreateScaleClusterHandler creates a handler for scaling clusters
func CreateScaleClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
		target, ok := arguments["target"].(string)
		if !ok || target == "" {
			return nil, fmt.Errorf("target argument is required")
		}
		replicas, ok := arguments["replicas"].(float64)
		if !ok {
			return nil, fmt.Errorf("replicas argument is required and must be a number")
		}
		machineDeployment, _ := arguments["machineDeployment"].(string)

		err = capiClient.ScaleCluster(ctx, namespace, name, target, int(replicas), machineDeployment)
		if err != nil {
			return nil, fmt.Errorf("failed to scale cluster: %w", err)
		}

		return jsonResult(scaleClusterResult{
			Namespace:         namespace,
			Name:              name,
			Target:            target,
			Replicas:          int(replicas),
			MachineDeployment: machineDeployment,
			Message:           fmt.Sprintf("Cluster %s/%s: %s scaled to %d replicas", namespace, name, target, int(replicas)),
		})
	}
}

// kubeconfigResult is the result of capi_get_kubeconfig.
type kubeconfigResult struct {
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Kubeconfig string `json:"kubeconfig"`
}

// CreateGetKubeconfigHandler creates a handler for retrieving cluster kubeconfig
func CreateGetKubeconfigHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		kubeconfig, err := capiClient.GetKubeconfig(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
		}

		return jsonResult(kubeconfigResult{
			Namespace:  namespace,
			Name:       name,
			Kubeconfig: kubeconfig,
		})
	}
}

// pauseResult is the result of capi_pause_cluster and capi_resume_cluster.
type pauseResult struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Paused    bool   `json:"paused"`
	Message   string `json:"message"`
}

// CreatePauseClusterHandler creates a handler for pausing cluster reconciliation
func CreatePauseClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		err = capiClient.PauseCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to pause cluster: %w", err)
		}

		return jsonResult(pauseResult{
			Namespace: namespace,
			Name:      name,
			Paused:    true,
			Message: "Reconciliation paused: CAPI controllers make no changes to the cluster (no updates, no scaling) " +
				"until capi_resume_cluster is called, so manual operations are safe.",
		})
	}
}

// CreateResumeClusterHandler creates a handler for resuming cluster reconciliation
func CreateResumeClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		err = capiClient.ResumeCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to resume cluster: %w", err)
		}

		return jsonResult(pauseResult{
			Namespace: namespace,
			Name:      name,
			Paused:    false,
			Message: "Reconciliation resumed: CAPI controllers manage the cluster again and apply any pending " +
				"updates or scaling.",
		})
	}
}

// deleteClusterResult is the result of capi_delete_cluster.
type deleteClusterResult struct {
	Namespace       string         `json:"namespace"`
	Name            string         `json:"name"`
	Deleted         bool           `json:"deleted"`
	Cluster         ClusterSummary `json:"cluster"`
	Message         string         `json:"message"`
	Recommendations []string       `json:"recommendations,omitempty"`
}

// CreateDeleteClusterHandler creates a handler for deleting a cluster
func CreateDeleteClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Get cluster status first to show what will be deleted
		status, err := capiClient.GetClusterStatus(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster status: %w", err)
		}

		result := deleteClusterResult{
			Namespace: namespace,
			Name:      name,
			Cluster:   clusterSummary(status),
		}

		// Safety check: a Ready cluster is only deleted with force=true.
		if !force && status.Ready {
			result.Deleted = false
			result.Message = "Safety check failed: the cluster is Ready and appears operational. Pass force=true to delete it anyway."
			result.Recommendations = []string{
				"Back up important data",
				"Migrate workloads to another cluster",
				"Confirm this is the intended cluster",
			}
			return jsonResult(result)
		}

		err = capiClient.DeleteCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to delete cluster: %w", err)
		}

		result.Deleted = true
		result.Message = "Deletion initiated. Cleaning up cluster resources, deprovisioning infrastructure and " +
			"processing finalizers can take several minutes; monitor with capi_list_clusters."
		return jsonResult(result)
	}
}

// upgradeClusterResult is the result of capi_upgrade_cluster.
type upgradeClusterResult struct {
	Namespace       string `json:"namespace"`
	Name            string `json:"name"`
	PreviousVersion string `json:"previousVersion,omitempty"`
	TargetVersion   string `json:"targetVersion"`
	UpgradeWorkers  bool   `json:"upgradeWorkers"`
	Message         string `json:"message"`
}

// CreateUpgradeClusterHandler creates a handler for upgrading cluster Kubernetes version
func CreateUpgradeClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
		targetVersion, ok := arguments["target_version"].(string)
		if !ok || targetVersion == "" {
			return nil, fmt.Errorf("target_version argument is required")
		}

		// Default to upgrading workers
		upgradeWorkers := true
		if uw, ok := arguments["upgrade_workers"].(bool); ok {
			upgradeWorkers = uw
		}

		// Get current cluster status
		status, err := capiClient.GetClusterStatus(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster status: %w", err)
		}

		opts := capi.UpgradeClusterOptions{
			Namespace:      namespace,
			Name:           name,
			TargetVersion:  targetVersion,
			UpgradeWorkers: upgradeWorkers,
		}

		if err := capiClient.UpgradeCluster(ctx, opts); err != nil {
			return nil, fmt.Errorf("failed to upgrade cluster: %w", err)
		}

		workers := "worker machines are upgraded after the control plane is ready"
		if !upgradeWorkers {
			workers = "worker machines are not upgraded (upgrade_workers=false)"
		}

		return jsonResult(upgradeClusterResult{
			Namespace:       namespace,
			Name:            name,
			PreviousVersion: status.Version,
			TargetVersion:   targetVersion,
			UpgradeWorkers:  upgradeWorkers,
			Message: fmt.Sprintf("Upgrade initiated. Control plane machines are upgraded first, one at a time; %s. "+
				"The rolling upgrade can take 30-60 minutes and workloads may be rescheduled; monitor with "+
				"capi_cluster_status, capi_cluster_health and capi_list_machines.", workers),
		})
	}
}

// updateClusterResult is the result of capi_update_cluster; labels and
// annotations are the cluster's metadata after the update.
type updateClusterResult struct {
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Message     string            `json:"message"`
}

// CreateUpdateClusterHandler creates a handler for updating cluster metadata
func CreateUpdateClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		// Get labels and annotations from arguments
		labels, _ := arguments["labels"].(map[string]interface{})
		annotations, _ := arguments["annotations"].(map[string]interface{})

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

		opts := capi.UpdateClusterOptions{
			Namespace:   namespace,
			Name:        name,
			Labels:      labelMap,
			Annotations: annotationMap,
		}

		cluster, err := capiClient.UpdateCluster(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to update cluster: %w", err)
		}

		return jsonResult(updateClusterResult{
			Namespace:   namespace,
			Name:        name,
			Labels:      cluster.Labels,
			Annotations: cluster.Annotations,
			Message:     "Cluster metadata updated",
		})
	}
}

// moveClusterResult is the result of capi_move_cluster.
type moveClusterResult struct {
	Namespace        string   `json:"namespace"`
	Name             string   `json:"name"`
	DryRun           bool     `json:"dryRun"`
	TargetKubeconfig string   `json:"targetKubeconfig,omitempty"`
	TargetNamespace  string   `json:"targetNamespace,omitempty"`
	Steps            []string `json:"steps"`
	Commands         []string `json:"commands"`
	Manifest         string   `json:"manifest"`
	Notes            []string `json:"notes"`
}

// CreateMoveClusterHandler creates a handler for moving clusters between management clusters
func CreateMoveClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		targetKubeconfig, _ := arguments["target_kubeconfig"].(string)
		targetNamespace, _ := arguments["target_namespace"].(string)
		dryRun, _ := arguments["dry_run"].(bool)

		opts := capi.MoveClusterOptions{
			Namespace:        namespace,
			Name:             name,
			TargetKubeconfig: targetKubeconfig,
			TargetNamespace:  targetNamespace,
			DryRun:           dryRun,
		}

		// Get move instructions/manifest
		manifest, err := capiClient.MoveCluster(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare cluster move: %w", err)
		}

		kubeconfigFlag := "<target-kubeconfig>"
		if targetKubeconfig != "" {
			kubeconfigFlag = targetKubeconfig
		}
		moveCommand := fmt.Sprintf("clusterctl move --to-kubeconfig=%s --namespace %s", kubeconfigFlag, namespace)
		if targetNamespace != "" && targetNamespace != namespace {
			moveCommand += " --to-namespace " + targetNamespace
		}

		notes := []string{
			"The source cluster is paused during the move",
			"All cluster resources are migrated",
			"Ensure network connectivity between the management clusters",
			"Provider versions must match on both management clusters",
		}
		if dryRun {
			notes = append([]string{"Dry run: no changes are made"}, notes...)
		}

		return jsonResult(moveClusterResult{
			Namespace:        namespace,
			Name:             name,
			DryRun:           dryRun,
			TargetKubeconfig: targetKubeconfig,
			TargetNamespace:  targetNamespace,
			Steps: []string{
				"Ensure the target management cluster is ready",
				"Install the required providers on the target management cluster",
				"Create the target namespace if needed",
				"Pause the cluster, then run clusterctl move (see commands)",
			},
			Commands: []string{
				fmt.Sprintf("kubectl patch cluster %s -n %s --type merge -p '{\"spec\":{\"paused\":true}}'", name, namespace),
				moveCommand,
			},
			Manifest: manifest,
			Notes:    notes,
		})
	}
}

// backupClusterResult is the result of capi_backup_cluster.
type backupClusterResult struct {
	Namespace         string   `json:"namespace"`
	Name              string   `json:"name"`
	Format            string   `json:"format"`
	IncludeSecrets    bool     `json:"includeSecrets"`
	Backup            string   `json:"backup"`
	SuggestedFilename string   `json:"suggestedFilename"`
	Notes             []string `json:"notes"`
}

// CreateBackupClusterHandler creates a handler for backing up cluster configurations
func CreateBackupClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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

		includeSecrets, _ := arguments["include_secrets"].(bool)
		outputFormat, _ := arguments["output_format"].(string)
		if outputFormat == "" {
			outputFormat = "yaml"
		}

		opts := capi.BackupClusterOptions{
			Namespace:      namespace,
			Name:           name,
			IncludeSecrets: includeSecrets,
			OutputFormat:   outputFormat,
		}

		backup, err := capiClient.BackupCluster(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster backup: %w", err)
		}

		notes := []string{
			"The backup covers CAPI resources only; workload data is not included",
			"Infrastructure provider resources may need a separate backup",
			"Store the backup in a secure location and test the restore procedure outside production",
		}
		if includeSecrets {
			notes = append(notes, "Secrets are included: handle and encrypt the backup with care")
		}

		return jsonResult(backupClusterResult{
			Namespace:         namespace,
			Name:              name,
			Format:            outputFormat,
			IncludeSecrets:    includeSecrets,
			Backup:            backup,
			SuggestedFilename: fmt.Sprintf("cluster-%s-%s-backup.%s", namespace, name, outputFormat),
			Notes:             notes,
		})
	}
}
