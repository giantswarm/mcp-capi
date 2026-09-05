package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

// TestCapiPauseCluster exercises the write policy on a representative
// mutating tool: the GitOps guard refuses objects owned by Flux or a Helm
// release, lets unmanaged objects through, and a read-only server does not
// offer the tool at all.
func TestCapiPauseCluster(t *testing.T) {
	t.Parallel()

	t.Run("pauses an unmanaged cluster", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "hand-made").
			ToolCall("capi_pause_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "hand-made").
			AssertContent("paused.golden").
			Execute()
	})

	t.Run("refuses a cluster rendered by a Helm release", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		// The shape of a workload cluster on a Giant Swarm management
		// cluster: rendered by the cluster-aws chart from an App in Git.
		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "gazelle").
			WithLabels(map[string]string{
				managedByLabel:  managedByHelm,
				"helm.sh/chart": "cluster-8.0.0",
			}).
			WithAnnotations(map[string]string{
				"meta.helm.sh/release-name":      "gazelle",
				"meta.helm.sh/release-namespace": namespace,
			}).
			Create().
			ToolCall("capi_pause_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "gazelle").
			AssertError("managed_by_helm.golden").
			Execute()
	})

	t.Run("refuses a cluster applied by a Flux Kustomization", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "flux-cluster").
			WithLabels(map[string]string{
				"kustomize.toolkit.fluxcd.io/name":      "clusters",
				"kustomize.toolkit.fluxcd.io/namespace": "flux-giantswarm",
			}).
			Create().
			ToolCall("capi_pause_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "flux-cluster").
			AssertError("managed_by_flux.golden").
			Execute()
	})

	t.Run("pauses a Helm-rendered cluster when the GitOps guard is off", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			WithoutGitOpsGuard().
			CreateNamespace(namespace).
			Cluster(namespace, "gazelle").
			WithLabels(map[string]string{managedByLabel: managedByHelm}).
			Create().
			ToolCall("capi_pause_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "gazelle").
			AssertContent("paused_guard_off.golden").
			Execute()
	})

	t.Run("is not offered by a read-only server", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			ReadOnly().
			CreateNamespace(namespace).
			CreateClusters(namespace, "hand-made").
			ToolCall("capi_pause_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "hand-made").
			AssertError("read_only.golden").
			Execute()
	})

	t.Run("read-only server still reads", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			ReadOnly().
			CreateNamespace(namespace).
			CreateClusters(namespace, "only-cluster").
			ToolCall("capi_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "only-cluster").
			AssertContent("read_only_get_cluster.golden").
			Execute()
	})
}
