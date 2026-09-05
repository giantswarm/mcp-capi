package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-capi/pkg/capi"
)

// infrastructureProvider describes one infrastructure provider.
type infrastructureProvider struct {
	Name        string `json:"name"`
	APIVersion  string `json:"apiVersion"`
	Description string `json:"description"`
}

// infrastructureProvidersResult is the result of capi_list_infrastructure_providers.
type infrastructureProvidersResult struct {
	Items []infrastructureProvider `json:"items"`
	Note  string                   `json:"note"`
}

// CreateListInfrastructureProvidersHandler lists the commonly available
// infrastructure providers. The list is static; it does not inspect the cluster.
func CreateListInfrastructureProvidersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return jsonResult(infrastructureProvidersResult{
			Items: []infrastructureProvider{
				{Name: "AWS", APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2", Description: "Amazon Web Services infrastructure provider"},
				{Name: "Azure", APIVersion: infraAPIV1Beta1, Description: "Microsoft Azure infrastructure provider"},
				{Name: "GCP", APIVersion: infraAPIV1Beta1, Description: "Google Cloud Platform infrastructure provider"},
				{Name: "vSphere", APIVersion: infraAPIV1Beta1, Description: "VMware vSphere infrastructure provider"},
			},
			Note: "Commonly available providers; the controllers deployed in the management cluster determine which are actually installed.",
		})
	}
}

// providerResource describes one CRD kind a provider brings.
type providerResource struct {
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

// providerConfig is the result of capi_get_provider_config.
type providerConfig struct {
	Provider            string             `json:"provider"`
	RequiredCredentials []string           `json:"requiredCredentials"`
	RequiredSettings    []string           `json:"requiredSettings,omitempty"`
	OptionalSettings    []string           `json:"optionalSettings,omitempty"`
	Resources           []providerResource `json:"resources"`
}

const descMachineTemplate = "Template for creating machines"

var providerConfigs = map[capi.Provider]providerConfig{
	capi.ProviderAWS: {
		Provider:            string(capi.ProviderAWS),
		RequiredCredentials: []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_REGION"},
		OptionalSettings:    []string{"AWS_SESSION_TOKEN (temporary credentials)", "AWS_PROFILE (use a specific profile)"},
		Resources: []providerResource{
			{Kind: "AWSCluster", Description: "Manages VPC, subnets, security groups"},
			{Kind: "AWSMachine", Description: "Individual EC2 instances"},
			{Kind: "AWSMachineTemplate", Description: descMachineTemplate},
			{Kind: "AWSManagedControlPlane", Description: "EKS-based control plane"},
		},
	},
	capi.ProviderAzure: {
		Provider:            string(capi.ProviderAzure),
		RequiredCredentials: []string{"AZURE_SUBSCRIPTION_ID", "AZURE_TENANT_ID", "AZURE_CLIENT_ID", "AZURE_CLIENT_SECRET"},
		OptionalSettings:    []string{"AZURE_ENVIRONMENT (AzurePublicCloud, AzureGermanCloud, ...)"},
		Resources: []providerResource{
			{Kind: "AzureCluster", Description: "Manages resource group, vnet, subnets"},
			{Kind: "AzureMachine", Description: "Individual VM instances"},
			{Kind: "AzureMachineTemplate", Description: descMachineTemplate},
			{Kind: "AzureManagedControlPlane", Description: "AKS-based control plane"},
		},
	},
	capi.ProviderGCP: {
		Provider:            string(capi.ProviderGCP),
		RequiredCredentials: []string{"GOOGLE_APPLICATION_CREDENTIALS (path to service account key)", "GCP_PROJECT_ID", "GCP_REGION"},
		OptionalSettings:    []string{"GCP_NETWORK (custom network name)"},
		Resources: []providerResource{
			{Kind: "GCPCluster", Description: "Manages VPC, subnets, firewall rules"},
			{Kind: "GCPMachine", Description: "Individual GCE instances"},
			{Kind: "GCPMachineTemplate", Description: descMachineTemplate},
		},
	},
	capi.ProviderVSphere: {
		Provider:            string(capi.ProviderVSphere),
		RequiredCredentials: []string{"VSPHERE_SERVER", "VSPHERE_USERNAME", "VSPHERE_PASSWORD"},
		RequiredSettings:    []string{"VSPHERE_DATACENTER", "VSPHERE_DATASTORE", "VSPHERE_NETWORK", "VSPHERE_RESOURCE_POOL"},
		OptionalSettings:    []string{"VSPHERE_FOLDER", "VSPHERE_TEMPLATE (VM template to clone)"},
		Resources: []providerResource{
			{Kind: "VSphereCluster", Description: "Manages cluster-level settings"},
			{Kind: "VSphereMachine", Description: "Individual VM instances"},
			{Kind: "VSphereMachineTemplate", Description: descMachineTemplate},
		},
	},
}

// CreateGetProviderConfigHandler returns the credentials, settings and CRD
// kinds of one infrastructure provider.
func CreateGetProviderConfigHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		provider, ok := arguments["provider"].(string)
		if !ok || provider == "" {
			return nil, fmt.Errorf("provider argument is required (aws, azure, gcp, vsphere)")
		}

		config, ok := providerConfigs[capi.Provider(strings.ToLower(provider))]
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("Unknown provider: %s. Supported providers: aws, azure, gcp, vsphere", provider)), nil
		}

		return jsonResult(config)
	}
}
