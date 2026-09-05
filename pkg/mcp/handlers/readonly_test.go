package handlers

import (
	"sort"
	"testing"

	"github.com/giantswarm/mcp-capi/pkg/capi"
)

const (
	kubeconfigTool = "capi_get_kubeconfig"
	backupTool     = "capi_backup_cluster"
)

// TestEveryToolIsClassified fails when a tool is registered without being
// placed in exactly one of readOnlyTools, mutatingTools and credentialTools,
// so a new tool cannot slip past the policy filter unclassified.
func TestEveryToolIsClassified(t *testing.T) {
	tools, err := buildTools(NewServerContext(nil))
	if err != nil {
		t.Fatalf("buildTools() error = %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("buildTools() returned no tools")
	}
	seen := map[string]bool{}
	for _, reg := range tools {
		name := reg.Tool.Name
		if seen[name] {
			t.Errorf("tool %q registered twice", name)
		}
		seen[name] = true
		classes := 0
		for _, in := range []bool{IsReadOnlyTool(name), IsMutatingTool(name), IsCredentialTool(name)} {
			if in {
				classes++
			}
		}
		switch classes {
		case 0:
			t.Errorf("tool %q is not classified: add it to readOnlyTools, mutatingTools or credentialTools in readonly.go", name)
		case 1:
		default:
			t.Errorf("tool %q is listed in %d classes, want exactly one", name, classes)
		}
	}
	for listName, list := range map[string]map[string]struct{}{"readOnlyTools": readOnlyTools, "mutatingTools": mutatingTools, "credentialTools": credentialTools} {
		for name := range list {
			if !seen[name] {
				t.Errorf("%s lists %q, which is not registered", listName, name)
			}
		}
	}
	if !IsCredentialTool(kubeconfigTool) {
		t.Errorf("%s must be a credential tool: it hands out the workload cluster's admin kubeconfig", kubeconfigTool)
	}
	if !IsReadOnlyTool(backupTool) {
		t.Errorf("%s is read-only (its default export carries no Secret data); the client refuses include_secrets", backupTool)
	}
}

func offeredNames(t *testing.T, policy capi.WritePolicy) []string {
	t.Helper()
	ctx := NewCallerIdentityServerContext(capi.NewBearerClientFactory("https://kubernetes.default.svc", "", nil), policy)
	tools, err := BuildAllTools(ctx)
	if err != nil {
		t.Fatalf("BuildAllTools(%+v) error = %v", policy, err)
	}
	names := make([]string, 0, len(tools))
	for _, reg := range tools {
		names = append(names, reg.Tool.Name)
	}
	sort.Strings(names)
	return names
}

func offers(names []string, name string) bool {
	idx := sort.SearchStrings(names, name)
	return idx < len(names) && names[idx] == name
}

func TestBuildAllToolsReadOnly(t *testing.T) {
	all := offeredNames(t, capi.WritePolicy{})
	filtered := offeredNames(t, capi.WritePolicy{ReadOnly: true, GitOpsGuard: true})

	if len(filtered) != len(readOnlyTools) {
		t.Fatalf("read-only server offers %d tools, want %d (the read-only list; no mutating and no credential tool)", len(filtered), len(readOnlyTools))
	}
	for _, name := range filtered {
		if IsMutatingTool(name) {
			t.Errorf("read-only server offers mutating tool %q", name)
		}
		if IsCredentialTool(name) {
			t.Errorf("read-only server offers credential tool %q without ExposeKubeconfig", name)
		}
	}
	for _, name := range []string{"capi_list_clusters", "capi_get_cluster", "capi_cluster_health", backupTool, "capi_list_machines"} {
		if !offers(filtered, name) {
			t.Errorf("read-only server does not offer %q", name)
		}
	}
	for _, name := range []string{"capi_create_cluster", "capi_delete_cluster", "capi_scale_cluster", "capi_pause_cluster", "capi_upgrade_cluster", "capi_delete_machine", "capi_drain_node", kubeconfigTool} {
		if offers(filtered, name) {
			t.Errorf("read-only server offers %q", name)
		}
	}
	if len(all) <= len(filtered) {
		t.Fatalf("permissive server offers %d tools, read-only %d; the filter removed nothing", len(all), len(filtered))
	}
}

// TestBuildAllToolsExposeKubeconfig pins the credential tools to their own
// switch: the zero policy (writes allowed) still hides capi_get_kubeconfig,
// and ExposeKubeconfig offers it even on a read-only server.
func TestBuildAllToolsExposeKubeconfig(t *testing.T) {
	permissive := offeredNames(t, capi.WritePolicy{})
	if offers(permissive, kubeconfigTool) {
		t.Errorf("a server without ExposeKubeconfig offers %q although writes are allowed; the switches are independent", kubeconfigTool)
	}
	if len(permissive) != len(readOnlyTools)+len(mutatingTools) {
		t.Errorf("permissive server offers %d tools, want %d (read-only + mutating, no credential tool)", len(permissive), len(readOnlyTools)+len(mutatingTools))
	}

	exposed := offeredNames(t, capi.WritePolicy{ReadOnly: true, GitOpsGuard: true, ExposeKubeconfig: true})
	if !offers(exposed, kubeconfigTool) {
		t.Errorf("a read-only server with ExposeKubeconfig does not offer %q", kubeconfigTool)
	}
	if len(exposed) != len(readOnlyTools)+len(credentialTools) {
		t.Errorf("read-only server with ExposeKubeconfig offers %d tools, want %d (read-only + credential, no mutating tool)", len(exposed), len(readOnlyTools)+len(credentialTools))
	}
	for _, name := range exposed {
		if IsMutatingTool(name) {
			t.Errorf("read-only server with ExposeKubeconfig offers mutating tool %q", name)
		}
	}

	everything := offeredNames(t, capi.WritePolicy{ExposeKubeconfig: true})
	if len(everything) != len(readOnlyTools)+len(mutatingTools)+len(credentialTools) {
		t.Errorf("fully open server offers %d tools, want %d", len(everything), len(readOnlyTools)+len(mutatingTools)+len(credentialTools))
	}
}
