package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiGetMachineSet(t *testing.T) {
	t.Parallel()

	t.Run("returns error for non-existent machine set", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			WithArg("name", "does-not-exist").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("returns error without namespace argument", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_get_machineset").
			WithArg("name", "some-ms").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("returns error without name argument", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			AssertError("missing_name.golden").
			Execute()
	})

	t.Run("gets a basic machine set", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "basic-ms").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 2, 2).Create().
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			WithArg("name", "basic-ms").
			AssertContent("basic.golden").
			Execute()
	})

	t.Run("gets machine set with version", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "versioned-ms").ForCluster("test-cluster").WithReplicas(3).
			WithVersion("v1.29.0").WithStatus(3, 3, 3).Create().
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			WithArg("name", "versioned-ms").
			AssertContent("with_version.golden").
			Execute()
	})

	t.Run("gets machine set with infrastructure reference", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "infra-ms").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 3, 3).
			WithInfraRef("AWSMachineTemplate", "aws-mt").Create().
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			WithArg("name", "infra-ms").
			AssertContent("with_infra_ref.golden").
			Execute()
	})

	t.Run("gets machine set with bootstrap reference", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "bootstrap-ms").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 3, 3).
			WithBootstrapRef("KubeadmConfigTemplate", "bootstrap-config").Create().
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			WithArg("name", "bootstrap-ms").
			AssertContent("with_bootstrap_ref.golden").
			Execute()
	})

	t.Run("gets machine set with owner reference", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "owned-ms").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 2, 2).
			OwnedBy("parent-md").Create().
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			WithArg("name", "owned-ms").
			AssertContent("with_owner.golden").
			Execute()
	})

	t.Run("gets machine set with all properties", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "full-ms").ForCluster("test-cluster").WithReplicas(5).
			WithVersion("v1.29.0").
			WithStatus(5, 4, 3).
			WithInfraRef("AWSMachineTemplate", "aws-mt").
			WithBootstrapRef("KubeadmConfigTemplate", "bootstrap-config").
			OwnedBy("parent-md").Create().
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			WithArg("name", "full-ms").
			AssertContent("all_properties.golden").
			Execute()
	})

	t.Run("gets machine set with zero ready replicas", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "no-ready-ms").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 0, 0).Create().
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			WithArg("name", "no-ready-ms").
			AssertContent("no_ready_replicas.golden").
			Execute()
	})

	t.Run("gets machine set with failure reason", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "failure-reason-ms").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 1, 1).
			WithFailureReason("InvalidConfiguration").Create().
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			WithArg("name", "failure-reason-ms").
			AssertContent("with_failure_reason.golden").
			Execute()
	})

	t.Run("gets machine set with failure message", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "failure-message-ms").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 0, 0).
			WithFailureMessage("unable to create machines: quota exceeded").Create().
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			WithArg("name", "failure-message-ms").
			AssertContent("with_failure_message.golden").
			Execute()
	})

	t.Run("gets machine set with both failure reason and message", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster").
			MachineSet(namespace, "failure-both-ms").ForCluster("test-cluster").WithReplicas(3).
			WithStatus(3, 0, 0).
			WithFailureReason("InvalidConfiguration").
			WithFailureMessage("spec.template.spec.infrastructureRef is required").Create().
			ToolCall("capi_get_machineset").
			WithArg("namespace", namespace).
			WithArg("name", "failure-both-ms").
			AssertContent("with_failure_reason_and_message.golden").
			Execute()
	})
}
