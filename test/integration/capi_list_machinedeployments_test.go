package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiListMachineDeployments(t *testing.T) {
	t.Parallel()

	t.Run("lists machine deployments in namespace", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineDeployment(namespace, "md-1").ForCluster("test-cluster").WithReplicas(3).WithVersion("v1.29.0").
			WithStatus(3, 3, 3, 3).Create().
			MachineDeployment(namespace, "md-2").ForCluster("test-cluster").WithReplicas(2).WithVersion("v1.28.0").
			WithStatus(2, 1, 2, 1).Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("multiple.golden").
			Execute()
	})

	t.Run("filters machine deployments by cluster name", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "cluster-a", "cluster-b").
			MachineDeployment(namespace, "md-a").ForCluster("cluster-a").WithReplicas(3).Create().
			MachineDeployment(namespace, "md-b").ForCluster("cluster-b").WithReplicas(2).Create().
			ToolCall("capi_list_machinedeployments").
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
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("empty.golden").
			Execute()
	})

	t.Run("returns error without namespace argument", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_list_machinedeployments").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("lists single machine deployment", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineDeployment(namespace, "only-md").ForCluster("test-cluster").WithReplicas(3).WithVersion("v1.29.0").
			WithStatus(3, 2, 3, 2).Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("single.golden").
			Execute()
	})

	t.Run("lists machine deployments with phase", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineDeployment(namespace, "md-running").ForCluster("test-cluster").WithReplicas(3).
			WithPhase("Running").WithStatus(3, 3, 3, 3).Create().
			MachineDeployment(namespace, "md-scaling").ForCluster("test-cluster").WithReplicas(5).
			WithPhase("ScalingUp").WithStatus(3, 3, 3, 3).Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("with_phases.golden").
			Execute()
	})

	t.Run("lists machine deployments with version", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineDeployment(namespace, "md-v128").ForCluster("test-cluster").WithReplicas(3).
			WithVersion("v1.28.0").Create().
			MachineDeployment(namespace, "md-v129").ForCluster("test-cluster").WithReplicas(2).
			WithVersion("v1.29.0").Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("with_versions.golden").
			Execute()
	})

	t.Run("lists machine deployments across clusters", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "cluster-1", "cluster-2").
			MachineDeployment(namespace, "md-c1").ForCluster("cluster-1").WithReplicas(3).WithVersion("v1.29.0").
			WithStatus(3, 3, 3, 3).Create().
			MachineDeployment(namespace, "md-c2").ForCluster("cluster-2").WithReplicas(5).WithVersion("v1.28.0").
			WithStatus(5, 4, 5, 4).Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("across_clusters.golden").
			Execute()
	})

	t.Run("lists machine deployment with zero status replicas", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineDeployment(namespace, "md-new").ForCluster("test-cluster").WithReplicas(3).WithVersion("v1.29.0").
			Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("zero_status_replicas.golden").
			Execute()
	})

	t.Run("lists machine deployment without version", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineDeployment(namespace, "md-noversion").ForCluster("test-cluster").WithReplicas(2).
			WithStatus(2, 2, 2, 2).Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("no_version.golden").
			Execute()
	})

	t.Run("lists machine deployment with mismatched replica counts", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineDeployment(namespace, "md-rolling").ForCluster("test-cluster").WithReplicas(5).WithVersion("v1.29.0").
			WithStatus(5, 2, 3, 2).Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("mismatched_replicas.golden").
			Execute()
	})

	t.Run("lists machine deployment with nil replicas", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineDeployment(namespace, "md-nil-replicas").ForCluster("test-cluster").WithNilReplicas().
			WithPhase("Running").Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("nil_replicas.golden").
			Execute()
	})

	t.Run("should return empty for non-existent cluster filter", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "real-cluster").
			MachineDeployment(namespace, "md-real").ForCluster("real-cluster").WithReplicas(3).
			WithStatus(3, 3, 3, 3).Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			WithArg("clusterName", "nonexistent").
			AssertContent("filter_nonexistent_cluster.golden").
			Execute()
	})

	t.Run("should handle zero replicas status", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineDeployment(namespace, "md-zero").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(0, 0, 0, 0).Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("zero_replicas_status.golden").
			Execute()
	})

	t.Run("should show deployment without phase", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineDeployment(namespace, "md-nophase").ForCluster("test-cluster").WithReplicas(2).
			WithStatus(2, 2, 2, 2).Create().
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("no_phase.golden").
			Execute()
	})
}
