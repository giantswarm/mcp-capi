# Testing Guide

This guide covers the testing approach for the MCP CAPI server project.

## Overview

The project uses:
- **Go standard testing** - `go test` with `t.Run` subtests and `t.Parallel`
- **k8senv** - Local Kubernetes API server backed by kine (no real cluster needed)
- **Golden files** - Snapshot-based output verification
- **Custom test harness** - Fluent API for integration tests

### Test Organization

```
test/
├── harness/                        # Test utilities
│   ├── doc.go                      # Package documentation
│   ├── golden.go                   # Golden file comparison utilities
│   ├── harness.go                  # Main orchestrator and builders
│   ├── mcpclient.go                # MCP client wrapper
│   ├── mcpserver.go                # MCP server initialization
│   ├── operations.go               # Operation implementations
│   ├── testenv.go                  # Kubernetes environment setup
│   └── testing.go                  # TestingT interface
└── integration/                    # Integration tests
    ├── integration_suite_test.go   # TestMain setup
    ├── capi_list_clusters_test.go  # Tool tests
    └── testdata/                   # Golden files
        └── capi_list_clusters/
```

### Why This Approach

**Fluent API Benefits:**
- Readable, declarative test setup that reads like a specification
- Method chaining reduces boilerplate and keeps tests concise
- Self-documenting: test intent is clear from the chain of operations
- Reviewers can understand test intent at a glance

**Lazy Execution (Transformation-Action) Benefits:**
- All operations visible before execution begins
- Fast test skipping: environment only created when `Execute()` is called
- Single point of execution simplifies debugging
- Clear separation between setup (what) and action (when)
- Reviewers see complete test setup in one place (no scattered setup/teardown)

## Running Tests

### Basic Commands

```bash
# Run all tests
make test

# Run tests with coverage (generates coverage.out)
make test-coverage

# Run a single test by pattern
make test-single FOCUS="lists_multiple_clusters"

# Simulate CI locally
make test-ci-pr       # Pull request checks
make test-ci-push     # Push to main checks
make test-auto-release # Auto-release workflow (requires merged_pr_event.json)
```

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `UPDATE_GOLDEN=true` | Regenerate golden files |
| `NO_COLOR=true` | Disable colored output (automatically set by `make test`) |
| `TEST_PARALLEL=N` | Maximum parallel test goroutines (default: 4) |

## Writing Integration Tests

Integration tests use a fluent API provided by the test harness. Here's a typical test:

```go
package integration_test

import (
    "testing"
    "github.com/giantswarm/mcp-capi/test/harness"
)

func TestCapiListClusters(t *testing.T) {
    t.Parallel()

    t.Run("lists multiple clusters", func(t *testing.T) {
        t.Parallel()
        namespace := "test-clusters"

        harness.New(t).
            CreateNamespace(namespace).
            CreateClusters(namespace, "cluster-1", "cluster-2").
            ToolCall("capi_list_clusters").
            WithArg("namespace", namespace).
            AssertContent("multiple.golden").
            Execute()
    })
}
```

### How It Works

1. `harness.New(t)` - Creates an isolated test environment
2. `CreateNamespace/CreateClusters` - Queues resource creation
3. `ToolCall("tool_name")` - Starts a tool call builder
4. `WithArg(key, value)` - Adds arguments to the tool call
5. `AssertContent(path)` - Queues golden file assertion
6. `Execute()` - Runs all queued operations

Operations are queued until `Execute()` is called (lazy execution model).

### Provider-Specific Clusters

Use the `Cluster()` builder with `WithProvider()` to create provider-specific clusters:

```go
// Create an AWS cluster
harness.New(t).
    Cluster(namespace, "my-cluster").
        WithProvider("aws").
        Create().
    Execute()
```

**Supported providers:**

| Provider | WithProvider Value | InfrastructureRef |
|----------|-------------------|-------------------|
| Generic | (omit) | None |
| AWS | `"aws"` | AWSCluster (v1beta2) |
| Azure | `"azure"` | AzureCluster (v1beta1) |
| GCP | `"gcp"` | GCPCluster (v1beta1) |
| vSphere | `"vsphere"` | VSphereCluster (v1beta1) |
| VCD | `"vcd"` | VCDCluster (v1beta1) |

## Golden File Testing

Golden files store expected MCP tool output for snapshot comparison.

### Why Golden Files

