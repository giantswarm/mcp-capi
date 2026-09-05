package handlers

// readOnlyTools are the tools that never change anything: they list, get,
// inspect or export. Only these are registered when the server runs
// read-only. Every tool must appear in exactly one of readOnlyTools and
// mutatingTools; the classification is deliberate and a test enforces it,
// so a new tool has to be placed here before it ships.
var readOnlyTools = map[string]struct{}{
	"test": {},

	// Clusters
	"capi_list_clusters":    {},
	"capi_get_cluster":      {},
	"capi_cluster_status":   {},
	"capi_cluster_health":   {},
	"capi_get_kubeconfig":   {},
	"capi_backup_cluster":   {},
	"capi_list_machines":    {},
	"capi_get_machine":      {},
	"capi_list_machinesets": {},
	"capi_get_machineset":   {},
	"capi_node_status":      {},

	// MachineDeployments
	"capi_list_machinedeployments": {},

	// Providers
	"capi_list_infrastructure_providers": {},
	"capi_get_provider_config":           {},
	"capi_aws_list_clusters":             {},
	"capi_aws_get_cluster":               {},
	"capi_aws_get_machine_template":      {},
	"capi_azure_list_clusters":           {},
	"capi_azure_get_cluster":             {},
	"capi_gcp_list_clusters":             {},
	"capi_gcp_get_cluster":               {},
	"capi_vsphere_list_clusters":         {},
	"capi_vsphere_get_cluster":           {},
}

// mutatingTools create, update, delete or otherwise change resources. They
// are hidden when the server runs read-only, and their writes go through the
// client's WritePolicy (GitOps guard) otherwise.
var mutatingTools = map[string]struct{}{
	// Clusters
	"capi_create_cluster":  {},
	"capi_scale_cluster":   {},
	"capi_pause_cluster":   {},
	"capi_resume_cluster":  {},
	"capi_delete_cluster":  {},
	"capi_upgrade_cluster": {},
	"capi_update_cluster":  {},
	"capi_move_cluster":    {},

	// Machines
	"capi_delete_machine":    {},
	"capi_remediate_machine": {},

	// MachineDeployments
	"capi_create_machinedeployment":  {},
	"capi_scale_machinedeployment":   {},
	"capi_update_machinedeployment":  {},
	"capi_rollout_machinedeployment": {},

	// Nodes
	"capi_drain_node":  {},
	"capi_cordon_node": {},
}

// IsReadOnlyTool reports whether the named tool is offered by a read-only
// server. Unknown names are treated as mutating.
func IsReadOnlyTool(name string) bool {
	_, ok := readOnlyTools[name]
	return ok
}

// IsMutatingTool reports whether the named tool is classified as mutating.
func IsMutatingTool(name string) bool {
	_, ok := mutatingTools[name]
	return ok
}
