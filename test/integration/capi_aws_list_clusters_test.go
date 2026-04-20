package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiAWSListClusters(t *testing.T) {
	t.Parallel()

	t.Run("should list AWS clusters", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			Cluster(namespace, "my-azure-cluster").WithProvider("azure").Create().
			ToolCall("capi_aws_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("aws_clusters.golden").
			Execute()
	})

	t.Run("should show no AWS clusters when none exist", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_aws_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("no_aws_clusters.golden").
			Execute()
	})

	t.Run("should list AWSManagedCluster clusters", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-managed-cluster").
			WithCustomInfraRef("AWSManagedCluster", "my-managed-cluster").Create().
			ToolCall("capi_aws_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("aws_managed_cluster.golden").
			Execute()
	})

	t.Run("should filter by namespace", func(t *testing.T) {
		t.Parallel()
		ns1 := testNamespace
		ns2 := "other-clusters"

		harness.New(t).
			CreateNamespace(ns1).
			CreateNamespace(ns2).
			Cluster(ns1, "aws-cluster-1").WithProvider("aws").Create().
			Cluster(ns2, "aws-cluster-2").WithProvider("aws").Create().
			ToolCall("capi_aws_list_clusters").
			WithArg("namespace", ns1).
			AssertContent("namespace_filter.golden").
			Execute()
	})

	t.Run("should skip clusters with nil infrastructure ref", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "no-infra-cluster").Create().
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			ToolCall("capi_aws_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("nil_infra_ref.golden").
			Execute()
	})

	t.Run("should list multiple AWS clusters with different phases", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "cluster-pending").WithProvider("aws").WithPhase("Pending").Create().
			Cluster(namespace, "cluster-provisioned").WithProvider("aws").WithPhase("Provisioned").Create().
			ToolCall("capi_aws_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("multiple_phases.golden").
			Execute()
	})
}
