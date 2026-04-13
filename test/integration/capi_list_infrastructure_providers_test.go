package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiListInfrastructureProviders(t *testing.T) {
	t.Parallel()

	t.Run("should list all infrastructure providers", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_list_infrastructure_providers").
			AssertContent("list_all.golden").
			Execute()
	})
}
