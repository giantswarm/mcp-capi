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

## Architecture

The MCP CAPI Server is built using:
- [mcp-go](https://github.com/mark3labs/mcp-go) - Go implementation of the Model Context Protocol
- [client-go](https://github.com/kubernetes/client-go) - Kubernetes Go client
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Kubernetes controller libraries

## Quick Start

### Prerequisites

- Go 1.22 or later
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

### Basic Usage

The server can be run in different transport modes:

```bash
# Stdio transport (default)
./bin/mcp-capi

# With environment variables
MCP_TRANSPORT=stdio ./bin/mcp-capi
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

### Infrastructure Providers
- `capi_aws_*` - AWS-specific operations
- `capi_azure_*` - Azure-specific operations
- `capi_gcp_*` - GCP-specific operations

## Resources

The server exposes CAPI data through MCP resources:

- `capi://clusters` - List of all clusters
- `capi://clusters/{name}` - Specific cluster details
- `capi://machines` - List of all machines
- `capi://providers` - Available infrastructure providers

## Development

### Project Structure

```
mcp-capi/
├── cmd/mcp-capi/       # Main application entry point
├── pkg/                # Public packages
│   ├── capi/          # CAPI client and utilities
│   ├── tools/         # MCP tool implementations
│   ├── resources/     # MCP resource handlers
│   └── prompts/       # MCP prompt definitions
├── internal/           # Private packages
├── docs/              # Documentation
└── examples/          # Usage examples
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

### Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute to this project.

## Configuration

The server can be configured through environment variables:

- `KUBECONFIG` - Path to kubeconfig file
- `MCP_TRANSPORT` - Transport type (stdio, sse, http)
- `LOG_LEVEL` - Logging level (debug, info, warn, error)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Issues and Support

- GitHub Issues: [github.com/giantswarm/mcp-capi/issues](https://github.com/giantswarm/mcp-capi/issues)
- Documentation: [docs/](docs/)

## Roadmap

See our [GitHub Issues](https://github.com/giantswarm/mcp-capi/issues) for the complete roadmap and planned features. 