package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiVSphereGetCluster(t *testing.T) {
	t.Parallel()

	t.Run("should get vSphere cluster details", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-vsphere-cluster").WithProvider("vsphere").Create().
			ToolCall("capi_vsphere_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-vsphere-cluster").
			AssertContent("vsphere_cluster_details.golden").
			Execute()
	})

	t.Run("should error for non-vSphere cluster", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			ToolCall("capi_vsphere_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-aws-cluster").
			AssertError("not_vsphere_cluster.golden").
			Execute()
	})
}
