package handlers

import (
	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/server"
)

// Azure Provider Tools

// CreateAzureListClustersHandler lists Azure clusters
func CreateAzureListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return buildListClustersHandler(serverCtx, providerListConfig{
		header:        "Azure Clusters:\n\n",
		infraKinds:    []string{"AzureCluster", "AzureManagedCluster"},
		provider:      capi.ProviderAzure,
		providerLabel: "Azure",
		noneMsg:       "No Azure clusters found.\n",
		totalFmt:      "Total Azure clusters: %d\n",
	})
}

// CreateAzureGetClusterHandler gets details of an Azure cluster
func CreateAzureGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return buildGetClusterHandler(serverCtx, providerGetConfig{
		infraKinds:     []string{"AzureCluster", "AzureManagedCluster"},
		notProviderFmt: "Cluster %s/%s is not an Azure cluster",
		headerFmt:      "Azure Cluster: %s/%s\n\n",
		noteFmt: "\nNote: For detailed Azure infrastructure information (resource group, vnet, etc.),\n" +
			"you would need to query the AzureCluster resource directly.\n",
	})
}

// CreateAzureManageResourceGroupHandler manages Azure resource groups (placeholder)
func CreateAzureManageResourceGroupHandler(_ *ServerContext) server.ToolHandlerFunc {
	return buildPlaceholderHandler("Azure Resource Group Management (Placeholder)\n\n" +
		"This tool would manage Azure resource groups for CAPI clusters.\n" +
		"Operations would include:\n" +
		"- Creating resource groups\n" +
		"- Setting resource group tags\n" +
		"- Managing resource group policies\n" +
		"- Listing resources in a group\n\n" +
		"Note: CAPI typically creates its own resource groups,\n" +
		"but this tool would help with custom configurations.\n")
}

// CreateAzureNetworkConfigHandler configures Azure networking (placeholder)
func CreateAzureNetworkConfigHandler(_ *ServerContext) server.ToolHandlerFunc {
	return buildPlaceholderHandler("Azure Network Configuration (Placeholder)\n\n" +
		"This tool would configure Azure networking for CAPI clusters.\n" +
		"Operations would include:\n" +
		"- Creating/updating VNets\n" +
		"- Managing subnets\n" +
		"- Configuring Network Security Groups\n" +
		"- Setting up VNet peering\n" +
		"- Managing load balancers\n\n" +
		"Common configurations:\n" +
		"- Custom subnet layouts\n" +
		"- Private cluster endpoints\n" +
		"- Multi-region networking\n")
}

// GCP Provider Tools

// CreateGCPListClustersHandler lists GCP clusters
func CreateGCPListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return buildListClustersHandler(serverCtx, providerListConfig{
		header:        "GCP Clusters:\n\n",
		infraKinds:    []string{"GCPCluster", "GCPManagedCluster"},
		provider:      capi.ProviderGCP,
		providerLabel: "GCP",
		noneMsg:       "No GCP clusters found.\n",
		totalFmt:      "Total GCP clusters: %d\n",
	})
}

// CreateGCPGetClusterHandler gets details of a GCP cluster
func CreateGCPGetClusterHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
	return buildGetClusterHandler(serverCtx, providerGetConfig{
		infraKinds:     []string{"GCPCluster", "GCPManagedCluster"},
		notProviderFmt: "Cluster %s/%s is not a GCP cluster",
		headerFmt:      "GCP Cluster: %s/%s\n\n",
		noteFmt: "\nNote: For detailed GCP infrastructure information (VPC, firewall rules, etc.),\n" +
			"you would need to query the GCPCluster resource directly.\n",
	})
}

// CreateGCPManageNetworkHandler manages GCP networks (placeholder)
func CreateGCPManageNetworkHandler(_ *ServerContext) server.ToolHandlerFunc {
	return buildPlaceholderHandler("GCP Network Management (Placeholder)\n\n" +
		"This tool would manage GCP networks for CAPI clusters.\n" +
		"Operations would include:\n" +
		"- Creating/updating VPC networks\n" +
		"- Managing subnets\n" +
		"- Configuring firewall rules\n" +
		"- Setting up Cloud NAT\n" +
		"- Managing load balancers\n\n" +
		"GCP-specific features:\n" +
		"- Shared VPC support\n" +
		"- Private Google Access\n" +
		"- Cloud Interconnect integration\n")
}
