package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiGetMachine(t *testing.T) {
	t.Parallel()

	t.Run("gets an existing machine", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-cluster").WithProvider("aws").WithMachines(1, 1).Create().
			ToolCall("capi_get_machine").
			WithArg("namespace", namespace).
			WithArg("name", "my-cluster-machine-0").
			AssertContent("existing_machine.golden").
			Execute()
	})

	t.Run("gets a machine without node ref", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "pending-cluster").WithProvider("azure").WithMachines(1, 0).Create().
			ToolCall("capi_get_machine").
			WithArg("namespace", namespace).
			WithArg("name", "pending-cluster-machine-0").
			AssertContent("machine_without_noderef.golden").
			Execute()
	})

	t.Run("returns error for non-existent machine", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_get_machine").
			WithArg("namespace", namespace).
			WithArg("name", "does-not-exist").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("returns error when namespace argument is missing", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_get_machine").
			WithArg("name", "some-machine").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("returns error when name argument is missing", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_get_machine").
			WithArg("namespace", namespace).
			AssertError("missing_name.golden").
			Execute()
	})

	t.Run("gets machine from cluster with multiple machines", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "multi-machine").WithProvider("gcp").WithMachines(3, 2).Create().
			ToolCall("capi_get_machine").
			WithArg("namespace", namespace).
			WithArg("name", "multi-machine-machine-0").
			AssertContent("machine_from_multi.golden").
			Execute()
	})

	t.Run("gets machine with all optional fields", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "full-cluster").
			Machine(namespace, "full-machine").ForCluster("full-cluster").
			WithPhase("Running").
			WithVersion("v1.29.0").
			WithProviderID("aws:///us-east-1/i-1234567890").
			WithNodeRef("full-machine-node").
			WithConfigRef("KubeadmConfig", "full-machine-bootstrap").
			WithInfraRef("AWSMachine", "full-machine-infra").
			WithCondition("Ready").True().Reason("MachineReady").Message("Machine is ready").Done().
			WithCondition("InfrastructureReady").True().Reason("InfraReady").Message("Infra provisioned").Done().
			WithAddress("InternalIP", "10.0.1.5").
			WithAddress("ExternalIP", "54.123.45.67").
			Create().
			ToolCall("capi_get_machine").
			WithArg("namespace", namespace).
			WithArg("name", "full-machine").
			AssertContent("all_fields.golden").
			Execute()
	})

	t.Run("gets machine with nil bootstrap config ref", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "nobootstrap-cluster").
			Machine(namespace, "nobootstrap-machine").ForCluster("nobootstrap-cluster").
			WithPhase("Running").
			WithInfraRef("AWSMachine", "nobootstrap-infra").
			Create().
			ToolCall("capi_get_machine").
			WithArg("namespace", namespace).
			WithArg("name", "nobootstrap-machine").
			AssertContent("nil_config_ref.golden").
			Execute()
	})

	t.Run("gets machine with empty infrastructure ref kind", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "noinfra-cluster").
			Machine(namespace, "noinfra-machine").ForCluster("noinfra-cluster").
			WithPhase("Provisioning").
			WithConfigRef("KubeadmConfig", "noinfra-bootstrap").
			Create().
			ToolCall("capi_get_machine").
			WithArg("namespace", namespace).
			WithArg("name", "noinfra-machine").
			AssertContent("empty_infra_ref.golden").
			Execute()
	})

	t.Run("gets machine with nil version", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "nover-cluster").
			Machine(namespace, "nover-machine").ForCluster("nover-cluster").
			WithPhase("Running").
			WithNodeRef("nover-node").
			Create().
			ToolCall("capi_get_machine").
			WithArg("namespace", namespace).
			WithArg("name", "nover-machine").
			AssertContent("nil_version.golden").
			Execute()
	})

	t.Run("gets machine with no conditions", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "nocond-cluster").
			Machine(namespace, "nocond-machine").ForCluster("nocond-cluster").
			WithPhase("Running").
			WithVersion("v1.29.0").
			WithInfraRef("AWSMachine", "nocond-infra").
			WithConfigRef("KubeadmConfig", "nocond-bootstrap").
			Create().
			ToolCall("capi_get_machine").
			WithArg("namespace", namespace).
			WithArg("name", "nocond-machine").
			AssertContent("no_conditions.golden").
			Execute()
	})

	t.Run("gets machine with condition without reason and message", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			CreateClusters(namespace, "sparse-cluster").
			Machine(namespace, "sparse-machine").ForCluster("sparse-cluster").
			WithPhase("Running").
			WithCondition("Ready").True().Done().
			WithCondition("InfrastructureReady").False().Done().
			Create().
			ToolCall("capi_get_machine").
			WithArg("namespace", namespace).
			WithArg("name", "sparse-machine").
			AssertContent("condition_no_reason_message.golden").
			Execute()
	})
}
