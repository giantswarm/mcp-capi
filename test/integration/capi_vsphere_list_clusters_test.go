package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiVSphereListClusters(t *testing.T) {
	t.Parallel()

	t.Run("should list vSphere clusters", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-vsphere-cluster").WithProvider("vsphere").Create().
			ToolCall("capi_vsphere_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("vsphere_clusters.golden").
			Execute()
	})

	t.Run("should show no vSphere clusters", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_vsphere_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("no_vsphere_clusters.golden").
			Execute()
	})
}
