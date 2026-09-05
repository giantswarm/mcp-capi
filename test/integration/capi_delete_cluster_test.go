package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

// TestCapiDeleteCluster covers the deletion guards: GitOps ownership and the
// giantswarm.io/prevent-deletion label, which is honoured whatever the write
// policy says.
func TestCapiDeleteCluster(t *testing.T) {
	t.Parallel()

	t.Run("refuses a cluster labelled prevent-deletion", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "protected").
			WithLabels(map[string]string{"giantswarm.io/prevent-deletion": "true"}).
			Create().
			ToolCall("capi_delete_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "protected").
			WithArg("force", true).
			AssertError("prevent_deletion.golden").
			Execute()
	})

	t.Run("refuses a cluster labelled prevent-deletion even with the GitOps guard off", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			WithoutGitOpsGuard().
			CreateNamespace(namespace).
			Cluster(namespace, "protected").
			WithLabels(map[string]string{"giantswarm.io/prevent-deletion": "true"}).
			Create().
			ToolCall("capi_delete_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "protected").
			WithArg("force", true).
			AssertError("prevent_deletion_guard_off.golden").
			Execute()
	})

	t.Run("refuses a cluster rendered by a Helm release", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "gazelle").
			WithLabels(map[string]string{managedByLabel: managedByHelm}).
			WithAnnotations(map[string]string{
				"meta.helm.sh/release-name":      "gazelle",
				"meta.helm.sh/release-namespace": namespace,
			}).
			Create().
			ToolCall("capi_delete_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "gazelle").
			WithArg("force", true).
			AssertError("managed_by_helm.golden").
			Execute()
	})

	t.Run("is not offered by a read-only server", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			ReadOnly().
			CreateNamespace(namespace).
			CreateClusters(namespace, "hand-made").
			ToolCall("capi_delete_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "hand-made").
			WithArg("force", true).
			AssertError("read_only.golden").
			Execute()
	})
}
