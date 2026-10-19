package handlers

import "github.com/mark3labs/mcp-go/mcp"

// moveTool is listed in three of the classifications below.
const moveTool = "capi_move_cluster"

// readOnlyTools are the tools that never change anything and hand out no
// credentials: they list, get, inspect or export resources. These are always
// registered, read-only server or not. Every tool must appear in exactly one
// of readOnlyTools, mutatingTools and credentialTools; the classification is
// deliberate and a test enforces it, so a new tool has to be placed here
// before it ships.
//
// capi_backup_cluster is read-only: it exports nothing yet. Its
// include_secrets argument is checked as a credential export, so the client
// refuses it unless the policy's ExposeKubeconfig is set.
var readOnlyTools = map[string]struct{}{
	// Clusters
	"capi_list_clusters":    {},
	"capi_get_cluster":      {},
	"capi_cluster_status":   {},
	"capi_cluster_health":   {},
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
	moveTool:               {},

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

// credentialTools hand out a workload cluster's credentials: the admin
// kubeconfig from its Secret. Reading a Secret is not a write, but the
// kubeconfig is the power to do anything to the workload cluster, so these
// tools are registered only when the policy's ExposeKubeconfig is set,
// independent of ReadOnly. The client refuses the export as well.
var credentialTools = map[string]struct{}{
	"capi_get_kubeconfig": {},
}

// nonDestructiveWrites are the mutating tools that never change or remove
// what exists, so their destructiveHint is false: the creates, and
// capi_move_cluster, which writes nothing yet (giantswarm/mcp-capi#308). Every
// other mutating tool scales, updates, pauses, drains or deletes something
// that is already there and is annotated destructive.
var nonDestructiveWrites = map[string]struct{}{
	"capi_create_cluster":           {},
	"capi_create_machinedeployment": {},
	moveTool:                        {},
}

// idempotentWrites are the mutating tools whose repeated call with the same
// arguments leaves the cluster as the first call did (set a size, pause,
// resume, cordon, apply the same update, plan a move). Creates, deletes,
// drains, remediations, upgrades and rollouts are not.
var idempotentWrites = map[string]struct{}{
	"capi_scale_cluster":            {},
	"capi_pause_cluster":            {},
	"capi_resume_cluster":           {},
	"capi_update_cluster":           {},
	"capi_scale_machinedeployment":  {},
	"capi_update_machinedeployment": {},
	"capi_cordon_node":              {},
	moveTool:                        {},
}

// annotationsFor derives a tool's MCP annotations from its classification, so
// a client (or an aggregator's read-only toolset) can tell the reads from the
// writes: read-only and credential tools change nothing (readOnlyHint true,
// destructiveHint false, idempotentHint true); mutating tools are writes whose
// destructiveHint is false only for nonDestructiveWrites. Every tool talks to
// the management cluster only (openWorldHint false). Without this, mcp-go's
// NewTool defaults (readOnlyHint false, destructiveHint true) would present
// capi_list_clusters as a destructive write.
func annotationsFor(name string) mcp.ToolAnnotation {
	f := false
	t := true
	if IsReadOnlyTool(name) || IsCredentialTool(name) {
		return mcp.ToolAnnotation{ReadOnlyHint: &t, DestructiveHint: &f, IdempotentHint: &t, OpenWorldHint: &f}
	}
	_, nonDestructive := nonDestructiveWrites[name]
	_, idempotent := idempotentWrites[name]
	destructive := !nonDestructive
	return mcp.ToolAnnotation{ReadOnlyHint: &f, DestructiveHint: &destructive, IdempotentHint: &idempotent, OpenWorldHint: &f}
}

// IsReadOnlyTool reports whether the named tool reads without handing out
// credentials; such tools are offered by every server. Unknown names are
// treated as mutating.
func IsReadOnlyTool(name string) bool {
	_, ok := readOnlyTools[name]
	return ok
}

// IsCredentialTool reports whether the named tool exports a workload
// cluster's credentials and is therefore offered only with ExposeKubeconfig.
func IsCredentialTool(name string) bool {
	_, ok := credentialTools[name]
	return ok
}

// IsMutatingTool reports whether the named tool is classified as mutating.
func IsMutatingTool(name string) bool {
	_, ok := mutatingTools[name]
	return ok
}
