package handlers

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// BuildAllTools constructs and returns the complete MCP tool registry.
// It builds all available tools including cluster management operations,
// machine operations, and provider-specific tools. The returned slice
// contains tool definitions paired with their handler functions, ready
// to be registered with an MCP server.
func BuildAllTools(serverCtx *ServerContext) ([]ToolRegistration, error) {
	// Test tool
	testTool := mcp.NewTool(
		"test",
		mcp.WithDescription("A simple test tool"),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("Message to echo back"),
		),
	)
	testRegistration := ToolRegistration{
		Tool:    testTool,
		Handler: TestToolHandler,
	}

	clusterTools := buildClusterTools(serverCtx)
	machineTools := buildMachineTools(serverCtx)
	providerTools := buildProviderTools(serverCtx)

	tools := make([]ToolRegistration, 0, 1+len(clusterTools)+len(machineTools)+len(providerTools))
	tools = append(tools, testRegistration)
	tools = append(tools, clusterTools...)
	tools = append(tools, machineTools...)
	tools = append(tools, providerTools...)

	return tools, nil
}

// buildClusterTools constructs and returns all cluster management tools.
// This includes tools for cluster lifecycle operations (create, delete, get, list),
// cluster configuration management (scale, upgrade, pause/resume, update metadata),
// and cluster operations (backup, move, kubeconfig retrieval, health checks).
func buildClusterTools(serverCtx *ServerContext) []ToolRegistration {
	return []ToolRegistration{
		// capi_create_cluster
		{
			Tool: mcp.NewTool(
				"capi_create_cluster",
				mcp.WithDescription("Create a new CAPI cluster (basic implementation)"),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace for the cluster")),
				mcp.WithString(
					"provider",
					mcp.Required(),
					mcp.Description("Infrastructure provider (aws, azure, gcp, vsphere)"),
				),
				mcp.WithString("kubernetes_version", mcp.Description("Kubernetes version (default: v1.29.0)")),
				mcp.WithNumber("control_plane_count", mcp.Description("Number of control plane nodes (default: 3)")),
				mcp.WithNumber("worker_count", mcp.Description("Number of worker nodes (default: 3)")),
				mcp.WithString("region", mcp.Description("Cloud provider region")),
				mcp.WithString("instance_type", mcp.Description("Instance type for nodes")),
			),
			Handler: CreateCreateClusterHandler(serverCtx),
		},

		// capi_list_clusters
		{
			Tool: mcp.NewTool(
				"capi_list_clusters",
				mcp.WithDescription(
					"List all CAPI clusters. Supports filtering by namespace, label key=value pairs, or searching by a term that matches any label value.",
				),
				mcp.WithString("namespace", mcp.Description("Namespace to filter clusters (optional, empty for all)")),
				mcp.WithObject(
					"label_selector",
					mcp.Description(
						"Label key-value pairs to filter clusters (e.g. {\"env\": \"production\", \"team\": \"platform\"})",
					),
				),
				mcp.WithString(
					"search",
					mcp.Description("Search term to match against cluster names and label values (case-insensitive)"),
				),
			),
			Handler: CreateListClustersHandler(serverCtx),
		},

		// capi_get_cluster
		{
			Tool: mcp.NewTool(
				"capi_get_cluster",
				mcp.WithDescription(
					"Get detailed information about a specific cluster. The name is matched against the Kubernetes resource name first, and if not found, against label values (case-insensitive).",
				),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString(
					"name",
					mcp.Required(),
					mcp.Description("Name of the cluster (matched against resource name and label values)"),
				),
			),
			Handler: CreateGetClusterHandler(serverCtx),
		},

		// capi_cluster_status
		{
			Tool: mcp.NewTool(
				"capi_cluster_status",
				mcp.WithDescription("Get detailed cluster status"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			),
			Handler: CreateClusterStatusHandler(serverCtx),
		},

		// capi_cluster_health
		{
			Tool: mcp.NewTool(
				"capi_cluster_health",
				mcp.WithDescription("Check cluster health status"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			),
			Handler: CreateClusterHealthHandler(serverCtx),
		},

		// capi_scale_cluster
		{
			Tool: mcp.NewTool(
				"capi_scale_cluster",
				mcp.WithDescription("Scale cluster control plane or workers"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
				mcp.WithString("target", mcp.Required(), mcp.Description("Scale target: controlplane or workers")),
				mcp.WithNumber("replicas", mcp.Required(), mcp.Description("Number of replicas")),
				mcp.WithString("machineDeployment", mcp.Description("Machine deployment name (required for workers)")),
			),
			Handler: CreateScaleClusterHandler(serverCtx),
		},

		// capi_get_kubeconfig
		{
			Tool: mcp.NewTool(
				"capi_get_kubeconfig",
				mcp.WithDescription("Retrieve cluster kubeconfig"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			),
			Handler: CreateGetKubeconfigHandler(serverCtx),
		},

		// capi_pause_cluster
		{
			Tool: mcp.NewTool(
				"capi_pause_cluster",
				mcp.WithDescription("Pause cluster reconciliation"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			),
			Handler: CreatePauseClusterHandler(serverCtx),
		},

		// capi_resume_cluster
		{
			Tool: mcp.NewTool(
				"capi_resume_cluster",
				mcp.WithDescription("Resume cluster reconciliation"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			),
			Handler: CreateResumeClusterHandler(serverCtx),
		},

		// capi_delete_cluster
		{
			Tool: mcp.NewTool(
				"capi_delete_cluster",
				mcp.WithDescription("Delete a cluster"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
				mcp.WithBoolean("force", mcp.Description("Force deletion even if cluster is healthy")),
			),
			Handler: CreateDeleteClusterHandler(serverCtx),
		},

		// capi_upgrade_cluster
		{
			Tool: mcp.NewTool(
				"capi_upgrade_cluster",
				mcp.WithDescription("Upgrade cluster Kubernetes version"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
				mcp.WithString("target_version", mcp.Required(), mcp.Description("Target Kubernetes version")),
				mcp.WithBoolean("upgrade_workers", mcp.Description("Upgrade worker nodes (default: true)")),
			),
			Handler: CreateUpgradeClusterHandler(serverCtx),
		},

		// capi_update_cluster
		{
			Tool: mcp.NewTool(
				"capi_update_cluster",
				mcp.WithDescription("Update cluster metadata (labels, annotations)"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
				mcp.WithObject("labels", mcp.Description("Labels to set/update (use empty string to remove)")),
				mcp.WithObject(
					"annotations",
					mcp.Description("Annotations to set/update (use empty string to remove)"),
				),
			),
			Handler: CreateUpdateClusterHandler(serverCtx),
		},

		// capi_move_cluster
		{
			Tool: mcp.NewTool(
				"capi_move_cluster",
				mcp.WithDescription("Move cluster between management clusters"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
				mcp.WithString("target_kubeconfig", mcp.Description("Target management cluster kubeconfig path")),
				mcp.WithString("target_namespace", mcp.Description("Target namespace (defaults to source namespace)")),
				mcp.WithBoolean("dry_run", mcp.Description("Perform dry run")),
			),
			Handler: CreateMoveClusterHandler(serverCtx),
		},

		// capi_backup_cluster
		{
			Tool: mcp.NewTool(
				"capi_backup_cluster",
				mcp.WithDescription("Backup cluster configuration"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
				mcp.WithBoolean("include_secrets", mcp.Description("Include secrets in backup")),
				mcp.WithString("output_format", mcp.Description("Output format (yaml or json, default: yaml)")),
			),
			Handler: CreateBackupClusterHandler(serverCtx),
		},
	}
}

// buildMachineTools constructs and returns all machine and node operation tools.
// This includes tools for individual machines (list, get, delete, remediate),
// MachineDeployment management (create, list, scale, update, rollout),
// MachineSet operations (list, get), and node operations (drain, cordon/uncordon, status).
func buildMachineTools(serverCtx *ServerContext) []ToolRegistration {
	return []ToolRegistration{
		// capi_list_machines
		{
			Tool: mcp.NewTool(
				"capi_list_machines",
				mcp.WithDescription("List CAPI machines"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace to search")),
				mcp.WithString("clusterName", mcp.Description("Filter by cluster name")),
			),
			Handler: CreateListMachinesHandler(serverCtx),
		},

		// capi_get_machine
		{
			Tool: mcp.NewTool(
				"capi_get_machine",
				mcp.WithDescription("Get detailed machine information"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the machine")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the machine")),
			),
			Handler: CreateGetMachineHandler(serverCtx),
		},

		// capi_delete_machine
		{
			Tool: mcp.NewTool(
				"capi_delete_machine",
				mcp.WithDescription("Delete a machine"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the machine")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the machine")),
				mcp.WithBoolean("force", mcp.Description("Force deletion")),
			),
			Handler: CreateDeleteMachineHandler(serverCtx),
		},

		// capi_remediate_machine
		{
			Tool: mcp.NewTool(
				"capi_remediate_machine",
				mcp.WithDescription("Trigger machine remediation"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the machine")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the machine")),
			),
			Handler: CreateRemediateMachineHandler(serverCtx),
		},

		// MachineDeployment tools
		{
			Tool: mcp.NewTool(
				"capi_list_machinedeployments",
				mcp.WithDescription("List machine deployments"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace to search")),
				mcp.WithString("clusterName", mcp.Description("Filter by cluster name")),
			),
			Handler: CreateListMachineDeploymentsHandler(serverCtx),
		},

		{
			Tool: mcp.NewTool(
				"capi_create_machinedeployment",
				mcp.WithDescription("Create a new machine deployment"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name")),
				mcp.WithString("cluster_name", mcp.Required(), mcp.Description("Cluster name")),
				mcp.WithNumber("replicas", mcp.Description("Number of replicas (default: 1)")),
				mcp.WithString("version", mcp.Description("Kubernetes version (default: v1.29.0)")),
				mcp.WithString("infra_kind", mcp.Required(), mcp.Description("Infrastructure reference kind")),
				mcp.WithString("infra_name", mcp.Required(), mcp.Description("Infrastructure reference name")),
				mcp.WithString("infra_api_version", mcp.Description("Infrastructure API version")),
				mcp.WithString("bootstrap_kind", mcp.Required(), mcp.Description("Bootstrap config kind")),
				mcp.WithString("bootstrap_name", mcp.Required(), mcp.Description("Bootstrap config name")),
				mcp.WithString("bootstrap_api_version", mcp.Description("Bootstrap API version")),
			),
			Handler: CreateCreateMachineDeploymentHandler(serverCtx),
		},

		{
			Tool: mcp.NewTool(
				"capi_scale_machinedeployment",
				mcp.WithDescription("Scale a machine deployment"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name")),
				mcp.WithNumber("replicas", mcp.Required(), mcp.Description("Number of replicas")),
			),
			Handler: CreateScaleMachineDeploymentHandler(serverCtx),
		},

		{
			Tool: mcp.NewTool(
				"capi_update_machinedeployment",
				mcp.WithDescription("Update machine deployment configuration"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name")),
				mcp.WithString("version", mcp.Description("Kubernetes version")),
				mcp.WithNumber("replicas", mcp.Description("Number of replicas")),
				mcp.WithNumber("min_ready_seconds", mcp.Description("Minimum ready seconds")),
				mcp.WithObject("labels", mcp.Description("Labels to update")),
				mcp.WithObject("annotations", mcp.Description("Annotations to update")),
			),
			Handler: CreateUpdateMachineDeploymentHandler(serverCtx),
		},

		{
			Tool: mcp.NewTool(
				"capi_rollout_machinedeployment",
				mcp.WithDescription("Trigger machine deployment rollout"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name")),
				mcp.WithString("reason", mcp.Description("Reason for rollout")),
			),
			Handler: CreateRolloutMachineDeploymentHandler(serverCtx),
		},

		// MachineSet tools
		{
			Tool: mcp.NewTool(
				"capi_list_machinesets",
				mcp.WithDescription("List machine sets"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
				mcp.WithString("clusterName", mcp.Description("Filter by cluster name")),
			),
			Handler: CreateListMachineSetsHandler(serverCtx),
		},

		{
			Tool: mcp.NewTool(
				"capi_get_machineset",
				mcp.WithDescription("Get machine set details"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name")),
			),
			Handler: CreateGetMachineSetHandler(serverCtx),
		},

		// Node operation tools
		{
			Tool: mcp.NewTool(
				"capi_drain_node",
				mcp.WithDescription("Drain a node"),
				mcp.WithString("namespace", mcp.Description("Namespace of the machine")),
				mcp.WithString("machine_name", mcp.Description("Name of the machine")),
				mcp.WithString("node_name", mcp.Description("Name of the node (alternative to machine)")),
				mcp.WithBoolean("ignore_daemonsets", mcp.Description("Ignore DaemonSet pods")),
				mcp.WithBoolean("delete_local_data", mcp.Description("Delete pods with local storage")),
				mcp.WithBoolean("force", mcp.Description("Force drain")),
				mcp.WithNumber("grace_period_seconds", mcp.Description("Grace period in seconds")),
			),
			Handler: CreateDrainNodeHandler(serverCtx),
		},

		{
			Tool: mcp.NewTool(
				"capi_cordon_node",
				mcp.WithDescription("Cordon/uncordon a node"),
				mcp.WithString("namespace", mcp.Description("Namespace of the machine")),
				mcp.WithString("machine_name", mcp.Description("Name of the machine")),
				mcp.WithString("node_name", mcp.Description("Name of the node (alternative to machine)")),
				mcp.WithBoolean("uncordon", mcp.Description("Uncordon instead of cordon")),
			),
			Handler: CreateCordonNodeHandler(serverCtx),
		},

		{
			Tool: mcp.NewTool(
				"capi_node_status",
				mcp.WithDescription("Get node status"),
				mcp.WithString("namespace", mcp.Description("Namespace of the machine")),
				mcp.WithString("machine_name", mcp.Description("Name of the machine")),
				mcp.WithString("node_name", mcp.Description("Name of the node (alternative to machine)")),
			),
			Handler: CreateNodeStatusHandler(serverCtx),
		},
	}
}

// buildProviderTools constructs and returns all read-only provider-specific tools.
// This includes generic provider discovery, provider configuration lookup,
// and per-provider cluster listing and detail retrieval for AWS, Azure, GCP,
// and vSphere.
func buildProviderTools(serverCtx *ServerContext) []ToolRegistration {
	return []ToolRegistration{
		// Generic provider tools

		// capi_list_infrastructure_providers
		{
			Tool: mcp.NewTool(
				"capi_list_infrastructure_providers",
				mcp.WithDescription("List available infrastructure providers"),
			),
			Handler: CreateListInfrastructureProvidersHandler(serverCtx),
		},

		// capi_get_provider_config
		{
			Tool: mcp.NewTool(
				"capi_get_provider_config",
				mcp.WithDescription("Get provider configuration details"),
				mcp.WithString(
					"provider",
					mcp.Required(),
					mcp.Description("Infrastructure provider (aws, azure, gcp, vsphere)"),
				),
			),
			Handler: CreateGetProviderConfigHandler(serverCtx),
		},

		// AWS provider tools

		// capi_aws_list_clusters
		{
			Tool: mcp.NewTool(
				"capi_aws_list_clusters",
				mcp.WithDescription("List AWS clusters"),
				mcp.WithString("namespace", mcp.Description("Namespace to filter clusters (optional, empty for all)")),
			),
			Handler: CreateAWSListClustersHandler(serverCtx),
		},

		// capi_aws_get_cluster
		{
			Tool: mcp.NewTool(
				"capi_aws_get_cluster",
				mcp.WithDescription("Get detailed information about an AWS cluster"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			),
			Handler: CreateAWSGetClusterHandler(serverCtx),
		},

		// capi_aws_get_machine_template
		{
			Tool: mcp.NewTool(
				"capi_aws_get_machine_template",
				mcp.WithDescription("Get AWS machine template details"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the machine template")),
				mcp.WithString("name", mcp.Description("Name of the machine template (optional, lists all if empty)")),
			),
			Handler: CreateAWSGetMachineTemplateHandler(serverCtx),
		},

		// Azure provider tools

		// capi_azure_list_clusters
		{
			Tool: mcp.NewTool(
				"capi_azure_list_clusters",
				mcp.WithDescription("List Azure clusters"),
				mcp.WithString("namespace", mcp.Description("Namespace to filter clusters (optional, empty for all)")),
			),
			Handler: CreateAzureListClustersHandler(serverCtx),
		},

		// capi_azure_get_cluster
		{
			Tool: mcp.NewTool(
				"capi_azure_get_cluster",
				mcp.WithDescription("Get detailed information about an Azure cluster"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			),
			Handler: CreateAzureGetClusterHandler(serverCtx),
		},

		// GCP provider tools

		// capi_gcp_list_clusters
		{
			Tool: mcp.NewTool(
				"capi_gcp_list_clusters",
				mcp.WithDescription("List GCP clusters"),
				mcp.WithString("namespace", mcp.Description("Namespace to filter clusters (optional, empty for all)")),
			),
			Handler: CreateGCPListClustersHandler(serverCtx),
		},

		// capi_gcp_get_cluster
		{
			Tool: mcp.NewTool(
				"capi_gcp_get_cluster",
				mcp.WithDescription("Get detailed information about a GCP cluster"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			),
			Handler: CreateGCPGetClusterHandler(serverCtx),
		},

		// vSphere provider tools

		// capi_vsphere_list_clusters
		{
			Tool: mcp.NewTool(
				"capi_vsphere_list_clusters",
				mcp.WithDescription("List vSphere clusters"),
				mcp.WithString("namespace", mcp.Description("Namespace to filter clusters (optional, empty for all)")),
			),
			Handler: CreateVSphereListClustersHandler(serverCtx),
		},

		// capi_vsphere_get_cluster
		{
			Tool: mcp.NewTool(
				"capi_vsphere_get_cluster",
				mcp.WithDescription("Get detailed information about a vSphere cluster"),
				mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
				mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			),
			Handler: CreateVSphereGetClusterHandler(serverCtx),
		},
	}
}
