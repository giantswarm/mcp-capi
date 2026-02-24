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
}
