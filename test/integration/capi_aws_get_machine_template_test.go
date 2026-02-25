package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiAWSGetMachineTemplate(t *testing.T) {
	t.Parallel()

	t.Run("should list machine templates", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			MachineDeployment(namespace, "my-aws-md").
			ForCluster("my-aws-cluster").
			WithInfraRef("AWSMachineTemplate", "my-aws-template").
			Create().
			ToolCall("capi_aws_get_machine_template").
			WithArg("namespace", namespace).
			AssertContent("machine_templates.golden").
			Execute()
	})

	t.Run("should show no templates when none exist", func(t *testing.T) {
		t.Parallel()
		namespace := "test-clusters"

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_aws_get_machine_template").
			WithArg("namespace", namespace).
			AssertContent("no_templates.golden").
			Execute()
	})
}
