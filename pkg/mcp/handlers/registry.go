package handlers

import (
	"github.com/giantswarm/mcp-toolkit/tooldesc"
	"github.com/mark3labs/mcp-go/mcp"
)

// BuildAllTools constructs and returns the complete MCP tool registry.
// It builds all available tools including cluster management operations,
// machine operations, and provider-specific tools. The returned slice
// contains tool definitions paired with their handler functions, ready
// to be registered with an MCP server.
//
// The server context's write policy decides which tools are offered, rather
// than letting them fail on every call (the clients refuse as well):
//
//   - mutatingTools only when the policy is not read-only;
//   - credentialTools (capi_get_kubeconfig) only when ExposeKubeconfig is
//     set, whatever ReadOnly says;
//   - readOnlyTools always.
//
// Every offered tool carries the MCP annotations of its class
// (annotationsFor), so clients can tell the reads from the writes.
func BuildAllTools(serverCtx *ServerContext) ([]ToolRegistration, error) {
	tools, err := buildTools(serverCtx)
	if err != nil {
		return nil, err
	}
	policy := serverCtx.WritePolicy()
	filtered := make([]ToolRegistration, 0, len(tools))
	for _, reg := range tools {
		name := reg.Tool.Name
		switch {
		case IsCredentialTool(name):
			if !policy.ExposeKubeconfig {
				continue
			}
		case IsMutatingTool(name):
			if policy.ReadOnly {
				continue
			}
		}
		reg.Tool.Annotations = annotationsFor(name)
		filtered = append(filtered, reg)
	}
	return filtered, nil
}

// buildTools constructs every tool regardless of the write policy.
func buildTools(serverCtx *ServerContext) ([]ToolRegistration, error) {
	var tools []ToolRegistration

	// Cluster tools
	tools = append(tools, buildClusterTools(serverCtx)...)

	// Machine tools
	tools = append(tools, buildMachineTools(serverCtx)...)

	// Provider tools
	tools = append(tools, buildProviderTools(serverCtx)...)

	return tools, nil
}

