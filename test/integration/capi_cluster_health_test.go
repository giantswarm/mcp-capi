package integration_test

import (
	"testing"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiClusterHealth(t *testing.T) {
	t.Parallel()

	t.Run("returns error for non-existent cluster", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "does-not-exist").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("returns unhealthy status for a bare cluster without status", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "bare-cluster").
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "bare-cluster").
			AssertContent("bare_cluster.golden").
			Execute()
	})

	t.Run("returns unhealthy status when control plane is not ready", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

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
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "partial-health").WithProvider("azure").WithPhase("Provisioned").
			WithControlPlaneReady(true).
			WithInfraReady(true).
			WithMachines(3, 1).
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "partial-health").
			AssertContent("partial_machines.golden").
			Execute()
	})

	t.Run("returns healthy status with all machines ready and provisioned phase", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "fully-healthy").WithProvider("aws").WithPhase("Provisioned").
			WithControlPlaneReady(true).
			WithInfraReady(true).
			WithMachines(3, 3).
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
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "pending-cluster").WithProvider("gcp").WithPhase("Pending").Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "pending-cluster").
			AssertContent("pending_phase.golden").
			Execute()
	})

	t.Run("returns error when namespace argument is missing", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_cluster_health").
			WithArg("name", "some-cluster").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("returns error when name argument is missing", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			AssertError("missing_name.golden").
			Execute()
	})

	t.Run("shows infra-specific recommendations when CP ready but infra not ready", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "cp-ok-infra-bad").WithProvider("aws").WithPhase("Provisioned").
			WithControlPlaneReady(true).
			WithInfraReady(false).
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "cp-ok-infra-bad").
			AssertContent("cp_ready_infra_not_ready.golden").
			Execute()
	})

	t.Run("shows CP-specific recommendations when CP not ready but infra ready", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "cp-bad-infra-ok").WithProvider("aws").WithPhase("Provisioned").
			WithControlPlaneReady(false).
			WithInfraReady(true).
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "cp-bad-infra-ok").
			AssertContent("cp_not_ready_infra_ready.golden").
			Execute()
	})

	t.Run("shows only worker recommendations when CP and infra ready but no machines", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "workers-only-bad").WithProvider("aws").WithPhase("Provisioned").
			WithControlPlaneReady(true).
			WithInfraReady(true).
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "workers-only-bad").
			AssertContent("cp_infra_ready_workers_not_ready.golden").
			Execute()
	})

	t.Run("shows condition with Error severity in Issues section", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "error-condition").WithProvider("aws").
			WithCondition("NetworkReady").False().Severity(clusterv1.ConditionSeverityError).Reason("NetworkFailed").Message("Network configuration failed").Done().
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "error-condition").
			AssertContent("condition_severity_error.golden").
			Execute()
	})

	t.Run("shows condition with Warning severity in Warnings section", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "warning-condition").WithProvider("aws").
			WithCondition("CertificatesExpiring").False().Severity(clusterv1.ConditionSeverityWarning).Reason("CertsExpiringSoon").Message("Certificates will expire in 30 days").Done().
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "warning-condition").
			AssertContent("condition_severity_warning.golden").
			Execute()
	})

	t.Run("reports workers not ready when zero machines exist", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "zero-machines").WithProvider("aws").WithPhase("Provisioned").
			WithControlPlaneReady(true).
			WithInfraReady(true).
			WithMachines(0, 0).
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "zero-machines").
			AssertContent("zero_machines.golden").
			Execute()
	})

	t.Run("returns warning for Failed phase", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "failed-cluster").WithProvider("aws").WithPhase("Failed").Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "failed-cluster").
			AssertContent("failed_phase.golden").
			Execute()
	})

	t.Run("returns warning for Deleting phase", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "deleting-cluster").WithProvider("aws").WithPhase("Deleting").Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "deleting-cluster").
			AssertContent("deleting_phase.golden").
			Execute()
	})

	t.Run("does not generate phase warning for empty phase", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "empty-phase").WithProvider("aws").
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "empty-phase").
			AssertContent("empty_phase.golden").
			Execute()
	})

	t.Run("shows both issues and warnings for combined error and warning conditions", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "combined-conditions").WithProvider("aws").
			WithCondition("NetworkReady").False().Severity(clusterv1.ConditionSeverityError).Reason("NetworkFailed").Message("Network configuration failed").Done().
			WithCondition("CertificatesExpiring").False().Severity(clusterv1.ConditionSeverityWarning).Reason("CertsExpiringSoon").Message("Certificates will expire in 30 days").Done().
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "combined-conditions").
			AssertContent("combined_issues_and_warnings.golden").
			Execute()
	})

	t.Run("shows all recommendation sections when CP infra and workers are all not ready", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "all-unhealthy").WithProvider("aws").WithPhase("Provisioning").
			WithControlPlaneReady(false).
			WithInfraReady(false).
			WithMachines(3, 0).
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "all-unhealthy").
			AssertContent("all_recommendations.golden").
			Execute()
	})

	t.Run("should show multiple error severity conditions", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "multi-error").WithProvider("aws").
			WithCondition("NetworkReady").False().Severity(clusterv1.ConditionSeverityError).Reason("NetworkFailed").Message("Network configuration failed").Done().
			WithCondition("StorageReady").False().Severity(clusterv1.ConditionSeverityError).Reason("StorageFailed").Message("Storage provisioning failed").Done().
			Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "multi-error").
			AssertContent("multiple_error_conditions.golden").
			Execute()
	})

	t.Run("should show provisioning phase warning", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "provisioning-cluster").WithProvider("aws").WithPhase("Provisioning").Create().
			ToolCall("capi_cluster_health").
			WithArg("namespace", namespace).
			WithArg("name", "provisioning-cluster").
			AssertContent("provisioning_phase.golden").
			Execute()
	})
}
