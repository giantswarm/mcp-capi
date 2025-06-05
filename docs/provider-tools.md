# Infrastructure Provider Tools

This document describes the MCP tools available for managing infrastructure-specific resources across different providers (AWS, Azure, GCP, vSphere).

## Generic Infrastructure Tools

### capi_list_infrastructure_providers
List available infrastructure providers that can be used with CAPI.

**Parameters:** None

**Example:**
```
capi_list_infrastructure_providers
```

### capi_get_provider_config
Get configuration requirements for a specific infrastructure provider.

**Parameters:**
- `provider` (required): Provider name (aws, azure, gcp, vsphere)

**Example:**
```
capi_get_provider_config --provider aws
```

## AWS Infrastructure Tools

### capi_aws_list_clusters
List all AWS clusters in the management cluster.

**Parameters:**
- `namespace` (optional): Namespace to filter clusters

**Example:**
```
capi_aws_list_clusters --namespace production
```

### capi_aws_get_cluster
Get detailed information about a specific AWS cluster.

**Parameters:**
- `namespace` (required): Cluster namespace
- `name` (required): Cluster name

**Example:**
```
capi_aws_get_cluster --namespace production --name my-aws-cluster
```

### capi_aws_create_cluster
Create AWS cluster with specific configuration (placeholder implementation).

**Parameters:**
- `namespace` (required): Cluster namespace
- `name` (required): Cluster name
- `region` (required): AWS region
- `vpc_cidr` (optional): VPC CIDR block

**Example:**
```
capi_aws_create_cluster --namespace production --name new-cluster --region us-west-2 --vpc_cidr 10.0.0.0/16
```

### capi_aws_update_vpc
Update VPC configuration for AWS clusters (placeholder implementation).

**Parameters:**
- `namespace` (required): Cluster namespace
- `name` (required): Cluster name
- `operation` (required): Operation to perform

### capi_aws_manage_security_groups
Manage security groups for AWS clusters (placeholder implementation).

**Parameters:**
- `namespace` (required): Cluster namespace
- `name` (required): Cluster name
- `operation` (required): Operation to perform

### capi_aws_get_machine_template
Get or list AWS machine templates.

**Parameters:**
- `namespace` (required): Namespace to search in
- `name` (optional): Template name (lists all if not provided)

**Example:**
```
capi_aws_get_machine_template --namespace production
```

## Azure Infrastructure Tools

### capi_azure_list_clusters
List all Azure clusters in the management cluster.

**Parameters:**
- `namespace` (optional): Namespace to filter clusters

**Example:**
```
capi_azure_list_clusters --namespace production
```

### capi_azure_get_cluster
Get detailed information about a specific Azure cluster.

**Parameters:**
- `namespace` (required): Cluster namespace
- `name` (required): Cluster name

**Example:**
```
capi_azure_get_cluster --namespace production --name my-azure-cluster
```

### capi_azure_manage_resource_group
Manage Azure resource groups (placeholder implementation).

**Parameters:**
- `namespace` (required): Cluster namespace
- `name` (required): Cluster name
- `operation` (required): Operation to perform

### capi_azure_network_config
Configure Azure networking (placeholder implementation).

**Parameters:**
- `namespace` (required): Cluster namespace
- `name` (required): Cluster name
- `operation` (required): Operation to perform

## GCP Infrastructure Tools

### capi_gcp_list_clusters
List all GCP clusters in the management cluster.

**Parameters:**
- `namespace` (optional): Namespace to filter clusters

**Example:**
```
capi_gcp_list_clusters --namespace production
```

### capi_gcp_get_cluster
Get detailed information about a specific GCP cluster.

**Parameters:**
- `namespace` (required): Cluster namespace
- `name` (required): Cluster name

**Example:**
```
capi_gcp_get_cluster --namespace production --name my-gcp-cluster
```

### capi_gcp_manage_network
Manage GCP networks (placeholder implementation).

**Parameters:**
- `namespace` (required): Cluster namespace
- `name` (required): Cluster name
- `operation` (required): Operation to perform

## vSphere Infrastructure Tools

### capi_vsphere_list_clusters
List all vSphere clusters in the management cluster.

**Parameters:**
- `namespace` (optional): Namespace to filter clusters

**Example:**
```
capi_vsphere_list_clusters --namespace production
```

### capi_vsphere_get_cluster
Get detailed information about a specific vSphere cluster.

**Parameters:**
- `namespace` (required): Cluster namespace
- `name` (required): Cluster name

**Example:**
```
capi_vsphere_get_cluster --namespace production --name my-vsphere-cluster
```

### capi_vsphere_manage_vms
Manage vSphere VMs (placeholder implementation).

**Parameters:**
- `namespace` (required): Cluster namespace
- `name` (required): Cluster name
- `operation` (required): Operation to perform

## Implementation Notes

Many of the provider-specific tools are currently placeholder implementations. Full implementations would require:

1. **Provider CRDs**: Each infrastructure provider has its own Custom Resource Definitions (CRDs) that need to be installed.

2. **Provider Controllers**: The infrastructure provider controllers must be running in the management cluster.

3. **Cloud Credentials**: Proper authentication and authorization for each cloud provider.

4. **Provider-Specific APIs**: Direct integration with provider-specific resources like AWSCluster, AzureCluster, etc.

The current implementations focus on:
- Filtering CAPI clusters by infrastructure provider type
- Showing provider configuration requirements
- Demonstrating the tool structure for future implementation

## Future Enhancements

1. **Full Provider Integration**: Implement direct manipulation of provider-specific resources.

2. **Template Management**: Create and manage infrastructure-specific machine templates.

3. **Network Operations**: Advanced networking configuration for each provider.

4. **Cost Optimization**: Tools to analyze and optimize cloud resource usage.

5. **Multi-Cloud Operations**: Tools for managing clusters across multiple providers. 