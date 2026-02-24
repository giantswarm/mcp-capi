package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiClusterHealth(t *testing.T) {
	t.Parallel()

	t.Run("returns error for non-existent cluster", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "does-not-exist").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("returns healthy status for basic cluster", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "healthy-basic").
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "healthy-basic").
			AssertContent("healthy_basic.golden").
			Execute()
	})

	t.Run("returns unhealthy status when control plane is not ready", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "unhealthy-cp").WithProvider("aws").
			WithCondition("Ready").False().Reason("ClusterNotReady").Message("Cluster has issues").Done().
			WithCondition("ControlPlaneReady").False().Reason("ControlPlaneUnhealthy").Message("Control plane has unhealthy replicas").Done().
			WithCondition("InfrastructureReady").False().Reason("WaitingForInfrastructure").Message("Infrastructure provisioning failed").Done().
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "unhealthy-cp").
			AssertContent("unhealthy_control_plane.golden").
			Execute()
	})

	t.Run("returns unhealthy status with partial machine readiness", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "partial-health").WithProvider("azure").WithMachines(3, 1).Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "partial-health").
			AssertContent("partial_machines.golden").
			Execute()
	})

	t.Run("returns healthy status with all machines ready and provisioned phase", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "fully-healthy").WithProvider("aws").WithPhase("Provisioned").WithMachines(3, 3).
			WithCondition("Ready").True().Reason("ClusterReady").Message("Cluster is fully operational").Done().
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "fully-healthy").
			AssertContent("fully_healthy.golden").
			Execute()
	})

	t.Run("returns warning for non-provisioned phase", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "pending-cluster").WithProvider("gcp").WithPhase("Pending").Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "pending-cluster").
			AssertContent("pending_phase.golden").
			Execute()
	})
}
