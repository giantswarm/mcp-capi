package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiAzureGetCluster(t *testing.T) {
	t.Parallel()

	t.Run("should get Azure cluster details", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-azure-cluster").WithProvider("azure").Create().
			ToolCall("capi_azure_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-azure-cluster").
			AssertContent("azure_cluster_details.golden").
			Execute()
	})

	t.Run("should error for non-Azure cluster", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			ToolCall("capi_azure_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-aws-cluster").
			AssertError("not_azure_cluster.golden").
			Execute()
	})

	t.Run("should error when cluster not found", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_azure_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "nonexistent-cluster").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("should error when namespace is missing", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_azure_get_cluster").
			WithArg("name", "my-cluster").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("should error when name is missing", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_azure_get_cluster").
			WithArg("namespace", namespace).
			AssertError("missing_name.golden").
			Execute()
	})

	t.Run("should accept AzureManagedCluster kind", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-managed-cluster").WithCustomInfraRef("AzureManagedCluster", "my-managed-cluster").Create().
			ToolCall("capi_azure_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-managed-cluster").
			AssertContent("azure_managed_cluster_details.golden").
			Execute()
	})

	t.Run("should error for cluster with nil infrastructure ref", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "no-infra-cluster").Create().
			ToolCall("capi_azure_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "no-infra-cluster").
			AssertError("nil_infra_ref.golden").
			Execute()
	})
}
