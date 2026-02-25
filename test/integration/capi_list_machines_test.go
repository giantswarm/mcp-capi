package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiListMachines(t *testing.T) {
	t.Parallel()

	t.Run("lists machines in namespace", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-cluster").WithProvider("aws").WithMachines(3, 2).Create().
			ToolCall("capi_list_machines").
			WithArg("namespace", namespace).
			AssertContent("list_in_namespace.golden").
			Execute()
	})

	t.Run("filters machines by cluster name", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "cluster-a").WithProvider("aws").WithMachines(2, 1).Create().
			Cluster(namespace, "cluster-b").WithProvider("azure").WithMachines(2, 2).Create().
			ToolCall("capi_list_machines").
			WithArg("namespace", namespace).
			WithArg("clusterName", "cluster-a").
			AssertContent("filter_by_cluster.golden").
			Execute()
	})

	t.Run("returns empty list when no machines exist", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_list_machines").
			WithArg("namespace", namespace).
			AssertContent("empty.golden").
			Execute()
	})

	t.Run("lists machines with all ready", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "healthy-cluster").WithProvider("gcp").WithMachines(3, 3).Create().
			ToolCall("capi_list_machines").
			WithArg("namespace", namespace).
			AssertContent("all_ready.golden").
			Execute()
	})

	t.Run("lists machines with none ready", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "unhealthy-cluster").WithProvider("vsphere").WithMachines(3, 0).Create().
			ToolCall("capi_list_machines").
			WithArg("namespace", namespace).
			AssertContent("none_ready.golden").
			Execute()
	})

	t.Run("returns error when namespace argument is missing", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_list_machines").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("lists machines from multiple clusters in same namespace", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "cluster-x").WithProvider("aws").WithMachines(2, 1).Create().
			Cluster(namespace, "cluster-y").WithProvider("azure").WithMachines(1, 1).Create().
			ToolCall("capi_list_machines").
			WithArg("namespace", namespace).
			AssertContent("multiple_clusters.golden").
			Execute()
	})

	t.Run("returns empty list when filtering by non-existent cluster", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "real-cluster").WithProvider("aws").WithMachines(2, 1).Create().
			ToolCall("capi_list_machines").
			WithArg("namespace", namespace).
			WithArg("clusterName", "non-existent-cluster").
			AssertContent("filter_non_existent_cluster.golden").
			Execute()
	})

	t.Run("lists single machine", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "single-machine-cluster").WithProvider("aws").WithMachines(1, 1).Create().
			ToolCall("capi_list_machines").
			WithArg("namespace", namespace).
			AssertContent("single_machine.golden").
			Execute()
	})

	t.Run("should show machine with provider ID", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "pid-cluster").
			Machine(namespace, "pid-machine").ForCluster("pid-cluster").
			WithProviderID("aws://us-east-1/i-1234567890abcdef0").
			WithPhase("Running").Create().
			ToolCall("capi_list_machines").
			WithArg("namespace", namespace).
			AssertContent("with_provider_id.golden").
			Execute()
	})

	t.Run("should show machine with empty phase", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "nophase-cluster").
			Machine(namespace, "nophase-machine").ForCluster("nophase-cluster").Create().
			ToolCall("capi_list_machines").
			WithArg("namespace", namespace).
			AssertContent("empty_phase.golden").
			Execute()
	})

	t.Run("should show machine with node ref", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "noderef-cluster").
			Machine(namespace, "noderef-machine").ForCluster("noderef-cluster").
			WithNodeRef("worker-node-1").
			WithPhase("Running").Create().
			ToolCall("capi_list_machines").
			WithArg("namespace", namespace).
			AssertContent("with_node_ref.golden").
			Execute()
	})
}
