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

	// Note: Content-based golden file tests for capi_node_status are not included
	// because the handler output contains non-deterministic fields (UID, CreationTimestamp)
	// that change on every test run. The error-path tests above verify argument validation
	// and node lookup behavior. The Node builder infrastructure in the harness is available
	// for future tests if partial-matching support is added.
}
