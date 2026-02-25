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

	t.Run("should error when cluster not found", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_vsphere_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "nonexistent-cluster").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("should error when namespace is missing", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_vsphere_get_cluster").
			WithArg("name", "my-cluster").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("should error when name is missing", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_vsphere_get_cluster").
			WithArg("namespace", namespace).
			AssertError("missing_name.golden").
			Execute()
	})

	t.Run("should error for cluster with nil infrastructure ref", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "no-infra-cluster").Create().
			ToolCall("capi_vsphere_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "no-infra-cluster").
			AssertError("nil_infra_ref.golden").
			Execute()
	})
}
