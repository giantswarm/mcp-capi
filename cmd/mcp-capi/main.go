package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "mcp-capi"
	serverVersion = "0.1.0"
)

// ServerContext holds shared resources for the server
type ServerContext struct {
	capiClient *capi.Client
}

func main() {
	// Create context that cancels on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received, closing server...")
		cancel()
	}()

	// Initialize CAPI client
	log.Println("Initializing CAPI client...")
	capiClient, err := capi.NewClient("")
	if err != nil {
		log.Fatalf("Failed to create CAPI client: %v", err)
	}

	// Initialize providers
	if err := capiClient.InitializeProviders(); err != nil {
		log.Printf("Warning: Failed to initialize providers: %v", err)
	}

	// Create server context
	serverCtx := &ServerContext{
		capiClient: capiClient,
	}

	// Create MCP server
	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true), // subscribe, list
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	// Add a simple test tool
	testTool := mcp.NewTool(
		"test",
		mcp.WithDescription("A simple test tool"),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("Message to echo back"),
		),
	)

	mcpServer.AddTool(testTool, testToolHandler)

	// Add CAPI create cluster tool
	createClusterTool := mcp.NewTool(
		"capi_create_cluster",
		mcp.WithDescription("Create a new CAPI cluster (basic implementation)"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace for the cluster"),
		),
		mcp.WithString("provider",
			mcp.Required(),
			mcp.Description("Infrastructure provider (aws, azure, gcp, vsphere)"),
		),
		mcp.WithString("kubernetes_version",
			mcp.Description("Kubernetes version (default: v1.29.0)"),
		),
		mcp.WithNumber("control_plane_count",
			mcp.Description("Number of control plane nodes (default: 3)"),
		),
		mcp.WithNumber("worker_count",
			mcp.Description("Number of worker nodes (default: 3)"),
		),
		mcp.WithString("region",
			mcp.Description("Cloud provider region"),
		),
		mcp.WithString("instance_type",
			mcp.Description("Instance type for nodes"),
		),
	)

	mcpServer.AddTool(createClusterTool, createCreateClusterHandler(serverCtx))

	// Add CAPI list clusters tool
	listClustersTool := mcp.NewTool(
		"capi_list_clusters",
		mcp.WithDescription("List all CAPI clusters"),
		mcp.WithString("namespace",
			mcp.Description("Namespace to filter clusters (optional, empty for all)"),
		),
	)

	mcpServer.AddTool(listClustersTool, createListClustersHandler(serverCtx))

	// Add CAPI get cluster tool
	getClusterTool := mcp.NewTool(
		"capi_get_cluster",
		mcp.WithDescription("Get details of a specific CAPI cluster"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
	)

	mcpServer.AddTool(getClusterTool, createGetClusterHandler(serverCtx))

	// Add CAPI cluster status tool
	clusterStatusTool := mcp.NewTool(
		"capi_cluster_status",
		mcp.WithDescription("Get detailed status of a CAPI cluster including conditions and provider status"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
	)

	mcpServer.AddTool(clusterStatusTool, createClusterStatusHandler(serverCtx))

	// Add CAPI cluster health tool
	clusterHealthTool := mcp.NewTool(
		"capi_cluster_health",
		mcp.WithDescription("Check cluster health and identify issues"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
	)

	mcpServer.AddTool(clusterHealthTool, createClusterHealthHandler(serverCtx))

	// Add CAPI upgrade cluster tool
	upgradeClusterTool := mcp.NewTool(
		"capi_upgrade_cluster",
		mcp.WithDescription("Upgrade a CAPI cluster to a new Kubernetes version"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
		mcp.WithString("target_version",
			mcp.Required(),
			mcp.Description("Target Kubernetes version (e.g., v1.29.0)"),
		),
		mcp.WithBoolean("upgrade_workers",
			mcp.Description("Also upgrade worker nodes (default: true)"),
		),
	)

	mcpServer.AddTool(upgradeClusterTool, createUpgradeClusterHandler(serverCtx))

	// Add CAPI update cluster tool
	updateClusterTool := mcp.NewTool(
		"capi_update_cluster",
		mcp.WithDescription("Update cluster metadata (labels and annotations)"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
		mcp.WithObject("labels",
			mcp.Description("Labels to add/update/remove (use empty string to remove)"),
		),
		mcp.WithObject("annotations",
			mcp.Description("Annotations to add/update/remove (use empty string to remove)"),
		),
	)

	mcpServer.AddTool(updateClusterTool, createUpdateClusterHandler(serverCtx))

	// Add CAPI move cluster tool
	moveClusterTool := mcp.NewTool(
		"capi_move_cluster",
		mcp.WithDescription("Prepare a cluster for migration to another management cluster"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
		mcp.WithString("target_kubeconfig",
			mcp.Description("Path to target management cluster kubeconfig"),
		),
		mcp.WithString("target_namespace",
			mcp.Description("Target namespace (defaults to source namespace)"),
		),
		mcp.WithBoolean("dry_run",
			mcp.Description("Show what would be moved without doing it"),
		),
	)

	mcpServer.AddTool(moveClusterTool, createMoveClusterHandler(serverCtx))

	// Add CAPI backup cluster tool
	backupClusterTool := mcp.NewTool(
		"capi_backup_cluster",
		mcp.WithDescription("Create a backup of cluster configuration and resources"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
		mcp.WithBoolean("include_secrets",
			mcp.Description("Include secrets in backup (kubeconfig, certificates)"),
		),
		mcp.WithString("output_format",
			mcp.Description("Output format: yaml or json (default: yaml)"),
		),
	)

	mcpServer.AddTool(backupClusterTool, createBackupClusterHandler(serverCtx))

	// Add CAPI scale cluster tool
	scaleClusterTool := mcp.NewTool(
		"capi_scale_cluster",
		mcp.WithDescription("Scale control plane or worker nodes of a CAPI cluster"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
		mcp.WithString("target",
			mcp.Required(),
			mcp.Description("What to scale: 'controlplane' or 'workers'"),
		),
		mcp.WithNumber("replicas",
			mcp.Required(),
			mcp.Description("Number of replicas to scale to"),
		),
		mcp.WithString("machineDeployment",
			mcp.Description("Name of the machine deployment (required when target is 'workers')"),
		),
	)

	mcpServer.AddTool(scaleClusterTool, createScaleClusterHandler(serverCtx))

	// Add CAPI list machines tool
	listMachinesTool := mcp.NewTool(
		"capi_list_machines",
		mcp.WithDescription("List CAPI machines with optional filtering by cluster"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace to list machines from"),
		),
		mcp.WithString("clusterName",
			mcp.Description("Filter machines by cluster name (optional)"),
		),
	)

	mcpServer.AddTool(listMachinesTool, createListMachinesHandler(serverCtx))

	// Add CAPI list machine deployments tool
	listMachineDeploymentsTool := mcp.NewTool(
		"capi_list_machinedeployments",
		mcp.WithDescription("List CAPI machine deployments (worker node pools)"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace to list machine deployments from"),
		),
		mcp.WithString("clusterName",
			mcp.Description("Filter machine deployments by cluster name (optional)"),
		),
	)

	mcpServer.AddTool(listMachineDeploymentsTool, createListMachineDeploymentsHandler(serverCtx))

	// Add CAPI create machine deployment tool
	createMachineDeploymentTool := mcp.NewTool(
		"capi_create_machinedeployment",
		mcp.WithDescription("Create a new worker node pool (MachineDeployment)"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace for the machine deployment"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the machine deployment"),
		),
		mcp.WithString("cluster_name",
			mcp.Required(),
			mcp.Description("Name of the cluster this deployment belongs to"),
		),
		mcp.WithNumber("replicas",
			mcp.Description("Number of replicas (default: 1)"),
		),
		mcp.WithString("version",
			mcp.Description("Kubernetes version (e.g., v1.29.0)"),
		),
		mcp.WithString("infra_kind",
			mcp.Required(),
			mcp.Description("Kind of infrastructure template (e.g., AWSMachineTemplate)"),
		),
		mcp.WithString("infra_name",
			mcp.Required(),
			mcp.Description("Name of infrastructure template"),
		),
		mcp.WithString("infra_api_version",
			mcp.Description("API version of infrastructure template"),
		),
		mcp.WithString("bootstrap_kind",
			mcp.Required(),
			mcp.Description("Kind of bootstrap config (e.g., KubeadmConfigTemplate)"),
		),
		mcp.WithString("bootstrap_name",
			mcp.Required(),
			mcp.Description("Name of bootstrap config template"),
		),
		mcp.WithString("bootstrap_api_version",
			mcp.Description("API version of bootstrap config"),
		),
	)

	mcpServer.AddTool(createMachineDeploymentTool, createCreateMachineDeploymentHandler(serverCtx))

	// Add CAPI scale machine deployment tool
	scaleMachineDeploymentTool := mcp.NewTool(
		"capi_scale_machinedeployment",
		mcp.WithDescription("Scale worker nodes up or down"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the machine deployment"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the machine deployment"),
		),
		mcp.WithNumber("replicas",
			mcp.Required(),
			mcp.Description("Number of replicas to scale to"),
		),
	)

	mcpServer.AddTool(scaleMachineDeploymentTool, createScaleMachineDeploymentHandler(serverCtx))

	// Add CAPI get kubeconfig tool
	getKubeconfigTool := mcp.NewTool(
		"capi_get_kubeconfig",
		mcp.WithDescription("Retrieve kubeconfig for a workload cluster"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
	)

	mcpServer.AddTool(getKubeconfigTool, createGetKubeconfigHandler(serverCtx))

	// Add CAPI pause cluster tool
	pauseClusterTool := mcp.NewTool(
		"capi_pause_cluster",
		mcp.WithDescription("Pause cluster reconciliation (stops all CAPI controllers from reconciling the cluster)"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
	)

	mcpServer.AddTool(pauseClusterTool, createPauseClusterHandler(serverCtx))

	// Add CAPI resume cluster tool
	resumeClusterTool := mcp.NewTool(
		"capi_resume_cluster",
		mcp.WithDescription("Resume cluster reconciliation (allows CAPI controllers to reconcile the cluster again)"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
	)

	mcpServer.AddTool(resumeClusterTool, createResumeClusterHandler(serverCtx))

	// Add CAPI get machine tool
	getMachineTool := mcp.NewTool(
		"capi_get_machine",
		mcp.WithDescription("Get detailed information about a specific CAPI machine"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the machine"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the machine"),
		),
	)

	mcpServer.AddTool(getMachineTool, createGetMachineHandler(serverCtx))

	// Add CAPI delete machine tool
	deleteMachineTool := mcp.NewTool(
		"capi_delete_machine",
		mcp.WithDescription("Delete a specific CAPI machine"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the machine"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the machine to delete"),
		),
		mcp.WithBoolean("force",
			mcp.Description("Force deletion even if machine is healthy or control plane"),
		),
	)

	mcpServer.AddTool(deleteMachineTool, createDeleteMachineHandler(serverCtx))

	// Add CAPI remediate machine tool
	remediateMachineTool := mcp.NewTool(
		"capi_remediate_machine",
		mcp.WithDescription("Trigger machine health check remediation"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the machine"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the machine to remediate"),
		),
	)

	mcpServer.AddTool(remediateMachineTool, createRemediateMachineHandler(serverCtx))

	// Add CAPI delete cluster tool
	deleteClusterTool := mcp.NewTool(
		"capi_delete_cluster",
		mcp.WithDescription("Delete a CAPI cluster safely (with confirmation)"),
		mcp.WithString("namespace",
			mcp.Required(),
			mcp.Description("Namespace of the cluster"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the cluster"),
		),
		mcp.WithBoolean("force",
			mcp.Description("Skip safety checks and force deletion (use with caution)"),
		),
	)

	mcpServer.AddTool(deleteClusterTool, createDeleteClusterHandler(serverCtx))

	// Add a simple test resource
	testResource := mcp.NewResource(
		"capi://test",
		"Test Resource",
		mcp.WithMIMEType("text/plain"),
	)

	mcpServer.AddResource(testResource, testResourceHandler)

	// Start server based on transport type
	transport := os.Getenv("MCP_TRANSPORT")
	if transport == "" {
		transport = "stdio"
	}

	// Set up signal handling for graceful shutdown
	go func() {
		<-ctx.Done()
		log.Println("Context cancelled, shutting down...")
		os.Exit(0)
	}()

	switch transport {
	case "stdio":
		log.Println("Starting MCP CAPI server with stdio transport...")
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	default:
		log.Fatalf("Unsupported transport: %s", transport)
	}
}