// buildClusterTools constructs and returns all cluster management tools.
// This includes tools for cluster lifecycle operations (create, delete, get, list),
// cluster configuration management (scale, upgrade, pause/resume, update metadata),
// and cluster operations (backup, move, kubeconfig retrieval, health checks).
func buildClusterTools(serverCtx *ServerContext) []ToolRegistration {
	var tools []ToolRegistration

	// capi_create_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_create_cluster",
			tooldesc.Describe(
				"Create the CAPI Cluster object of a new workload cluster on AWS, Azure, GCP or vSphere.",
				"Creates only the Cluster object: its infrastructure cluster, control plane and worker MachineDeployments are created separately.",
			),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace for the cluster")),
			mcp.WithString("provider", mcp.Required(), mcp.Description("Infrastructure provider (aws, azure, gcp, vsphere)")),
			mcp.WithString("kubernetes_version", mcp.Description("Kubernetes version (default: v1.29.0)")),
			mcp.WithNumber("control_plane_count", mcp.Description("Number of control plane nodes (default: 3)")),
			mcp.WithNumber("worker_count", mcp.Description("Number of worker nodes (default: 3)")),
			mcp.WithString("region", mcp.Description("Cloud provider region")),
			mcp.WithString("instance_type", mcp.Description("Instance type for nodes")),
		),
		Handler: CreateCreateClusterHandler(serverCtx),
	})

	// capi_list_clusters
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_list_clusters",
			tooldesc.Describe(
				"List CAPI workload clusters, filtered by namespace, labels or a search term over names and label values.",
				"One provider's clusters: capi_aws_list_clusters and its siblings.",
			),
			mcp.WithString("namespace", mcp.Description("Namespace to filter clusters (optional, empty for all)")),
			mcp.WithObject("label_selector", mcp.Description("Label key-value pairs to filter clusters (e.g. {\"env\": \"production\", \"team\": \"platform\"})")),
			mcp.WithString("search", mcp.Description("Search term to match against cluster names and label values (case-insensitive)")),
		),
		Handler: CreateListClustersHandler(serverCtx),
	})

	// capi_get_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_get_cluster",
			tooldesc.Describe(
				"Get one CAPI cluster's spec and status, by resource name or a label value.",
				"The name matches the resource name first, then label values (case-insensitive). Health: capi_cluster_health.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster (matched against resource name and label values)")),
		),
		Handler: CreateGetClusterHandler(serverCtx),
	})

	// capi_cluster_status
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_cluster_status",
			tooldesc.Describe(
				"Get a CAPI cluster's phase, readiness and conditions.",
				"Issues and recommendations: capi_cluster_health.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
		),
		Handler: CreateClusterStatusHandler(serverCtx),
	})

	// capi_cluster_health
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_cluster_health",
			tooldesc.Describe(
				"Check a CAPI cluster's health: control plane, infrastructure and workers, with issues and recommendations.",
				"Phase and conditions only: capi_cluster_status.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
		),
		Handler: CreateClusterHealthHandler(serverCtx),
	})

	// capi_scale_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_scale_cluster",
			tooldesc.Describe(
				"Scale a CAPI cluster's control plane, or one of its worker MachineDeployments, to a replica count.",
				"Workers need machineDeployment. Scaling down removes machines.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			mcp.WithString("target", mcp.Required(), mcp.Description("Scale target: controlplane or workers")),
			mcp.WithNumber("replicas", mcp.Required(), mcp.Description("Number of replicas")),
			mcp.WithString("machineDeployment", mcp.Description("Machine deployment name (required for workers)")),
		),
		Handler: CreateScaleClusterHandler(serverCtx),
	})

	// capi_get_kubeconfig
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_get_kubeconfig",
			tooldesc.Describe(
				"Get a workload cluster's admin kubeconfig from its Secret.",
				"Hands out full admin access to the workload cluster: only when the person asked for it.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
		),
		Handler: CreateGetKubeconfigHandler(serverCtx),
	})

	// capi_pause_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_pause_cluster",
			tooldesc.Describe(
				"Pause the reconciliation of a CAPI cluster and its machines.",
				"The CAPI controllers leave it alone until capi_resume_cluster.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
		),
		Handler: CreatePauseClusterHandler(serverCtx),
	})

	// capi_resume_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_resume_cluster",
			tooldesc.Describe(
				"Resume the reconciliation of a paused CAPI cluster.",
				"Pausing: capi_pause_cluster.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
		),
		Handler: CreateResumeClusterHandler(serverCtx),
	})

	// capi_delete_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_delete_cluster",
			tooldesc.Describe(
				"Delete a CAPI workload cluster and deprovision its infrastructure.",
				"Irreversible: confirm with the person first. A Ready cluster is refused unless force is true.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			mcp.WithBoolean("force", mcp.Description("Force deletion even if cluster is healthy")),
		),
		Handler: CreateDeleteClusterHandler(serverCtx),
	})

	// capi_upgrade_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_upgrade_cluster",
			tooldesc.Describe(
				"Upgrade a CAPI cluster's Kubernetes version: control plane, and workers unless upgrade_workers is false.",
				"Replaces every upgraded machine. Labels and annotations: capi_update_cluster.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			mcp.WithString("target_version", mcp.Required(), mcp.Description("Target Kubernetes version")),
			mcp.WithBoolean("upgrade_workers", mcp.Description("Upgrade worker nodes (default: true)")),
		),
		Handler: CreateUpgradeClusterHandler(serverCtx),
	})

	// capi_update_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_update_cluster",
			tooldesc.Describe(
				"Set or remove labels and annotations on a CAPI cluster.",
				"An empty value removes the key. Version: capi_upgrade_cluster.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			mcp.WithObject("labels", mcp.Description("Labels to set/update (use empty string to remove)")),
			mcp.WithObject("annotations", mcp.Description("Annotations to set/update (use empty string to remove)")),
		),
		Handler: CreateUpdateClusterHandler(serverCtx),
	})

	// capi_move_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_move_cluster",
			tooldesc.Describe(
				"Plan a clusterctl move of a CAPI cluster to another management cluster.",
				"Moves nothing itself: returns the steps, commands and manifest to run.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			mcp.WithString("target_kubeconfig", mcp.Description("Target management cluster kubeconfig path")),
			mcp.WithString("target_namespace", mcp.Description("Target namespace (defaults to source namespace)")),
			mcp.WithBoolean("dry_run", mcp.Description("Perform dry run")),
		),
		Handler: CreateMoveClusterHandler(serverCtx),
	})

	// capi_backup_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_backup_cluster",
			tooldesc.Describe(
				"Export a CAPI cluster's resources as YAML or JSON for a backup.",
				"Carries no Secret data unless include_secrets is set.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
			mcp.WithBoolean("include_secrets", mcp.Description("Include the cluster's Secrets (kubeconfig, certificates). Refused unless the server allows credential export.")),
			mcp.WithString("output_format", mcp.Description("Output format (yaml or json, default: yaml)")),
		),
		Handler: CreateBackupClusterHandler(serverCtx),
	})

	return tools
}

