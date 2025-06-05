package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// AWS Provider Tools

// createAWSListClustersHandler lists AWS clusters
func createAWSListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, _ := arguments["namespace"].(string)

		// List all clusters
		clusters, err := serverCtx.capiClient.ListClusters(ctx, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		var content strings.Builder
		content.WriteString("AWS Clusters:\n\n")

		awsClusterCount := 0
		for _, cluster := range clusters.Items {
			// Check if this is an AWS cluster
			if cluster.Spec.InfrastructureRef != nil &&
				(cluster.Spec.InfrastructureRef.Kind == "AWSCluster" ||
					cluster.Spec.InfrastructureRef.Kind == "AWSManagedCluster") {
				awsClusterCount++

				content.WriteString(fmt.Sprintf("Cluster: %s/%s\n", cluster.Namespace, cluster.Name))
				content.WriteString(fmt.Sprintf("  Infrastructure: %s\n", cluster.Spec.InfrastructureRef.Kind))
				content.WriteString(fmt.Sprintf("  Phase: %s\n", cluster.Status.Phase))
				content.WriteString(fmt.Sprintf("  Ready: %v\n", cluster.Status.InfrastructureReady))

				// Try to get provider information
				provider, _ := serverCtx.capiClient.GetProviderForCluster(ctx, cluster.Namespace, cluster.Name)
				if provider == capi.ProviderAWS {
					content.WriteString("  Provider: AWS (confirmed)\n")
				}

				content.WriteString("\n")
			}
		}

		if awsClusterCount == 0 {
			content.WriteString("No AWS clusters found.\n")
		} else {
			content.WriteString(fmt.Sprintf("Total AWS clusters: %d\n", awsClusterCount))
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: content.String(),
				},
			},
		}, nil
	}
}

// createAWSGetClusterHandler gets details of an AWS cluster
func createAWSGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			return nil, fmt.Errorf("name argument is required")
		}

		// Get the cluster
		cluster, err := serverCtx.capiClient.GetCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster: %w", err)
		}

		// Verify it's an AWS cluster
		if cluster.Spec.InfrastructureRef == nil ||
			(cluster.Spec.InfrastructureRef.Kind != "AWSCluster" &&
				cluster.Spec.InfrastructureRef.Kind != "AWSManagedCluster") {
			return mcp.NewToolResultError(fmt.Sprintf("Cluster %s/%s is not an AWS cluster", namespace, name)), nil
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("AWS Cluster: %s/%s\n\n", namespace, name))

		// Basic cluster info
		content.WriteString("Cluster Information:\n")
		content.WriteString(fmt.Sprintf("  Phase: %s\n", cluster.Status.Phase))
		content.WriteString(fmt.Sprintf("  Infrastructure Ready: %v\n", cluster.Status.InfrastructureReady))
		content.WriteString(fmt.Sprintf("  Control Plane Ready: %v\n", cluster.Status.ControlPlaneReady))

		// Infrastructure reference
		content.WriteString("\nInfrastructure:\n")
		content.WriteString(fmt.Sprintf("  Kind: %s\n", cluster.Spec.InfrastructureRef.Kind))
		content.WriteString(fmt.Sprintf("  Name: %s\n", cluster.Spec.InfrastructureRef.Name))
		content.WriteString(fmt.Sprintf("  API Version: %s\n", cluster.Spec.InfrastructureRef.APIVersion))

		// Network configuration
		if cluster.Spec.ClusterNetwork != nil {
			content.WriteString("\nNetwork Configuration:\n")
			if cluster.Spec.ClusterNetwork.Pods != nil && len(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks) > 0 {
				content.WriteString(fmt.Sprintf("  Pod CIDR: %s\n", strings.Join(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks, ", ")))
			}
			if cluster.Spec.ClusterNetwork.Services != nil && len(cluster.Spec.ClusterNetwork.Services.CIDRBlocks) > 0 {
				content.WriteString(fmt.Sprintf("  Service CIDR: %s\n", strings.Join(cluster.Spec.ClusterNetwork.Services.CIDRBlocks, ", ")))
			}
		}

		// Conditions
		if len(cluster.Status.Conditions) > 0 {
			content.WriteString("\nConditions:\n")
			for _, condition := range cluster.Status.Conditions {
				content.WriteString(fmt.Sprintf("  - %s: %s", condition.Type, condition.Status))
				if condition.Reason != "" {
					content.WriteString(fmt.Sprintf(" (%s)", condition.Reason))
				}
				content.WriteString("\n")
			}
		}

		content.WriteString("\nNote: For detailed AWS infrastructure information (VPC, subnets, etc.),\n")
		content.WriteString("you would need to query the AWSCluster resource directly.\n")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: content.String(),
				},
			},
		}, nil
	}
}

// createAWSGetMachineTemplateHandler gets AWS machine templates
func createAWSGetMachineTemplateHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, fmt.Errorf("namespace argument is required")
		}
		name, _ := arguments["name"].(string)

		var content strings.Builder

		if name != "" {
			// Get specific machine template
			content.WriteString(fmt.Sprintf("AWS Machine Template: %s/%s\n\n", namespace, name))
			content.WriteString("Note: Direct access to AWSMachineTemplate requires the AWS provider CRDs.\n")
			content.WriteString("In a full implementation, this would show:\n")
			content.WriteString("  - Instance type\n")
			content.WriteString("  - AMI ID\n")
			content.WriteString("  - Security groups\n")
			content.WriteString("  - SSH key name\n")
			content.WriteString("  - IAM instance profile\n")
			content.WriteString("  - User data configuration\n")
		} else {
			// List all machine templates
			content.WriteString(fmt.Sprintf("AWS Machine Templates in namespace %s:\n\n", namespace))

			// In a real implementation, we would list AWSMachineTemplate resources
			// For now, we'll check for machine deployments and their templates
			mds, err := serverCtx.capiClient.ListMachineDeployments(ctx, namespace, "")
			if err != nil {
				return nil, fmt.Errorf("failed to list machine deployments: %w", err)
			}

			awsTemplateCount := 0
			for _, md := range mds.Items {
				if md.Spec.Template.Spec.InfrastructureRef.Kind == "AWSMachineTemplate" {
					awsTemplateCount++
					content.WriteString(fmt.Sprintf("Template: %s (used by MachineDeployment: %s)\n",
						md.Spec.Template.Spec.InfrastructureRef.Name, md.Name))
				}
			}

			if awsTemplateCount == 0 {
				content.WriteString("No AWS machine templates found in use.\n")
			}
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: content.String(),
				},
			},
		}, nil
	}
}

// Placeholder handlers for provider-specific operations

// createAWSCreateClusterHandler creates AWS-specific cluster configuration
func createAWSCreateClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
					Type: "text",
					Text: content.String(),
				},
			},
		}, nil
	}
}

// createAWSUpdateVPCHandler updates AWS VPC configuration
func createAWSUpdateVPCHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
					Type: "text",
					Text: content.String(),
				},
			},
		}, nil
	}
}

// createAWSManageSecurityGroupsHandler manages AWS security groups
func createAWSManageSecurityGroupsHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
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
					Type: "text",
					Text: content.String(),
				},
			},
		}, nil
	}
}
