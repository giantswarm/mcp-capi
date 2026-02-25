package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiGetProviderConfig(t *testing.T) {
	t.Parallel()

	t.Run("should get AWS provider config", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_get_provider_config").
			WithArg("provider", "aws").
			AssertContent("aws_config.golden").
			Execute()
	})

	t.Run("should get Azure provider config", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_get_provider_config").
			WithArg("provider", "azure").
			AssertContent("azure_config.golden").
			Execute()
	})

	t.Run("should get unknown provider config", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_get_provider_config").
			WithArg("provider", "unknown").
			AssertError("unknown_provider.golden").
			Execute()
	})
}
