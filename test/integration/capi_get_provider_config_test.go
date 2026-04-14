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

	t.Run("should get GCP provider config", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_get_provider_config").
			WithArg("provider", "gcp").
			AssertContent("gcp_config.golden").
			Execute()
	})

	t.Run("should get vSphere provider config", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_get_provider_config").
			WithArg("provider", "vsphere").
			AssertContent("vsphere_config.golden").
			Execute()
	})

	t.Run("should error when provider argument is missing", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_get_provider_config").
			AssertError("missing_provider.golden").
			Execute()
	})

	t.Run("should normalize provider name to lowercase", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_get_provider_config").
			WithArg("provider", "AWS").
			AssertContent("aws_config.golden").
			Execute()
	})
}
