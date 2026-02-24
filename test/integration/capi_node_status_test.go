package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiNodeStatus(t *testing.T) {
	t.Parallel()

	t.Run("returns error when node_name not found", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_node_status").
			WithArg("node_name", "does-not-exist").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("returns error when neither node_name nor namespace and machine_name provided", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_node_status").
			AssertError("missing_args.golden").
			Execute()
	})

	t.Run("returns error when only namespace provided without machine_name", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_node_status").
			WithArg("namespace", "test-clusters").
			AssertError("missing_machine_name.golden").
			Execute()
	})

	t.Run("shows ready node status", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			Node("ready-node").
			WithCondition("Ready").Status("True").Reason("KubeletReady").Message("kubelet is posting ready status").Done().
			WithAddress("InternalIP", "10.0.0.1").
			WithAddress("Hostname", "ready-node").
			WithTaint("node.kubernetes.io/not-ready", "", "NoSchedule").
			Create().
			ToolCall("capi_node_status").
			WithArg("node_name", "ready-node").
			AssertContentNormalized("ready_node.golden", harness.NormalizeUID, harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("shows not-ready node status", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			Node("not-ready-node").
			WithCondition("Ready").Status("False").Reason("KubeletNotReady").Message("container runtime network not ready").Done().
			WithCondition("MemoryPressure").Status("False").Reason("KubeletHasSufficientMemory").Message("kubelet has sufficient memory available").Done().
			WithAddress("InternalIP", "10.0.0.5").
			WithAddress("Hostname", "not-ready-node").
			WithTaint("node.kubernetes.io/not-ready", "", "NoSchedule").
			Create().
			ToolCall("capi_node_status").
			WithArg("node_name", "not-ready-node").
			AssertContentNormalized("not_ready_node.golden", harness.NormalizeUID, harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("shows node with multiple conditions", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			Node("multi-cond-node").
			WithCondition("Ready").Status("True").Reason("KubeletReady").Message("kubelet is posting ready status").Done().
			WithCondition("MemoryPressure").Status("False").Reason("KubeletHasSufficientMemory").Message("kubelet has sufficient memory available").Done().
			WithCondition("DiskPressure").Status("False").Reason("KubeletHasNoDiskPressure").Message("kubelet has no disk pressure").Done().
			WithCondition("PIDPressure").Status("False").Reason("KubeletHasSufficientPID").Message("kubelet has sufficient PID available").Done().
			WithAddress("InternalIP", "10.0.0.2").
			WithAddress("Hostname", "multi-cond-node").
			WithTaint("node.kubernetes.io/not-ready", "", "NoSchedule").
			Create().
			ToolCall("capi_node_status").
			WithArg("node_name", "multi-cond-node").
			AssertContentNormalized("multiple_conditions.golden", harness.NormalizeUID, harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("shows cordoned node status", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			Node("cordoned-node").
			WithUnschedulable(true).
			WithCondition("Ready").Status("True").Reason("KubeletReady").Message("kubelet is posting ready status").Done().
			WithAddress("InternalIP", "10.0.0.3").
			WithAddress("Hostname", "cordoned-node").
			WithTaint("node.kubernetes.io/unschedulable", "", "NoSchedule").
			WithTaint("node.kubernetes.io/not-ready", "", "NoSchedule").
			Create().
			ToolCall("capi_node_status").
			WithArg("node_name", "cordoned-node").
			AssertContentNormalized("cordoned_node.golden", harness.NormalizeUID, harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("shows node with provider ID", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			Node("provider-node").
			WithProviderID("aws:///us-east-1a/i-1234567890abcdef0").
			WithCondition("Ready").Status("True").Reason("KubeletReady").Message("kubelet is posting ready status").Done().
			WithAddress("InternalIP", "10.0.0.4").
			WithAddress("ExternalIP", "54.123.45.67").
			WithAddress("Hostname", "provider-node").
			WithTaint("node.kubernetes.io/not-ready", "", "NoSchedule").
			Create().
			ToolCall("capi_node_status").
			WithArg("node_name", "provider-node").
			AssertContentNormalized("provider_id.golden", harness.NormalizeUID, harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("retrieves node status via machine_name lookup", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Node("lookup-cluster-machine-0-node").
			WithCondition("Ready").Status("True").Reason("KubeletReady").Message("kubelet is posting ready status").Done().
			WithAddress("InternalIP", "10.0.1.1").
			WithAddress("Hostname", "lookup-cluster-machine-0-node").
			WithTaint("node.kubernetes.io/not-ready", "", "NoSchedule").
			Create().
			Cluster(namespace, "lookup-cluster").WithProvider("aws").WithMachines(1, 1).Create().
			ToolCall("capi_node_status").
			WithArg("namespace", namespace).
			WithArg("machine_name", "lookup-cluster-machine-0").
			AssertContentNormalized("machine_lookup.golden", harness.NormalizeUID, harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("shows node with multiple address types", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			Node("multi-addr-node").
			WithCondition("Ready").Status("True").Reason("KubeletReady").Message("kubelet is posting ready status").Done().
			WithAddress("InternalIP", "10.0.0.10").
			WithAddress("ExternalIP", "203.0.113.50").
			WithAddress("Hostname", "multi-addr-node").
			WithTaint("node.kubernetes.io/not-ready", "", "NoSchedule").
			Create().
			ToolCall("capi_node_status").
			WithArg("node_name", "multi-addr-node").
			AssertContentNormalized("multiple_addresses.golden", harness.NormalizeUID, harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("shows node with taints", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			Node("tainted-node").
			WithCondition("Ready").Status("True").Reason("KubeletReady").Message("kubelet is posting ready status").Done().
			WithAddress("InternalIP", "10.0.0.11").
			WithAddress("Hostname", "tainted-node").
			WithTaint("dedicated", "gpu", "NoSchedule").
			WithTaint("critical", "", "NoExecute").
			WithTaint("prefer-no", "test", "PreferNoSchedule").
			Create().
			ToolCall("capi_node_status").
			WithArg("node_name", "tainted-node").
			AssertContentNormalized("taints.golden", harness.NormalizeUID, harness.NormalizeTimestamp).
			Execute()
	})

	t.Run("returns error when machine has no associated node", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "noref-cluster").WithProvider("aws").WithMachines(1, 0).Create().
			ToolCall("capi_node_status").
			WithArg("namespace", namespace).
			WithArg("machine_name", "noref-cluster-machine-0").
			AssertError("machine_no_noderef.golden").
			Execute()
	})
}
