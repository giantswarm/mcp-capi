package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiAWSGetCluster(t *testing.T) {
	t.Parallel()

	t.Run("should get AWS cluster details", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			ToolCall("capi_aws_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-aws-cluster").
			AssertContent("aws_cluster_details.golden").
			Execute()
	})

	t.Run("should error for non-AWS cluster", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-azure-cluster").WithProvider("azure").Create().
			ToolCall("capi_aws_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-azure-cluster").
			AssertError("not_aws_cluster.golden").
			Execute()
	})

	t.Run("should error when cluster not found", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_aws_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "nonexistent-cluster").
			AssertError("not_found.golden").
			Execute()
	})

	t.Run("should error when namespace is missing", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_aws_get_cluster").
			WithArg("name", "my-cluster").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("should error when name is missing", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_aws_get_cluster").
			WithArg("namespace", namespace).
			AssertError("missing_name.golden").
			Execute()
	})

	t.Run("should accept AWSManagedCluster kind", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-managed-cluster").WithCustomInfraRef("AWSManagedCluster", "my-managed-cluster").Create().
			ToolCall("capi_aws_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-managed-cluster").
			AssertContent("aws_managed_cluster_details.golden").
			Execute()
	})

	t.Run("should show cluster conditions", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").
			WithCondition("Ready").True().Reason("AllGood").Done().
			Create().
			ToolCall("capi_aws_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-aws-cluster").
			AssertContent("with_conditions.golden").
			Execute()
	})

	t.Run("should error for cluster with nil infrastructure ref", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "no-infra-cluster").Create().
			ToolCall("capi_aws_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "no-infra-cluster").
			AssertError("nil_infra_ref.golden").
			Execute()
	})

	t.Run("should show cluster network with pod and service CIDRs", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").
			WithNetwork([]string{"10.244.0.0/16"}, []string{"10.96.0.0/12"}).
			Create().
			ToolCall("capi_aws_get_cluster").
			WithArg("namespace", namespace).
			WithArg("name", "my-aws-cluster").
			AssertContent("with_cluster_network.golden").
			Execute()
	})
}
