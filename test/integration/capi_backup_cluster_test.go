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
			WithArg("namespace", "test-clusters").
			AssertError("missing_name.golden").
			Execute()
	})

	t.Run("returns error for non-existent cluster", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

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
		namespace := "test-clusters"

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
		namespace := "test-clusters"

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
		namespace := "test-clusters"

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
		namespace := "test-clusters"

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
}
