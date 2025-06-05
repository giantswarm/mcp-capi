package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// createListInfrastructureProvidersHandler creates a handler for listing available infrastructure providers
func createListInfrastructureProvidersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// In a real implementation, this would discover installed providers
		// For now, we'll return a static list of commonly available providers

		var content strings.Builder
		content.WriteString("Available Infrastructure Providers:\n\n")

		providers := []struct {
			Name        string
			APIVersion  string
			Description string
		}{
			{
				Name:        "AWS",
				APIVersion:  "infrastructure.cluster.x-k8s.io/v1beta2",
				Description: "Amazon Web Services infrastructure provider",
			},
			{
				Name:        "Azure",
				APIVersion:  "infrastructure.cluster.x-k8s.io/v1beta1",
				Description: "Microsoft Azure infrastructure provider",
			},
			{
				Name:        "GCP",
				APIVersion:  "infrastructure.cluster.x-k8s.io/v1beta1",
				Description: "Google Cloud Platform infrastructure provider",
			},
			{
				Name:        "vSphere",
				APIVersion:  "infrastructure.cluster.x-k8s.io/v1beta1",
				Description: "VMware vSphere infrastructure provider",
			},
		}

		for _, provider := range providers {
			content.WriteString(fmt.Sprintf("Provider: %s\n", provider.Name))
			content.WriteString(fmt.Sprintf("  API Version: %s\n", provider.APIVersion))
			content.WriteString(fmt.Sprintf("  Description: %s\n", provider.Description))
			content.WriteString("\n")
		}

		content.WriteString("Note: This list shows commonly available providers.\n")
		content.WriteString("To see actually installed providers in your cluster, check the deployed controllers.\n")

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

// createGetProviderConfigHandler creates a handler for getting provider configuration
func createGetProviderConfigHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		provider, ok := arguments["provider"].(string)
		if !ok || provider == "" {
			return nil, fmt.Errorf("provider argument is required (aws, azure, gcp, vsphere)")
		}

		var content strings.Builder
		content.WriteString(fmt.Sprintf("Configuration for %s Provider:\n\n", strings.ToUpper(provider)))

		switch strings.ToLower(provider) {
		case "aws":
			content.WriteString("AWS Provider Configuration:\n")
			content.WriteString("  Required Credentials:\n")
			content.WriteString("    - AWS_ACCESS_KEY_ID\n")
			content.WriteString("    - AWS_SECRET_ACCESS_KEY\n")
			content.WriteString("    - AWS_REGION\n")
			content.WriteString("  Optional:\n")
			content.WriteString("    - AWS_SESSION_TOKEN (for temporary credentials)\n")
			content.WriteString("    - AWS_PROFILE (to use a specific profile)\n\n")
			content.WriteString("  Common Resources:\n")
			content.WriteString("    - AWSCluster: Manages VPC, subnets, security groups\n")
			content.WriteString("    - AWSMachine: Individual EC2 instances\n")
			content.WriteString("    - AWSMachineTemplate: Template for creating machines\n")
			content.WriteString("    - AWSManagedControlPlane: EKS-based control plane\n")

		case "azure":
			content.WriteString("Azure Provider Configuration:\n")
			content.WriteString("  Required Credentials:\n")
			content.WriteString("    - AZURE_SUBSCRIPTION_ID\n")
			content.WriteString("    - AZURE_TENANT_ID\n")
			content.WriteString("    - AZURE_CLIENT_ID\n")
			content.WriteString("    - AZURE_CLIENT_SECRET\n")
			content.WriteString("  Optional:\n")
			content.WriteString("    - AZURE_ENVIRONMENT (AzurePublicCloud, AzureGermanCloud, etc.)\n\n")
			content.WriteString("  Common Resources:\n")
			content.WriteString("    - AzureCluster: Manages resource group, vnet, subnets\n")
			content.WriteString("    - AzureMachine: Individual VM instances\n")
			content.WriteString("    - AzureMachineTemplate: Template for creating machines\n")
			content.WriteString("    - AzureManagedControlPlane: AKS-based control plane\n")

		case "gcp":
			content.WriteString("GCP Provider Configuration:\n")
			content.WriteString("  Required Credentials:\n")
			content.WriteString("    - GOOGLE_APPLICATION_CREDENTIALS (path to service account key)\n")
			content.WriteString("    - GCP_PROJECT_ID\n")
			content.WriteString("    - GCP_REGION\n")
			content.WriteString("  Optional:\n")
			content.WriteString("    - GCP_NETWORK (custom network name)\n\n")
			content.WriteString("  Common Resources:\n")
			content.WriteString("    - GCPCluster: Manages VPC, subnets, firewall rules\n")
			content.WriteString("    - GCPMachine: Individual GCE instances\n")
			content.WriteString("    - GCPMachineTemplate: Template for creating machines\n")

		case "vsphere":
			content.WriteString("vSphere Provider Configuration:\n")
			content.WriteString("  Required Credentials:\n")
			content.WriteString("    - VSPHERE_SERVER\n")
			content.WriteString("    - VSPHERE_USERNAME\n")
			content.WriteString("    - VSPHERE_PASSWORD\n")
			content.WriteString("  Required Settings:\n")
			content.WriteString("    - VSPHERE_DATACENTER\n")
			content.WriteString("    - VSPHERE_DATASTORE\n")
			content.WriteString("    - VSPHERE_NETWORK\n")
			content.WriteString("    - VSPHERE_RESOURCE_POOL\n")
			content.WriteString("  Optional:\n")
			content.WriteString("    - VSPHERE_FOLDER\n")
			content.WriteString("    - VSPHERE_TEMPLATE (VM template to clone)\n\n")
			content.WriteString("  Common Resources:\n")
			content.WriteString("    - VSphereCluster: Manages cluster-level settings\n")
			content.WriteString("    - VSphereMachine: Individual VM instances\n")
			content.WriteString("    - VSphereMachineTemplate: Template for creating machines\n")

		default:
			return mcp.NewToolResultError(fmt.Sprintf("Unknown provider: %s. Supported providers: aws, azure, gcp, vsphere", provider)), nil
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
