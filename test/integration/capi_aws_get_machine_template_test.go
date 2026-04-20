package integration_test

import (
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiAWSGetMachineTemplate(t *testing.T) {
	t.Parallel()

	t.Run("should list machine templates", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

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
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_aws_get_machine_template").
			WithArg("namespace", namespace).
			AssertContent("no_templates.golden").
			Execute()
	})

	t.Run("should get specific template by name", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			ToolCall("capi_aws_get_machine_template").
			WithArg("namespace", namespace).
			WithArg("name", "my-aws-template").
			AssertContent("specific_template.golden").
			Execute()
	})

	t.Run("should error when namespace is missing", func(t *testing.T) {
		t.Parallel()

		harness.New(t).
			ToolCall("capi_aws_get_machine_template").
			AssertError("missing_namespace.golden").
			Execute()
	})

	t.Run("should only list AWSMachineTemplate templates", func(t *testing.T) {
		t.Parallel()
		namespace := testNamespace

		harness.New(t).
			CreateNamespace(namespace).
			Cluster(namespace, "my-aws-cluster").WithProvider("aws").Create().
			MachineDeployment(namespace, "aws-md").
			ForCluster("my-aws-cluster").
			WithInfraRef("AWSMachineTemplate", "aws-template").
			Create().
			MachineDeployment(namespace, "azure-md").
			ForCluster("my-aws-cluster").
			WithInfraRef("AzureMachineTemplate", "azure-template").
			Create().
			ToolCall("capi_aws_get_machine_template").
			WithArg("namespace", namespace).
			AssertContent("mixed_template_kinds.golden").
			Execute()
	})
}
