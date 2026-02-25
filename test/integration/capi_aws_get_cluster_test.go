package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiAWSGetCluster(t *testing.T) {
	t.Parallel()

	t.Run("should get AWS cluster details", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			ToolCall("capi_aws_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-aws-cluster").
			AssertContent("aws_cluster_details.golden").
			Execute()
	})

	t.Run("should error for non-AWS cluster", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-azure-cluster").WithProvider("azure").Create().
			ToolCall("capi_aws_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-azure-cluster").
			AssertError("not_aws_cluster.golden").
			Execute()
	})

	t.Run("should error when cluster not found", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_aws_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "nonexistent-cluster").
			AssertError("not_found.golden").
			Execute()
	})
}
