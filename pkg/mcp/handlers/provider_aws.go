package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AWS Provider Tools

const awsClusterNote = "Provider-specific infrastructure details (VPC, subnets, security groups) live on the AWSCluster resource, which this tool does not read."

// CreateAWSListClustersHandler lists AWS clusters as {items: [...]}.
func CreateAWSListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return createProviderListClustersHandler(serverCtx, "AWSCluster", "AWSManagedCluster")
}

// CreateAWSGetClusterHandler gets details of an AWS cluster.
func CreateAWSGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return createProviderGetClusterHandler(serverCtx, "AWS", awsClusterNote, "AWSCluster", "AWSManagedCluster")
}

// awsMachineTemplateResult is the result for one named AWS machine template.
type awsMachineTemplateResult struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Note      string `json:"note"`
}

// awsMachineTemplateSummary is one entry of the AWS machine template list.
type awsMachineTemplateSummary struct {
	Name   string    `json:"name"`
	UsedBy ObjectRef `json:"usedBy"`
}

// CreateAWSGetMachineTemplateHandler gets or lists AWS machine templates.
func CreateAWSGetMachineTemplateHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		capiClient, err := serverCtx.Client(ctx)
		if err != nil {
			return nil, err
		}
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		name, _ := arguments["name"].(string)

		if name != "" {
			return jsonResult(awsMachineTemplateResult{
				Namespace: namespace,
				Name:      name,
				Note:      "Reading an AWSMachineTemplate (instance type, AMI, security groups, SSH key, IAM profile, user data) requires the AWS provider CRDs and is not implemented.",
			})
		}

		// Templates are discovered through the MachineDeployments that reference them.
		mds, err := capiClient.ListMachineDeployments(ctx, namespace, "")
		if err != nil {
			return nil, fmt.Errorf("failed to list machine deployments: %w", err)
		}

		items := make([]awsMachineTemplateSummary, 0)
		for _, md := range mds.Items {
			ref := md.Spec.Template.Spec.InfrastructureRef
			if ref.Kind != "AWSMachineTemplate" {
				continue
			}
			items = append(items, awsMachineTemplateSummary{
				Name:   ref.Name,
				UsedBy: ObjectRef{Kind: "MachineDeployment", Name: md.Name},
			})
		}

		return listResult(items)
	}
}

// Placeholder handlers for provider-specific operations. They are not
// registered as tools.

// CreateAWSCreateClusterHandler creates AWS-specific cluster configuration
func CreateAWSCreateClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var content strings.Builder
		content.WriteString("AWS Cluster Creation (Placeholder)\n\n")
		content.WriteString("This tool would create AWS-specific cluster resources including:\n")
		content.WriteString("- AWSCluster resource with VPC, subnet, and security group configuration\n")
		content.WriteString("- IAM roles and policies for cluster components\n")
		content.WriteString("- S3 buckets for OIDC discovery (if using IRSA)\n")
		content.WriteString("- Load balancers for API server access\n\n")
		content.WriteString("Required parameters would include:\n")
		content.WriteString("- Region\n")
		content.WriteString("- VPC CIDR\n")
		content.WriteString("- Availability zones\n")
		content.WriteString("- Instance types\n")
		content.WriteString("- SSH key name\n")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: textContentType,
					Text: content.String(),
				},
			},
		}, nil
	}
}

// CreateAWSUpdateVPCHandler updates AWS VPC configuration
func CreateAWSUpdateVPCHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var content strings.Builder
		content.WriteString("AWS VPC Update (Placeholder)\n\n")
		content.WriteString("This tool would update AWS VPC configuration for CAPI clusters.\n")
		content.WriteString("Operations would include:\n")
		content.WriteString("- Adding/removing subnets\n")
		content.WriteString("- Updating route tables\n")
		content.WriteString("- Modifying security group rules\n")
		content.WriteString("- Configuring VPC peering\n\n")
		content.WriteString("Note: VPC updates must be done carefully to avoid disrupting running clusters.\n")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: textContentType,
					Text: content.String(),
				},
			},
		}, nil
	}
}

// CreateAWSManageSecurityGroupsHandler manages AWS security groups
func CreateAWSManageSecurityGroupsHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var content strings.Builder
		content.WriteString("AWS Security Groups Management (Placeholder)\n\n")
		content.WriteString("This tool would manage security groups for CAPI AWS clusters.\n")
		content.WriteString("Operations would include:\n")
		content.WriteString("- Adding/removing ingress rules\n")
		content.WriteString("- Adding/removing egress rules\n")
		content.WriteString("- Creating new security groups\n")
		content.WriteString("- Attaching security groups to instances\n\n")
		content.WriteString("Common use cases:\n")
		content.WriteString("- Opening ports for additional services\n")
		content.WriteString("- Restricting access to specific IP ranges\n")
		content.WriteString("- Enabling inter-cluster communication\n")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: textContentType,
					Text: content.String(),
				},
			},
		}, nil
	}
}
