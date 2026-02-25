package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiAWSListClusters(t *testing.T) {
	t.Parallel()

	t.Run("should list AWS clusters", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			Cluster(namespace, "my-azure-cluster").WithProvider("azure").Create().
			ToolCall("capi_aws_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("aws_clusters.golden").
			Execute()
	})

	t.Run("should show no AWS clusters when none exist", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_aws_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("no_aws_clusters.golden").
			Execute()
	})
}