// buildMachineTools constructs and returns all machine and node operation tools.
// This includes tools for individual machines (list, get, delete, remediate),
// MachineDeployment management (create, list, scale, update, rollout),
// MachineSet operations (list, get), and node operations (drain, cordon/uncordon, status).
func buildMachineTools(serverCtx *ServerContext) []ToolRegistration {
	var tools []ToolRegistration

	// capi_list_machines
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_list_machines",
			tooldesc.Describe(
				"List CAPI Machines in a namespace, optionally of one cluster.",
				"Machine groups: capi_list_machinedeployments and capi_list_machinesets.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace to search")),
			mcp.WithString("clusterName", mcp.Description("Filter by cluster name")),
		),
		Handler: CreateListMachinesHandler(serverCtx),
	})

	// capi_get_machine
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_get_machine",
			tooldesc.Describe(
				"Get one CAPI Machine's spec and status.",
				"Its node: capi_node_status.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the machine")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the machine")),
		),
		Handler: CreateGetMachineHandler(serverCtx),
	})

	// capi_delete_machine
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_delete_machine",
			tooldesc.Describe(
				"Delete a CAPI Machine: its node is drained and its infrastructure removed.",
				"Irreversible for the machine. For an unhealthy machine prefer capi_remediate_machine.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the machine")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the machine")),
			mcp.WithBoolean("force", mcp.Description("Force deletion")),
		),
		Handler: CreateDeleteMachineHandler(serverCtx),
	})

	// capi_remediate_machine
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_remediate_machine",
			tooldesc.Describe(
				"Mark a CAPI Machine for remediation by its MachineHealthCheck: replacement, reboot or custom.",
				"Deleting the machine outright: capi_delete_machine.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the machine")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the machine")),
		),
		Handler: CreateRemediateMachineHandler(serverCtx),
	})

	// MachineDeployment tools
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_list_machinedeployments",
			tooldesc.Describe(
				"List CAPI MachineDeployments (worker node pools) in a namespace, optionally of one cluster.",
				"Their machines: capi_list_machines.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace to search")),
			mcp.WithString("clusterName", mcp.Description("Filter by cluster name")),
		),
		Handler: CreateListMachineDeploymentsHandler(serverCtx),
	})

	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_create_machinedeployment",
			tooldesc.Describe(
				"Create a CAPI MachineDeployment (worker node pool) from existing infrastructure and bootstrap templates.",
				"The referenced templates must exist; this creates neither.",
			),
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
	})

	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_scale_machinedeployment",
			tooldesc.Describe(
				"Scale a CAPI MachineDeployment (worker node pool) to a replica count.",
				"Scaling down removes machines.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name")),
			mcp.WithNumber("replicas", mcp.Required(), mcp.Description("Number of replicas")),
		),
		Handler: CreateScaleMachineDeploymentHandler(serverCtx),
	})

	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_update_machinedeployment",
			tooldesc.Describe(
				"Change a CAPI MachineDeployment's version, replicas, min ready seconds, labels or annotations.",
				"A version change replaces its machines. Replicas only: capi_scale_machinedeployment.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name")),
			mcp.WithString("version", mcp.Description("Kubernetes version")),
			mcp.WithNumber("replicas", mcp.Description("Number of replicas")),
			mcp.WithNumber("min_ready_seconds", mcp.Description("Minimum ready seconds")),
			mcp.WithObject("labels", mcp.Description("Labels to update")),
			mcp.WithObject("annotations", mcp.Description("Annotations to update")),
		),
		Handler: CreateUpdateMachineDeploymentHandler(serverCtx),
	})

	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_rollout_machinedeployment",
			tooldesc.Describe(
				"Roll out a CAPI MachineDeployment again, replacing all of its machines.",
				"Replaces machines as its rollout strategy says; the reason is recorded as an annotation.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name")),
			mcp.WithString("reason", mcp.Description("Reason for rollout")),
		),
		Handler: CreateRolloutMachineDeploymentHandler(serverCtx),
	})

	// MachineSet tools
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_list_machinesets",
			tooldesc.Describe(
				"List CAPI MachineSets in a namespace, optionally of one cluster.",
				"Usually owned by a MachineDeployment: capi_list_machinedeployments.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
			mcp.WithString("clusterName", mcp.Description("Filter by cluster name")),
		),
		Handler: CreateListMachineSetsHandler(serverCtx),
	})

	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_get_machineset",
			tooldesc.Describe(
				"Get one CAPI MachineSet's spec and status.",
				"All machine sets: capi_list_machinesets.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name")),
		),
		Handler: CreateGetMachineSetHandler(serverCtx),
	})

	// Node operation tools
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_drain_node",
			tooldesc.Describe(
				"Drain a node, by its name or its CAPI Machine's: cordon it and evict its pods, respecting PodDisruptionBudgets.",
				"Refuses before evicting anything when a DaemonSet pod, an unmanaged pod or emptyDir data would need ignore_daemonsets, force or delete_local_data. Waits until the evicted pods are gone; the node stays cordoned. Cordon alone: capi_cordon_node.",
			),
			mcp.WithString("namespace", mcp.Description("Namespace of the machine")),
			mcp.WithString("machine_name", mcp.Description("Name of the machine")),
			mcp.WithString("node_name", mcp.Description("Name of the node (alternative to machine)")),
			mcp.WithBoolean("ignore_daemonsets", mcp.Description("Leave DaemonSet pods on the node instead of refusing the drain")),
			mcp.WithBoolean("delete_local_data", mcp.Description("Evict pods with emptyDir volumes, losing that data")),
			mcp.WithBoolean("force", mcp.Description("Evict pods no controller manages; they are not recreated")),
			mcp.WithNumber("grace_period_seconds", mcp.Description("Termination grace period of the evicted pods (default: each pod's own)")),
			mcp.WithNumber("timeout_seconds", mcp.Description("Give up after this many seconds, waiting for PodDisruptionBudgets and terminating pods (default: 120)")),
		),
		Handler: CreateDrainNodeHandler(serverCtx),
	})

	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_cordon_node",
			tooldesc.Describe(
				"Cordon or uncordon a node, by its name or its CAPI Machine's.",
				"Draining: capi_drain_node.",
			),
			mcp.WithString("namespace", mcp.Description("Namespace of the machine")),
			mcp.WithString("machine_name", mcp.Description("Name of the machine")),
			mcp.WithString("node_name", mcp.Description("Name of the node (alternative to machine)")),
			mcp.WithBoolean("uncordon", mcp.Description("Uncordon instead of cordon")),
		),
		Handler: CreateCordonNodeHandler(serverCtx),
	})

	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_node_status",
			tooldesc.Describe(
				"Get a node's status and conditions, by its name or its CAPI Machine's.",
				"The Machine itself: capi_get_machine.",
			),
			mcp.WithString("namespace", mcp.Description("Namespace of the machine")),
			mcp.WithString("machine_name", mcp.Description("Name of the machine")),
			mcp.WithString("node_name", mcp.Description("Name of the node (alternative to machine)")),
		),
		Handler: CreateNodeStatusHandler(serverCtx),
	})

	return tools
}

