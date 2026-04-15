package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
)

// AWS Provider Tools

// CreateAWSListClustersHandler lists AWS clusters
func CreateAWSListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return buildListClustersHandler(serverCtx, providerListConfig{
		header:        "AWS Clusters:\n\n",
		infraKinds:    []string{"AWSCluster", "AWSManagedCluster"},
		provider:      capi.ProviderAWS,
		providerLabel: "AWS",
		noneMsg:       "No AWS clusters found.\n",
		totalFmt:      "Total AWS clusters: %d\n",
	})
}

// CreateAWSGetClusterHandler gets details of an AWS cluster
func CreateAWSGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, errors.New("namespace argument is required")
		}
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			return nil, errors.New("name argument is required")
		}

		cluster, err := serverCtx.CAPIClient.GetCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster: %w", err)
		}

		if cluster.Spec.InfrastructureRef == nil ||
			(cluster.Spec.InfrastructureRef.Kind != "AWSCluster" &&
				cluster.Spec.InfrastructureRef.Kind != "AWSManagedCluster") {
			return mcp.NewToolResultError(fmt.Sprintf("Cluster %s/%s is not an AWS cluster", namespace, name)), nil
		}

		var content strings.Builder
		fmt.Fprintf(&content, "AWS Cluster: %s/%s\n\n", namespace, name)
		writeAWSClusterBasicInfo(&content, cluster)
		writeAWSNetworkConfig(&content, cluster)
		writeAWSConditions(&content, cluster)
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

// CreateAWSGetMachineTemplateHandler gets AWS machine templates
func CreateAWSGetMachineTemplateHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, errors.New("namespace argument is required")
		}
		name, _ := arguments["name"].(string)

		var content strings.Builder

		if name != "" {
			// Get specific machine template
			fmt.Fprintf(&content, "AWS Machine Template: %s/%s\n\n", namespace, name)
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
			fmt.Fprintf(&content, "AWS Machine Templates in namespace %s:\n\n", namespace)

			// In a real implementation, we would list AWSMachineTemplate resources
			// For now, we'll check for machine deployments and their templates
			mds, err := serverCtx.CAPIClient.ListMachineDeployments(ctx, namespace, "")
			if err != nil {
				return nil, fmt.Errorf("failed to list machine deployments: %w", err)
			}

			awsTemplateCount := 0
			for i := range mds.Items {
				md := &mds.Items[i]
				if md.Spec.Template.Spec.InfrastructureRef.Kind == "AWSMachineTemplate" {
					awsTemplateCount++
					fmt.Fprintf(&content, "Template: %s (used by MachineDeployment: %s)\n",
						md.Spec.Template.Spec.InfrastructureRef.Name, md.Name)
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

// CreateAWSCreateClusterHandler creates AWS-specific cluster configuration (placeholder)
func CreateAWSCreateClusterHandler(_ *ServerContext) server.ToolHandlerFunc {
	return buildPlaceholderHandler("AWS Cluster Creation (Placeholder)\n\n" +
		"This tool would create AWS-specific cluster resources including:\n" +
		"- AWSCluster resource with VPC, subnet, and security group configuration\n" +
		"- IAM roles and policies for cluster components\n" +
		"- S3 buckets for OIDC discovery (if using IRSA)\n" +
		"- Load balancers for API server access\n\n" +
		"Required parameters would include:\n" +
		"- Region\n" +
		"- VPC CIDR\n" +
		"- Availability zones\n" +
		"- Instance types\n" +
		"- SSH key name\n")
}

// CreateAWSUpdateVPCHandler updates AWS VPC configuration (placeholder)
func CreateAWSUpdateVPCHandler(_ *ServerContext) server.ToolHandlerFunc {
	return buildPlaceholderHandler("AWS VPC Update (Placeholder)\n\n" +
		"This tool would update AWS VPC configuration for CAPI clusters.\n" +
		"Operations would include:\n" +
		"- Adding/removing subnets\n" +
		"- Updating route tables\n" +
		"- Modifying security group rules\n" +
		"- Configuring VPC peering\n\n" +
		"Note: VPC updates must be done carefully to avoid disrupting running clusters.\n")
}

// CreateAWSManageSecurityGroupsHandler manages AWS security groups (placeholder)
func CreateAWSManageSecurityGroupsHandler(_ *ServerContext) server.ToolHandlerFunc {
	return buildPlaceholderHandler("AWS Security Groups Management (Placeholder)\n\n" +
		"This tool would manage security groups for CAPI AWS clusters.\n" +
		"Operations would include:\n" +
		"- Adding/removing ingress rules\n" +
		"- Adding/removing egress rules\n" +
		"- Creating new security groups\n" +
		"- Attaching security groups to instances\n\n" +
		"Common use cases:\n" +
		"- Opening ports for additional services\n" +
		"- Restricting access to specific IP ranges\n" +
		"- Enabling inter-cluster communication\n")
}

// writeAWSClusterBasicInfo writes basic cluster status and infrastructure info to the builder.
func writeAWSClusterBasicInfo(w *strings.Builder, cluster *clusterv1.Cluster) {
	w.WriteString("Cluster Information:\n")
	fmt.Fprintf(w, "  Phase: %s\n", cluster.Status.Phase)
	fmt.Fprintf(w, "  Infrastructure Ready: %v\n", cluster.Status.InfrastructureReady)
	fmt.Fprintf(w, "  Control Plane Ready: %v\n", cluster.Status.ControlPlaneReady)
	w.WriteString("\nInfrastructure:\n")
	fmt.Fprintf(w, "  Kind: %s\n", cluster.Spec.InfrastructureRef.Kind)
	fmt.Fprintf(w, "  Name: %s\n", cluster.Spec.InfrastructureRef.Name)
	fmt.Fprintf(w, "  API Version: %s\n", cluster.Spec.InfrastructureRef.APIVersion)
}

// writeAWSNetworkConfig writes network configuration details to the builder if present.
func writeAWSNetworkConfig(w *strings.Builder, cluster *clusterv1.Cluster) {
	if cluster.Spec.ClusterNetwork == nil {
		return
	}
	w.WriteString("\nNetwork Configuration:\n")
	if cluster.Spec.ClusterNetwork.Pods != nil && len(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks) > 0 {
		fmt.Fprintf(w, "  Pod CIDR: %s\n", strings.Join(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks, ", "))
	}
	if cluster.Spec.ClusterNetwork.Services != nil && len(cluster.Spec.ClusterNetwork.Services.CIDRBlocks) > 0 {
		fmt.Fprintf(w, "  Service CIDR: %s\n", strings.Join(cluster.Spec.ClusterNetwork.Services.CIDRBlocks, ", "))
	}
}

// writeAWSConditions writes cluster conditions to the builder if any exist.
func writeAWSConditions(w *strings.Builder, cluster *clusterv1.Cluster) {
	if len(cluster.Status.Conditions) == 0 {
		return
	}
	w.WriteString("\nConditions:\n")
	for _, condition := range cluster.Status.Conditions {
		fmt.Fprintf(w, "  - %s: %s", condition.Type, condition.Status)
		if condition.Reason != "" {
			fmt.Fprintf(w, " (%s)", condition.Reason)
		}
		w.WriteString("\n")
	}
}