- **Regression detection**: Catches unintended output changes automatically
- **Executable specs**: Golden files document expected behavior and serve as living documentation
- **No manual assertions**: Complex output verified without writing individual assertions
- **Review-friendly**: Output changes appear as file diffs in PRs, making it easy to approve intentional changes or catch regressions

### Updating Golden Files

When you intentionally change tool output:

```bash
# Regenerate all golden files
UPDATE_GOLDEN=true make test

# Review changes
git diff test/integration/testdata/

# Commit if correct
git add test/integration/testdata/
```

### Golden File Location

Golden files are organized by tool name:

```
test/integration/testdata/
└── <tool_name>/
    ├── scenario_1.golden
    ├── scenario_2.golden
    └── ...
```

### When to Update vs. Investigate

**Update golden files when:**
- You intentionally changed tool output format
- You added new fields to responses
- You modified formatting functions

**Investigate failures when:**
- You didn't change output-related code
- The diff shows unexpected content changes
- Multiple unrelated tests fail

## Test Harness Quick Reference

### Harness Methods

```go
harness.New(t)                           // Create new harness
    .CreateNamespace(name)               // Create namespace
    .CreateCluster(namespace, name)      // Create single generic cluster
    .CreateClusters(namespace, names...) // Create multiple generic clusters
    .Cluster(namespace, name)            // Start cluster builder (for provider-specific)
    .ToolCall(toolName)                  // Start tool call builder
    .Execute()                           // Run all operations
```

### ToolCall Builder

```go
.ToolCall("tool_name")
    .WithArg("key", value)                  // Add single argument
    .WithArgs(map[string]any{...})          // Add multiple arguments
    .AssertContent("path/to/file.golden")   // Assert and return to harness
```

### Cluster Builder

The ClusterBuilder provides fine-grained control over cluster properties:

```go
harness.New(t).
    CreateNamespace(namespace).
    Cluster(namespace, "my-cluster").
        WithProvider("aws").
        WithPhase("Provisioned").
        WithVersion("v1.28.0").
        WithMachines(3, 3).  // total, ready
        WithCondition("Ready").True().Reason("ClusterReady").Done().
        WithCondition("ControlPlaneReady").True().Done().
        WithKubeadmControlPlane().
            Version("v1.28.0").
            Replicas(3).
            Done().
        Create().
    ToolCall("capi_list_clusters").
    WithArg("namespace", namespace).
    AssertContent("my_test.golden").
    Execute()
```

**ClusterBuilder Methods:**
- `WithProvider(provider string)` - Set provider (aws, azure, gcp, vsphere, vcd)
- `WithPhase(phase string)` - Set cluster phase (Pending, Provisioning, Provisioned, Deleting, Failed)
- `WithVersion(version string)` - Set Kubernetes version
- `WithMachines(total, ready int)` - Set machine count and ready count
- `WithCondition(type string)` - Start condition builder (returns ConditionBuilder)
- `WithKubeadmControlPlane()` - Start control plane builder (returns ControlPlaneBuilder)
- `Create()` - Queue cluster creation and return to harness

**ConditionBuilder Methods:**
- `True()` / `False()` / `Unknown()` - Set condition status
- `Reason(reason string)` - Set reason code
- `Message(message string)` - Set message
- `Done()` - Return to ClusterBuilder

**ControlPlaneBuilder Methods:**
- `Version(version string)` - Set Kubernetes version
- `Replicas(replicas int32)` - Set replica count
- `Done()` - Return to ClusterBuilder

## Adding New Tests

### Step-by-Step

1. **Create test file** (if new tool): `test/integration/<tool>_test.go`

2. **Write test using harness**:
   ```go
   t.Run("describes the scenario", func(t *testing.T) {
       t.Parallel()
       harness.New(t).
           // Setup resources
           // Call tool
           // Assert output
           Execute()
   })
   ```

3. **Generate golden file**:
   ```bash
   UPDATE_GOLDEN=true make test
   ```

4. **Verify golden file content** in `test/integration/testdata/`

5. **Run tests** to confirm:
   ```bash
   make test
   ```

### Checklist

- [ ] Test covers the main success path
- [ ] Test covers edge cases (empty results, errors)
- [ ] Golden files are reviewed and committed
- [ ] Multiple tool calls in one test share the same harness
- [ ] Provider-specific tests use `Cluster().WithProvider()`
