package handlers

import (
	"sort"
	"testing"

	"github.com/giantswarm/mcp-capi/pkg/capi"
)

// TestEveryToolIsClassified fails when a tool is registered without being
// placed in exactly one of readOnlyTools and mutatingTools, so a new tool
// cannot slip past the read-only filter unclassified.
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
		ro, mut := IsReadOnlyTool(name), IsMutatingTool(name)
		switch {
		case ro && mut:
			t.Errorf("tool %q is listed as both read-only and mutating", name)
		case !ro && !mut:
			t.Errorf("tool %q is not classified: add it to readOnlyTools or mutatingTools in readonly.go", name)
		}
	}
	for name := range readOnlyTools {
		if !seen[name] {
			t.Errorf("readOnlyTools lists %q, which is not registered", name)
		}
	}
	for name := range mutatingTools {
		if !seen[name] {
			t.Errorf("mutatingTools lists %q, which is not registered", name)
		}
	}
}

func TestBuildAllToolsReadOnly(t *testing.T) {
	all, err := BuildAllTools(NewServerContext(nil))
	if err != nil {
		t.Fatalf("BuildAllTools() error = %v", err)
	}
	readOnlyCtx := NewCallerIdentityServerContext(capi.NewBearerClientFactory("https://kubernetes.default.svc", "", nil), capi.WritePolicy{ReadOnly: true, GitOpsGuard: true})
	filtered, err := BuildAllTools(readOnlyCtx)
	if err != nil {
		t.Fatalf("BuildAllTools(read-only) error = %v", err)
	}

	if len(filtered) != len(readOnlyTools) {
		t.Fatalf("read-only server offers %d tools, want %d", len(filtered), len(readOnlyTools))
	}
	var offered []string
	for _, reg := range filtered {
		if IsMutatingTool(reg.Tool.Name) {
			t.Errorf("read-only server offers mutating tool %q", reg.Tool.Name)
		}
		offered = append(offered, reg.Tool.Name)
	}
	sort.Strings(offered)
	for _, name := range []string{"capi_list_clusters", "capi_get_cluster", "capi_cluster_health", "capi_get_kubeconfig", "capi_list_machines"} {
		if idx := sort.SearchStrings(offered, name); idx >= len(offered) || offered[idx] != name {
			t.Errorf("read-only server does not offer %q", name)
		}
	}
	for _, name := range []string{"capi_create_cluster", "capi_delete_cluster", "capi_scale_cluster", "capi_pause_cluster", "capi_upgrade_cluster", "capi_delete_machine", "capi_drain_node"} {
		if idx := sort.SearchStrings(offered, name); idx < len(offered) && offered[idx] == name {
			t.Errorf("read-only server offers %q", name)
		}
	}
	if len(all) <= len(filtered) {
		t.Fatalf("permissive server offers %d tools, read-only %d; the filter removed nothing", len(all), len(filtered))
	}
}
