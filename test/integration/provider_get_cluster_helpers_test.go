package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

// providerGetClusterConfig holds the provider-specific parameters for the get cluster test suite.
type providerGetClusterConfig struct {
	provider      string
	toolName      string
	managedKind   string
	clusterPrefix string
}

// runProviderGetClusterTests runs the standard set of get-cluster tests for a provider.
func runProviderGetClusterTests(t *testing.T, cfg providerGetClusterConfig) {
	t.Helper()
	t.Parallel()

	t.Run("should get "+cfg.provider+" cluster details", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace
		clusterName := "my-" + cfg.provider + "-cluster"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, clusterName).WithProvider(cfg.provider).Create().
			ToolCall(cfg.toolName).
			WithArg("namespace", namespace).
			WithArg("name", clusterName).
			AssertContent(cfg.provider + "_cluster_details.golden").
			Execute()
	})

	t.Run("should error for non-"+cfg.provider+" cluster", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			ToolCall(cfg.toolName).
			WithArg("namespace", namespace).
			WithArg("name", "my-aws-cluster").
			AssertError("not_" + cfg.provider + "_cluster.golden").
			Execute()
	})

	t.Run("should error when cluster not found", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall(cfg.toolName).
			WithArg("namespace", namespace).
			WithArg("name", "nonexistent-cluster").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("should error when namespace is missing", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall(cfg.toolName).
			WithArg("name", "my-cluster").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("should error when name is missing", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall(cfg.toolName).
			WithArg("namespace", namespace).
			AssertError("missing_name.golden").
			Execute()
	})

	t.Run("should accept "+cfg.managedKind+" kind", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-managed-cluster").
			WithCustomInfraRef(cfg.managedKind, "my-managed-cluster").Create().
			ToolCall(cfg.toolName).
			WithArg("namespace", namespace).
			WithArg("name", "my-managed-cluster").
			AssertContent(cfg.provider + "_managed_cluster_details.golden").
			Execute()
	})

	t.Run("should error for cluster with nil infrastructure ref", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "no-infra-cluster").Create().
			ToolCall(cfg.toolName).
			WithArg("namespace", namespace).
			WithArg("name", "no-infra-cluster").
			AssertError("nil_infra_ref.golden").
			Execute()
	})
}
