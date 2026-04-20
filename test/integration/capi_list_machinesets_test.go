package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiListMachineSets(t *testing.T) {
	t.Parallel()

	t.Run("lists machine sets in namespace", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "ms-1").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 3, 3).Create().
			MachineSet(namespace, "ms-2").ForCluster("test-cluster").WithReplicas(2).
			WithStatus(2, 1, 1).Create().
			ToolCall("capi_list_machinesets").
			WithArg("namespace", namespace).
			AssertContent("multiple.golden").
			Execute()
	})

	t.Run("filters machine sets by cluster name", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "cluster-a", "cluster-b").
			MachineSet(namespace, "ms-a").ForCluster("cluster-a").WithReplicas(3).
			WithStatus(3, 3, 3).Create().
			MachineSet(namespace, "ms-b").ForCluster("cluster-b").WithReplicas(2).
			WithStatus(2, 2, 2).Create().
			ToolCall("capi_list_machinesets").
			WithArg("namespace", namespace).
			WithArg("clusterName", "cluster-a").
			AssertContent("filter_by_cluster.golden").
			Execute()
	})

	t.Run("handles empty namespace", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_list_machinesets").
			WithArg("namespace", namespace).
			AssertContent("empty.golden").
			Execute()
	})

	t.Run("returns error without namespace argument", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_list_machinesets").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("lists machine set with owner reference", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "ms-owned").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 2, 2).
			OwnedBy("parent-md").Create().
			ToolCall("capi_list_machinesets").
			WithArg("namespace", namespace).
			AssertContent("with_owner.golden").
			Execute()
	})

	t.Run("lists machine set with infrastructure reference", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "ms-infra").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 3, 3).
			WithInfraRef("AWSMachineTemplate", "aws-machine-template").Create().
			ToolCall("capi_list_machinesets").
			WithArg("namespace", namespace).
			AssertContent("with_infra_ref.golden").
			Execute()
	})

	t.Run("lists single machine set", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "only-ms").ForCluster("test-cluster").WithReplicas(2).
			WithStatus(2, 2, 2).Create().
			ToolCall("capi_list_machinesets").
			WithArg("namespace", namespace).
			AssertContent("single.golden").
			Execute()
	})

	t.Run("lists machine sets across clusters", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "cluster-1", "cluster-2").
			MachineSet(namespace, "ms-c1").ForCluster("cluster-1").WithReplicas(3).
			WithStatus(3, 3, 3).Create().
			MachineSet(namespace, "ms-c2").ForCluster("cluster-2").WithReplicas(5).
			WithStatus(5, 4, 4).Create().
			ToolCall("capi_list_machinesets").
			WithArg("namespace", namespace).
			AssertContent("across_clusters.golden").
			Execute()
	})

	t.Run("lists machine set with nil replicas", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "ms-nil-replicas").ForCluster("test-cluster").WithNilReplicas().
			WithStatus(2, 1, 1).Create().
			ToolCall("capi_list_machinesets").
			WithArg("namespace", namespace).
			AssertContent("nil_replicas.golden").
			Execute()
	})

	t.Run("lists machine set with non-machinedeployment owner", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "ms-custom-owner").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 3, 3).
			OwnedByKind("MachinePool", "my-machinepool").Create().
			ToolCall("capi_list_machinesets").
			WithArg("namespace", namespace).
			AssertContent("non_md_owner.golden").
			Execute()
	})

	t.Run("lists orphaned machine set without owner references", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "ms-orphan").ForCluster("test-cluster").WithReplicas(2).
			WithStatus(2, 2, 2).Create().
			ToolCall("capi_list_machinesets").
			WithArg("namespace", namespace).
			AssertContent("orphaned.golden").
			Execute()
	})

	t.Run("should return empty for non-existent cluster filter", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "real-cluster").
			MachineSet(namespace, "ms-real").ForCluster("real-cluster").WithReplicas(3).
			WithStatus(3, 3, 3).Create().
			ToolCall("capi_list_machinesets").
			WithArg("namespace", namespace).
			WithArg("clusterName", "nonexistent").
			AssertContent("filter_nonexistent_cluster.golden").
			Execute()
	})
}
