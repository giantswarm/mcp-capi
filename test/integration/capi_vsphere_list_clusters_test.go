package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiVSphereListClusters(t *testing.T) {
	t.Parallel()

	t.Run("should list vSphere clusters", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

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
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_vsphere_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("no_vsphere_clusters.golden").
			Execute()
	})

	t.Run("should filter out non-vSphere clusters", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-vsphere-cluster").WithProvider("vsphere").Create().
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			ToolCall("capi_vsphere_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("mixed_providers.golden").
			Execute()
	})
}
