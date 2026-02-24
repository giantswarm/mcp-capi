package integration_test

import (
	"fmt"
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiClusterStatus(t *testing.T) {
	t.Parallel()

	t.Run("returns error when namespace argument is missing", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_cluster_status").
			WithArg("name", "some-cluster").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("returns error when name argument is missing", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_cluster_status").
			WithArg("namespace", "some-ns").
			AssertError("missing_name.golden").
			Execute()
	})

	t.Run("returns error for non-existent cluster", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "does-not-exist").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("gets status of a basic cluster", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "only-cluster").
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "only-cluster").
			AssertContent("basic.golden").
			Execute()
	})

	t.Run("gets status with provider and phase", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "aws-cluster").WithProvider("aws").WithPhase("Provisioned").Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "aws-cluster").
			AssertContent("provider_and_phase.golden").
			Execute()
	})

	t.Run("gets status with version from topology", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "versioned-cluster").WithProvider("aws").WithVersion("v1.29.0").Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "versioned-cluster").
			AssertContent("version_from_topology.golden").
			Execute()
	})

	t.Run("gets status with partial machine readiness", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "partial-cluster").WithProvider("azure").WithMachines(3, 1).Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "partial-cluster").
			AssertContent("partial_machines.golden").
			Execute()
	})

	t.Run("gets status with all machines ready", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "all-ready-cluster").WithProvider("gcp").WithMachines(5, 5).Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "all-ready-cluster").
			AssertContent("all_machines_ready.golden").
			Execute()
	})

	t.Run("gets status with no machines ready", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "no-ready-cluster").WithProvider("vsphere").WithMachines(4, 0).Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "no-ready-cluster").
			AssertContent("no_machines_ready.golden").
			Execute()
	})

	t.Run("gets status with conditions", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "healthy-cluster").WithProvider("aws").
			WithCondition("Ready").True().Reason("ClusterReady").Message("Cluster is fully operational").Done().
			WithCondition("ControlPlaneReady").True().Reason("ControlPlaneInitialized").Message("Control plane is ready").Done().
			WithCondition("InfrastructureReady").True().Reason("InfrastructureProvisioned").Message("Infrastructure is ready").Done().
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "healthy-cluster").
			AssertContent("with_conditions.golden").
			Execute()
	})

	t.Run("gets status with unhealthy conditions", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "unhealthy-cluster").WithProvider("azure").
			WithCondition("Ready").False().Reason("ClusterNotReady").Message("Cluster has issues").Done().
			WithCondition("ControlPlaneReady").False().Reason("ControlPlaneUnhealthy").Message("Control plane has unhealthy replicas").Done().
			WithCondition("InfrastructureReady").False().Reason("WaitingForInfrastructure").Message("Infrastructure provisioning failed").Done().
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "unhealthy-cluster").
			AssertContent("unhealthy_conditions.golden").
			Execute()
	})

	t.Run("gets status with version from control plane", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "kcp-cluster").
			WithProvider("aws").
			WithKubeadmControlPlane().Version("v1.28.0").Done().
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "kcp-cluster").
			AssertContent("version_from_control_plane.golden").
			Execute()
	})

	t.Run("gets status with version precedence over control plane", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "precedence-cluster").
			WithProvider("aws").
			WithVersion("v1.28.0").
			WithKubeadmControlPlane().Version("v1.99.0").Done().
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "precedence-cluster").
			AssertContent("version_precedence.golden").
			Execute()
	})

	t.Run("gets status with all properties", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "full-cluster").WithProvider("aws").WithPhase("Provisioned").WithVersion("v1.29.0").WithMachines(3, 3).
			WithCondition("Ready").True().Reason("ClusterReady").Message("Cluster is fully operational").Done().
			WithCondition("ControlPlaneReady").True().Reason("ControlPlaneInitialized").Message("Control plane is ready").Done().
			WithCondition("InfrastructureReady").True().Reason("InfrastructureProvisioned").Message("Infrastructure is ready").Done().
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "full-cluster").
			AssertContent("all_properties.golden").
			Execute()
	})

	t.Run("gets status with unknown infrastructure provider", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "custom-cluster").
			WithCustomInfraRef("DOCluster", "custom-cluster").
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "custom-cluster").
			AssertContent("unknown_infra_kind.golden").
			Execute()
	})

	t.Run("gets status with non-kubeadm control plane", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "rosa-cluster").
			WithProvider("aws").
			WithControlPlaneRef("ROSAControlPlane", "rosa-cp").
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "rosa-cluster").
			AssertContent("non_kubeadm_control_plane.golden").
			Execute()
	})

	t.Run("gets status with missing control plane resource", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "missing-cp-cluster").
			WithProvider("aws").
			WithControlPlaneRef("KubeadmControlPlane", "missing-cp-cluster-control-plane").
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "missing-cp-cluster").
			AssertContent("missing_control_plane.golden").
			Execute()
	})

	for _, provider := range providers {
		t.Run(fmt.Sprintf("gets %s cluster status with all properties", provider), func(t *testing.T) {
			t.Parallel()
			namespace := "test-clusters"

			harness.New(t).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-full-cluster").WithProvider(provider).WithPhase("Provisioned").WithVersion("v1.29.0").WithMachines(3, 2).
				WithCondition("Ready").True().Reason("ClusterReady").Message("Cluster is fully operational").Done().
				WithCondition("ControlPlaneReady").True().Reason("ControlPlaneInitialized").Message("Control plane is ready").Done().
				WithCondition("InfrastructureReady").True().Reason("InfrastructureProvisioned").Message("Infrastructure is ready").Done().
				Create().
				ToolCall("capi_cluster_status").
				WithArg("namespace", namespace).
				WithArg("name", provider+"-full-cluster").
				AssertContent(provider+"_all_properties.golden").
				Execute()
		})

		t.Run(fmt.Sprintf("gets %s cluster status with version from control plane", provider), func(t *testing.T) {
			t.Parallel()
			namespace := "test-clusters"

			harness.New(t).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-kcp-cluster").
				WithProvider(provider).
				WithKubeadmControlPlane().Version("v1.30.0").Done().
				Create().
				ToolCall("capi_cluster_status").
				WithArg("namespace", namespace).
				WithArg("name", provider+"-kcp-cluster").
				AssertContent(provider+"_control_plane_version.golden").
				Execute()
		})
	}

	t.Run("gets status with mixed conditions ready true cp false", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "mixed-1").WithProvider("aws").WithPhase("Provisioned").
			WithCondition("Ready").True().Reason("ClusterReady").Message("Cluster is ready").Done().
			WithCondition("ControlPlaneReady").False().Reason("CPNotReady").Message("Control plane not ready").Done().
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "mixed-1").
			AssertContent("mixed_ready_true_cp_false.golden").
			Execute()
	})

	t.Run("gets status with mixed conditions ready false infra true", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "mixed-2").WithProvider("azure").WithPhase("Provisioning").
			WithCondition("Ready").False().Reason("ClusterNotReady").Message("Cluster not ready").Done().
			WithCondition("InfrastructureReady").True().Reason("InfraReady").Message("Infrastructure is ready").Done().
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "mixed-2").
			AssertContent("mixed_ready_false_infra_true.golden").
			Execute()
	})

	t.Run("gets status with no conditions", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "nocond-cluster").WithProvider("gcp").WithPhase("Pending").
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "nocond-cluster").
			AssertContent("no_conditions.golden").
			Execute()
	})

	t.Run("gets status with nil control plane ref", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "nocp-cluster").WithProvider("aws").WithPhase("Provisioned").WithVersion("v1.29.0").
			WithCondition("Ready").True().Reason("ClusterReady").Message("Cluster is ready").Done().
			WithCondition("InfrastructureReady").True().Reason("InfraReady").Message("Infrastructure is ready").Done().
			Create().
			ToolCall("capi_cluster_status").
			WithArg("namespace", namespace).
			WithArg("name", "nocp-cluster").
			AssertContent("nil_control_plane_ref.golden").
			Execute()
	})
}
