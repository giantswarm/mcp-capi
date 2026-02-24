# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MCP CAPI Server is a Model Context Protocol (MCP) server that bridges AI assistants with Cluster API (CAPI) for Kubernetes cluster management. It implements the MCP specification using mcp-go and provides tools for cluster operations across multiple cloud providers (AWS, Azure, GCP, vSphere).

## Development Commands

### Building and Running
```bash
# Build the binary for current platform
make build

# Install to GOPATH
make install

# Run the server (stdio transport by default)
make build && ./mcp-capi
# or explicitly:
./mcp-capi serve

# Run with SSE transport for HTTP clients
./mcp-capi serve --transport sse --http-addr :8080

# Run with streamable HTTP transport
./mcp-capi serve --transport streamable-http --http-addr :8080
```

### Testing
```bash
# Run all tests
make test

# Run all tests with coverage (generates coverage.out)
make test-coverage

# Run YAML linter
make check
```

### Version Management
```bash
# Check version
./mcp-capi version

# Update to latest release
./mcp-capi self-update
```

## Architecture

### Core Components

**Entry Point Flow:**
- `main.go` → `cmd.Execute()` → defaults to `serve` command if no subcommand specified
- Version is injected at build time by goreleaser

**Command Structure (`cmd/`):**
- `root.go` - Cobra root command setup, defaults to serving when no subcommand given
- `serve.go` - MCP server initialization, multi-transport support (stdio/SSE/streamable-http)
- `version.go` - Version information display
- `selfupdate.go` - Self-update mechanism using go-selfupdate

**MCP Handlers (`pkg/mcp/handlers/`):**
- `registry.go` - Tool registration via `BuildAllTools()`
- `cluster.go` - Cluster management tool handlers
- `machine.go` - Machine and node operation tool handlers
- `provider_aws.go`, `provider_azure_gcp.go`, `provider_vsphere.go`, `provider_generic.go` - Provider-specific tools
- `resources.go` - Resource handlers
- `types.go` - ServerContext and ToolRegistration types

**CAPI Client (`pkg/capi/`):**
- `client.go` - Core Kubernetes/CAPI client with controller-runtime integration
- `providers.go` - Infrastructure provider detection and scheme registration
- `utils.go` - Formatting utilities and status helpers

### Key Design Patterns

**ServerContext Pattern:**
The `ServerContext` struct (in `pkg/mcp/handlers/types.go`) holds shared resources (like `CapiClient`) that are passed to all tool handlers. Tool handlers are created via factory functions that capture the ServerContext:

```go
func createListClustersHandler(serverCtx *ServerContext) server.ToolHandlerFunc {
    return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Handler implementation uses serverCtx.capiClient
    }
}
```

**Dual-Client Architecture:**
The CAPI client maintains two Kubernetes clients:
- `k8sClient` (kubernetes.Interface) - Standard K8s operations (nodes, secrets)
- `ctrlClient` (controller-runtime/client) - CAPI custom resources (Clusters, Machines)

**Provider Abstraction:**
Infrastructure providers (AWS, Azure, GCP, vSphere) are detected dynamically via the cluster's `InfrastructureRef.Kind`. Provider-specific resources are accessed as unstructured objects since provider schemes aren't all imported.

### Transport Layers

The server supports three transport types for different integration scenarios:

1. **stdio** (default) - Standard input/output, ideal for MCP client integrations
2. **sse** - Server-Sent Events over HTTP, for web applications
3. **streamable-http** - HTTP-based transport for REST-like interactions

All transports share the same tool registration and handler logic.

## Tool Categories

Tools are registered via `pkg/mcp/handlers/registry.go:BuildAllTools()` and organized by functionality:

- **Cluster Management**: create, list, get, delete, scale, upgrade, pause/resume clusters
- **Machine Operations**: list, get, delete, remediate machines
- **MachineDeployment**: create, list, scale, update, rollout machine deployments
- **MachineSet**: list, get machine sets
- **Node Operations**: drain, cordon/uncordon, status checks
- **Provider-Specific**: Tools for AWS, Azure, GCP, vSphere infrastructure

Each tool handler follows the pattern:
1. Extract and validate arguments from `request.GetArguments()`
2. Call appropriate `capiClient` method
3. Format response using `mcp.TextContent`
4. Return `*mcp.CallToolResult`

## Important Implementation Notes

### Client Initialization
- The CAPI client loads kubeconfig from: explicit path → in-cluster → `KUBECONFIG` env var → `~/.kube/config`
- `InitializeProviders()` must be called after client creation to register CAPI schemes
- Provider initialization failures are logged as warnings but don't block server startup

### Cluster Operations
- Cluster creation (`CreateCluster`) creates only the basic Cluster resource; production implementations need infrastructure resources, control plane, and machine deployments
- Deletion safety checks prevent deleting healthy clusters unless `force=true`
- Upgrade operations update control plane first, then optionally workers
- Scaling targets either "controlplane" or "workers" (MachineDeployments)

