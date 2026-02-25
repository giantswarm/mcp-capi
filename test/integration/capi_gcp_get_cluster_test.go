package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiGCPGetCluster(t *testing.T) {
	t.Parallel()

	t.Run("should get GCP cluster details", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-gcp-cluster").WithProvider("gcp").Create().
			ToolCall("capi_gcp_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-gcp-cluster").
			AssertContent("gcp_cluster_details.golden").
			Execute()
	})

	t.Run("should error for non-GCP cluster", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			ToolCall("capi_gcp_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-aws-cluster").
			AssertError("not_gcp_cluster.golden").
			Execute()
	})
}
