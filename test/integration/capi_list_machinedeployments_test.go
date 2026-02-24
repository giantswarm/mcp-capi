package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiListMachineDeployments(t *testing.T) {
	t.Parallel()

	t.Run("lists machine deployments in namespace", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

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
		namespace := "test-clusters"

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
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_list_machinedeployments").
			WithArg("namespace", namespace).
			AssertContent("empty.golden").
			Execute()
	})

	t.Run("returns error without namespace argument", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_list_machinedeployments").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("lists single machine deployment", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

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
		namespace := "test-clusters"

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
		namespace := "test-clusters"

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
		namespace := "test-clusters"

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
}
