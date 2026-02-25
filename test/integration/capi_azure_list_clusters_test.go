package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiAzureListClusters(t *testing.T) {
	t.Parallel()

	t.Run("should list Azure clusters", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-azure-cluster").WithProvider("azure").Create().
			ToolCall("capi_azure_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("azure_clusters.golden").
			Execute()
	})

	t.Run("should show no Azure clusters", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_azure_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("no_azure_clusters.golden").
			Execute()
	})

	t.Run("should list AzureManagedCluster clusters", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-managed-cluster").WithCustomInfraRef("AzureManagedCluster", "my-managed-cluster").Create().
			ToolCall("capi_azure_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("azure_managed_cluster.golden").
			Execute()
	})

	t.Run("should filter out non-Azure clusters", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-azure-cluster").WithProvider("azure").Create().
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			ToolCall("capi_azure_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("mixed_providers.golden").
			Execute()
	})
}
