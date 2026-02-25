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
}