### Machine Management
- Machines have a `NodeRef` in their status linking to the underlying Kubernetes node
- Machine deletion checks health conditions and control plane status unless forced
- Node operations (drain, cordon) require mapping machine names to node names via `NodeRef`

### Status and Health
- Cluster status aggregates: Phase, Ready condition, ControlPlaneReady, InfrastructureReady
- Health checks examine conditions, machine readiness, and component status
- Version information comes from Topology spec or KubeadmControlPlane spec

### Error Handling
- All tool handlers return errors that are automatically formatted by MCP
- User-facing errors should be descriptive and suggest remediation steps
- Provider initialization failures are non-fatal (server continues with reduced functionality)

## Code Style and Conventions

- Tool handlers return detailed, formatted output with emojis (✅, ❌, ⚠️, 🚀, etc.) for user-friendliness
- Safety checks provide clear warnings and require explicit confirmation (force flags)
- All CAPI operations use `context.Context` for cancellation support
- Namespace and name are required parameters for most operations
- Use `strings.Builder` for constructing multi-line output

## Testing

### Test Organization

- Test files use `_test.go` suffix
- Unit tests are located alongside source files
- Integration tests are in `test/integration/`
- Test utilities are in `test/harness/`

### Running Tests

**IMPORTANT:** Never run `go test` directly. Always use `make test` which sets up the required environment (envtest binaries, environment variables, etc.).

```bash
# Run all tests (unit + integration)
make test

# Disable color output (useful for CI)
NO_COLOR=true make test

# Simulate CI locally
make test-ci-pr    # For pull request checks
make test-ci-push  # For push checks

# Run a single test by pattern
make test-single FOCUS="should list clusters"

# Run tests in a specific file
make test-single FOCUS_FILE="test/integration/cluster_test.go"
```

### Golden File Testing

Integration tests use **golden files** to verify exact MCP server output, ensuring response format consistency.

**How it works:**
1. Tests call MCP tools and extract text content
2. Output is compared against golden files in `test/integration/testdata/`
3. Tests fail if output doesn't match, showing a clear diff

**Updating Golden Files:**

```bash
# Regenerate all golden files (after intentional output changes)
UPDATE_GOLDEN=true make test

# Then verify the changes and commit updated golden files
git diff test/integration/testdata/
```

**Namespace Handling:**

The test framework uses fixed namespace names for reproducibility:
- `test-clusters` - Primary test namespace
- `other-clusters` - Secondary test namespace
- `multi-ns-1`, `multi-ns-2` - Multi-namespace test scenarios

Golden files contain these actual namespace names directly, ensuring deterministic comparisons.

**Golden File Utilities:**

Located in `test/harness/golden.go` (package-private functions):
- `compareWithGolden(text, goldenPath)` - Compare output against golden file
- `updateGoldenFile(text, goldenPath)` - Write/update a golden file

**When to Update Golden Files:**
- After intentionally changing MCP tool output format
- When adding new fields to responses
- After modifying `capi.FormatClusterInfo()` or similar formatting functions
- Never update to make failing tests pass without understanding why they failed

**Best Practices:**
- Review golden file diffs carefully before committing
- Use descriptive golden file names that match test scenarios
- Keep golden files in version control to track output changes

## Configuration

Environment variables:
- `KUBECONFIG` - Path to kubeconfig for management cluster access
- `LOG_LEVEL` - Logging verbosity (debug, info, warn, error)

## MCP Integration

To integrate with MCP clients (Claude Desktop, Cursor, VSCode):

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

## Common Patterns When Adding New Tools

1. Define tool in `pkg/mcp/handlers/registry.go:BuildAllTools()` using `mcp.NewTool()` with typed parameters
2. Create handler function in appropriate file in `pkg/mcp/handlers/` using factory pattern with ServerContext
3. Implement business logic in `pkg/capi/` if it involves CAPI operations
4. Format output with clear sections, status indicators, and helpful next-steps guidance
5. Include safety checks and validation for destructive operations
6. Return detailed error messages with context for debugging

## Provider-Specific Considerations

- AWS uses APIVersion `infrastructure.cluster.x-k8s.io/v1beta2`
- Azure/GCP/vSphere use `v1beta1`
- Provider detection is based on `InfrastructureRef.Kind` (e.g., `AWSCluster`, `AzureCluster`)
- Provider-specific tools are in `pkg/mcp/handlers/`: `provider_aws.go`, `provider_azure_gcp.go`, `provider_vsphere.go`, `provider_generic.go`

## Deployment Considerations

- For production, run as systemd service or in Docker container
- SSE/HTTP transports are suitable for multi-client scenarios
- stdio transport is recommended for single-client MCP integrations
- Server includes graceful shutdown handling via signal handlers


## Issue Management

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

### Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

