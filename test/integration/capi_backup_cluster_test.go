package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiBackupCluster(t *testing.T) {
	t.Parallel()

	t.Run("returns error without namespace argument", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_backup_cluster").
			WithArg("name", "some-cluster").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("returns error without name argument", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_backup_cluster").
			WithArg("namespace", testNamespace).
			AssertError("missing_name.golden").
			Execute()
	})

	t.Run("returns error for non-existent cluster", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_backup_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "does-not-exist").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("backs up cluster with default settings", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "backup-cluster").
			ToolCall("capi_backup_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "backup-cluster").
			AssertContentNormalized("default_backup.golden", harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("backs up cluster with JSON output format", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "json-cluster").
			ToolCall("capi_backup_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "json-cluster").
			WithArg("output_format", "json").
			AssertContentNormalized("json_format.golden", harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("backs up cluster with secrets included", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "secrets-cluster").
			ToolCall("capi_backup_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "secrets-cluster").
			WithArg("include_secrets", true).
			AssertContentNormalized("include_secrets.golden", harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("backs up cluster with JSON format and secrets", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "full-backup-cluster").
			ToolCall("capi_backup_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "full-backup-cluster").
			WithArg("output_format", "json").
			WithArg("include_secrets", true).
			AssertContentNormalized("json_with_secrets.golden", harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("should exclude secrets when include_secrets is false", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "no-secrets-cluster").
			CreateSecret(namespace, "no-secrets-cluster-kubeconfig", map[string][]byte{
				"value": []byte("apiVersion: v1\nclusters:\n- cluster:\n    server: https://example.com\n"),
			}).
			ToolCall("capi_backup_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "no-secrets-cluster").
			WithArg("include_secrets", false).
			AssertContentNormalized("backup_secrets_excluded.golden", harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("should error on invalid output format", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "xml-format-cluster").
			ToolCall("capi_backup_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "xml-format-cluster").
			WithArg("output_format", "xml").
			AssertContentNormalized("backup_invalid_format.golden", harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("should backup cluster with rich state", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "rich-backup-cluster").
			WithProvider("aws").
			WithPhase("Provisioned").
			WithMachines(3, 2).
			WithCondition("Ready").True().Reason("ClusterReady").Message("Cluster is fully operational").Done().
			WithCondition("InfrastructureReady").
			True().Reason("InfrastructureProvisioned").Message("Infrastructure is ready").Done().
			Create().
			ToolCall("capi_backup_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "rich-backup-cluster").
			AssertContentNormalized("backup_rich_state.golden", harness.NormalizeTimestamp).
			Execute()
	})
}
