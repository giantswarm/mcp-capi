# MCP CAPI Server

A Model Context Protocol (MCP) server for Cluster API (CAPI), enabling seamless integration between Large Language Models (LLMs) and Kubernetes cluster management through CAPI.

## Overview

The MCP CAPI Server provides a bridge between AI assistants (like Claude, GPT, etc.) and Cluster API, allowing natural language interactions for managing Kubernetes clusters across multiple infrastructure providers.

## Features

- **Cluster Management**: Create, update, scale, and delete Kubernetes clusters
- **Multi-Provider Support**: Works with AWS, Azure, GCP, vSphere, and more
- **Machine Operations**: Manage control plane and worker nodes
- **Real-time Monitoring**: Watch cluster status changes and events
- **Resource Discovery**: Browse CAPI resources through MCP resources
- **Guided Workflows**: Interactive prompts for complex operations
- **Multi-Transport Support**: Connect via stdio, Server-Sent Events (SSE), or Streamable HTTP
- **Enhanced CLI**: Version management, self-update capability, and comprehensive help system
- **Backwards Compatible**: Maintains compatibility with existing configurations and scripts

## Architecture

The MCP CAPI Server is built using:
- [mcp-go](https://github.com/mark3labs/mcp-go) - Go implementation of the Model Context Protocol
- [client-go](https://github.com/kubernetes/client-go) - Kubernetes Go client
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Kubernetes controller libraries

## Quick Start

### Prerequisites

- Go 1.24.4 or later
- Access to a CAPI management cluster
- Kubeconfig configured for the management cluster

### Installation

```bash
# Clone the repository
git clone https://github.com/giantswarm/mcp-capi.git
cd mcp-capi

# Build the server
make build

# Run the server
make run
```

### Self-Update

The server includes a built-in self-update mechanism:

```bash
mcp-capi self-update
```

## Usage

### CLI Commands

The MCP server provides several commands:

```bash
$ mcp-capi --help
mcp-capi is a Model Context Protocol (MCP) server that provides
tools for interacting with Cluster API (CAPI) clusters. It offers various capabilities
including cluster management, machine operations, scaling, and infrastructure
provider management.

When run without subcommands, it starts the MCP server (equivalent to 'mcp-capi serve').

Usage:
  mcp-capi [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  self-update Update mcp-capi to the latest version
  serve       Start the MCP CAPI server
  version     Print the version number of mcp-capi

Flags:
  -h, --help      help for mcp-capi
  -v, --version   version for mcp-capi

Use "mcp-capi [command] --help" for more information about a command.
```

### Basic Usage (Backwards Compatible)

Start the MCP server with default settings using stdio transport:

```bash
# Both commands are equivalent and start the server
mcp-capi
mcp-capi serve
```

### Multi-Transport Support

The server supports three transport types for different deployment scenarios:

#### Standard I/O (Default)
Best for MCP client integrations and development:

```bash
mcp-capi serve --transport stdio
```

#### Server-Sent Events (SSE)
Ideal for web applications and browser-based clients:

```bash
mcp-capi serve --transport sse --http-addr :8080
```

#### Streamable HTTP
Perfect for HTTP-based integrations and REST-like interactions:

```bash
mcp-capi serve --transport streamable-http --http-addr :8080
```

### Advanced Configuration Examples

```bash
# Run with SSE transport on custom port with custom endpoints
mcp-capi serve \
  --transport sse \
  --http-addr :9090 \
  --sse-endpoint /events \
  --message-endpoint /messages

# Run with streamable HTTP transport
mcp-capi serve \
  --transport streamable-http \
  --http-addr :8080 \
  --http-endpoint /api/mcp
```

### Version Management

```bash
# Check current version
mcp-capi version
mcp-capi --version

# Update to latest version
mcp-capi self-update
```

## Integration with AI Assistants

This MCP server can be integrated with various AI assistants that support the Model Context Protocol:

- **Cursor**: The server can be integrated with Cursor for AI-powered development
- **VSCode Insiders**: Compatible with VSCode Insiders for enhanced coding assistance
- **Claude Desktop**: Add the server to your Claude Desktop configuration
- **Custom MCP Clients**: Use any MCP-compatible client to connect

### Example MCP Client Configuration

#### Standard I/O Transport (Recommended)
```json
{
  "servers": {
    "capi": {
      "command": "/path/to/mcp-capi",
      "env": {
        "KUBECONFIG": "/path/to/kubeconfig"
      }
    }
  }
}
```

#### SSE Transport
```json
{
  "servers": {
    "capi": {
      "url": "http://localhost:8080/sse",
      "transport": "sse"
    }
  }
}
```

#### Streamable HTTP Transport
```json
{
  "servers": {
    "capi": {
      "url": "http://localhost:8080/mcp",
      "transport": "http"
    }
  }
}
```

## Available Tools

### Cluster Management
- `capi_create_cluster` - Create a new CAPI cluster
- `capi_list_clusters` - List all clusters
- `capi_get_cluster` - Get cluster details
- `capi_delete_cluster` - Delete a cluster
- `capi_scale_cluster` - Scale cluster nodes

### Machine Management
- `capi_list_machines` - List machines
- `capi_get_machine` - Get machine details
- `capi_delete_machine` - Delete a specific machine
- `capi_remediate_machine` - Trigger machine health check remediation

### MachineDeployment Operations
- `capi_create_machinedeployment` - Create new worker node pool
- `capi_list_machinedeployments` - List machine deployments
- `capi_scale_machinedeployment` - Scale worker nodes
- `capi_update_machinedeployment` - Update MachineDeployment configuration
- `capi_rollout_machinedeployment` - Trigger rolling update

### MachineSet Operations
- `capi_list_machinesets` - List machine sets
- `capi_get_machineset` - Get machine set details

### Node Operations
- `capi_drain_node` - Safely drain a node
- `capi_cordon_node` - Cordon/uncordon nodes
- `capi_node_status` - Get node status from workload cluster

### Infrastructure Provider Tools
#### Generic
- `capi_list_infrastructure_providers` - List available providers
- `capi_get_provider_config` - Get provider configuration requirements

#### AWS
- `capi_aws_list_clusters` - List AWS clusters
- `capi_aws_get_cluster` - Get AWS cluster details
- `capi_aws_create_cluster` - Create AWS cluster (placeholder)
- `capi_aws_update_vpc` - Update VPC configuration (placeholder)
- `capi_aws_manage_security_groups` - Manage security groups (placeholder)
- `capi_aws_get_machine_template` - Get/list AWS machine templates

#### Azure
- `capi_azure_list_clusters` - List Azure clusters
- `capi_azure_get_cluster` - Get Azure cluster details
- `capi_azure_manage_resource_group` - Manage resource groups (placeholder)
- `capi_azure_network_config` - Configure Azure networking (placeholder)

#### GCP
- `capi_gcp_list_clusters` - List GCP clusters
- `capi_gcp_get_cluster` - Get GCP cluster details
- `capi_gcp_manage_network` - Manage GCP networks (placeholder)

#### vSphere
- `capi_vsphere_list_clusters` - List vSphere clusters
- `capi_vsphere_get_cluster` - Get vSphere cluster details
- `capi_vsphere_manage_vms` - Manage vSphere VMs (placeholder)

## Resources

The server exposes CAPI data through MCP resources:

- `capi://clusters` - List of all clusters
- `capi://clusters/{name}` - Specific cluster details
- `capi://machines` - List of all machines
- `capi://providers` - Available infrastructure providers

## Configuration

The server can be configured through environment variables:

- `KUBECONFIG` - Path to kubeconfig file
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

## Deployment

### Docker

You can run the server in a Docker container:

```dockerfile
FROM golang:1.24.4-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mcp-capi

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mcp-capi .
EXPOSE 8080
CMD ["./mcp-capi", "serve", "--transport", "sse", "--http-addr", ":8080"]
```

### Systemd Service

Create a systemd service for automatic startup:

```ini
[Unit]
Description=MCP CAPI Server
After=network.target

[Service]
Type=simple
User=capi
ExecStart=/usr/local/bin/mcp-capi serve
Environment=KUBECONFIG=/etc/kubernetes/admin.conf
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## Development

### Project Structure

```
mcp-capi/
├── cmd/                    # Command implementations
│   ├── root.go            # Root Cobra command
│   ├── serve.go           # Server command with multi-transport support
│   ├── version.go         # Version command
│   ├── selfupdate.go      # Self-update command
│   ├── doc.go             # Package documentation
│   └── *.go               # Tool handlers and server logic
├── pkg/                   # Public packages
│   ├── capi/             # CAPI client and utilities
│   ├── tools/            # MCP tool implementations
│   ├── resources/        # MCP resource handlers
│   └── prompts/          # MCP prompt definitions
├── internal/             # Private packages
├── docs/                 # Documentation
└── examples/             # Usage examples
```

### Building

```bash
# Build for current platform
make build

# Build for multiple platforms
make release

# Run tests
make test

# Run with coverage
make test-coverage
```

### Downloading CRDs

The project includes make targets to download Custom Resource Definitions (CRDs) from upstream CAPI providers:

```bash
# Download all CRDs from all providers
make download-crds

# Download CRDs for specific providers
make download-capi-crds   # Core CAPI CRDs
make download-capa-crds   # AWS provider CRDs
make download-capz-crds   # Azure provider CRDs
make download-capv-crds   # vSphere provider CRDs
make download-capvcd-crds # Cloud Director provider CRDs
make download-capg-crds   # GCP provider CRDs
```

You can override the default versions by setting environment variables:

```bash
# Example: Download CAPI CRDs for a specific version
CAPI_VERSION=v1.11.2 make download-capi-crds
```

### Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute to this project.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Issues and Support

- GitHub Issues: [github.com/giantswarm/mcp-capi/issues](https://github.com/giantswarm/mcp-capi/issues)
- Documentation: [docs/](docs/)

## Roadmap

See our [GitHub Issues](https://github.com/giantswarm/mcp-capi/issues) for the complete roadmap and planned features.