// buildProviderTools constructs and returns all read-only provider-specific tools.
// This includes generic provider discovery, provider configuration lookup,
// and per-provider cluster listing and detail retrieval for AWS, Azure, GCP,
// and vSphere.
func buildProviderTools(serverCtx *ServerContext) []ToolRegistration {
	var tools []ToolRegistration

	// Generic provider tools

	// capi_list_infrastructure_providers
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_list_infrastructure_providers",
			tooldesc.Describe(
				"List the CAPI infrastructure providers this server knows: AWS, Azure, GCP, vSphere.",
				"A static list; it does not inspect the cluster.",
			),
		),
		Handler: CreateListInfrastructureProvidersHandler(serverCtx),
	})

	// capi_get_provider_config
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_get_provider_config",
			tooldesc.Describe(
				"Get an infrastructure provider's required credentials, settings and CRD kinds.",
				"The provider list: capi_list_infrastructure_providers.",
			),
			mcp.WithString("provider", mcp.Required(), mcp.Description("Infrastructure provider (aws, azure, gcp, vsphere)")),
		),
		Handler: CreateGetProviderConfigHandler(serverCtx),
	})

	// AWS provider tools

	// capi_aws_list_clusters
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_aws_list_clusters",
			tooldesc.Describe(
				"List the CAPI clusters on AWS (AWSCluster, AWSManagedCluster).",
				"All providers: capi_list_clusters.",
			),
			mcp.WithString("namespace", mcp.Description("Namespace to filter clusters (optional, empty for all)")),
		),
		Handler: CreateAWSListClustersHandler(serverCtx),
	})

	// capi_aws_get_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_aws_get_cluster",
			tooldesc.Describe(
				"Get an AWS CAPI cluster's readiness, network and conditions.",
				"Any provider: capi_get_cluster.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
		),
		Handler: CreateAWSGetClusterHandler(serverCtx),
	})

	// capi_aws_get_machine_template
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_aws_get_machine_template",
			tooldesc.Describe(
				"Get an AWSMachineTemplate, or list them all in a namespace.",
				"Instance type, AMI and the like of a node pool.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the machine template")),
			mcp.WithString("name", mcp.Description("Name of the machine template (optional, lists all if empty)")),
		),
		Handler: CreateAWSGetMachineTemplateHandler(serverCtx),
	})

	// Azure provider tools

	// capi_azure_list_clusters
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_azure_list_clusters",
			tooldesc.Describe(
				"List the CAPI clusters on Azure (AzureCluster, AzureManagedCluster).",
				"All providers: capi_list_clusters.",
			),
			mcp.WithString("namespace", mcp.Description("Namespace to filter clusters (optional, empty for all)")),
		),
		Handler: CreateAzureListClustersHandler(serverCtx),
	})

	// capi_azure_get_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_azure_get_cluster",
			tooldesc.Describe(
				"Get an Azure CAPI cluster's readiness, network and conditions.",
				"Any provider: capi_get_cluster.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
		),
		Handler: CreateAzureGetClusterHandler(serverCtx),
	})

	// GCP provider tools

	// capi_gcp_list_clusters
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_gcp_list_clusters",
			tooldesc.Describe(
				"List the CAPI clusters on GCP (GCPCluster, GCPManagedCluster).",
				"All providers: capi_list_clusters.",
			),
			mcp.WithString("namespace", mcp.Description("Namespace to filter clusters (optional, empty for all)")),
		),
		Handler: CreateGCPListClustersHandler(serverCtx),
	})

	// capi_gcp_get_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_gcp_get_cluster",
			tooldesc.Describe(
				"Get a GCP CAPI cluster's readiness, network and conditions.",
				"Any provider: capi_get_cluster.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
		),
		Handler: CreateGCPGetClusterHandler(serverCtx),
	})

	// vSphere provider tools

	// capi_vsphere_list_clusters
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_vsphere_list_clusters",
			tooldesc.Describe(
				"List the CAPI clusters on vSphere (VSphereCluster).",
				"All providers: capi_list_clusters.",
			),
			mcp.WithString("namespace", mcp.Description("Namespace to filter clusters (optional, empty for all)")),
		),
		Handler: CreateVSphereListClustersHandler(serverCtx),
	})

	// capi_vsphere_get_cluster
	tools = append(tools, ToolRegistration{
		Tool: mcp.NewTool(
			"capi_vsphere_get_cluster",
			tooldesc.Describe(
				"Get a vSphere CAPI cluster's readiness, network and conditions.",
				"Any provider: capi_get_cluster.",
			),
			mcp.WithString("namespace", mcp.Required(), mcp.Description("Namespace of the cluster")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the cluster")),
		),
		Handler: CreateVSphereGetClusterHandler(serverCtx),
	})

	return tools
}
