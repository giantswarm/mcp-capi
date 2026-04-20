package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiGCPListClusters(t *testing.T) {
	t.Parallel()

	t.Run("should list GCP clusters", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-gcp-cluster").WithProvider("gcp").Create().
			ToolCall("capi_gcp_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("gcp_clusters.golden").
			Execute()
	})

	t.Run("should show no GCP clusters", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_gcp_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("no_gcp_clusters.golden").
			Execute()
	})

	t.Run("should list GCPManagedCluster clusters", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-managed-cluster").WithCustomInfraRef("GCPManagedCluster", "my-managed-cluster").Create().
			ToolCall("capi_gcp_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("gcp_managed_cluster.golden").
			Execute()
	})
}
