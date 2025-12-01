// Package harness provides utilities for integration testing the MCP CAPI server.
//
// The primary type is [Harness], which sets up an isolated test environment
// including envtest (a local Kubernetes API server), MCP server, and MCP client
// using stdio transport. The harness is automatically cleaned up when the
// test completes.
//
// # Lazy Execution Model
//
// The harness uses a lazy execution model where operations are queued when
// methods are called, and only executed when [Harness.Execute] is invoked.
// This design provides:
//
//   - Clear separation between transformation (setup) and action (execution)
//   - All operations visible before execution begins
//   - Fast test setup if tests are skipped (environment only created on Execute)
//   - Single point of execution for easier debugging
//
// # Usage
//
// Use [New] to create a new test harness, chain operations using the fluent
// API, and call [Harness.Execute] to run all queued operations:
//
//	harness.New(t).
//		CreateNamespace("test").
//		CreateClusters("test", "cluster-1", "cluster-2").
//		ToolCall("capi_list_clusters").
//		WithArg("namespace", "test").
//		AssertContent("testdata/list_clusters.golden").
//		Execute()
//
// # Multiple Tool Calls
//
// Chain multiple tool calls using the fluent API. Each [ToolCall.AssertContent]
// returns to the harness, allowing continued chaining:
//
//	harness.New(t).
//		CreateNamespace("ns1").
//		CreateClusters("ns1", "cluster-1").
//		CreateNamespace("ns2").
//		CreateClusters("ns2", "cluster-2").
//		ToolCall("capi_list_clusters").
//		WithArg("namespace", "ns1").
//		AssertContent("testdata/list_ns1.golden").
//		ToolCall("capi_list_clusters").
//		WithArg("namespace", "ns2").
//		AssertContent("testdata/list_ns2.golden").
//		Execute()
//
// # Golden File Testing
//
// The package supports golden file testing via the [ToolCall.AssertContent] method.
// Set the UPDATE_GOLDEN=true environment variable to update golden files:
//
//	UPDATE_GOLDEN=true make test
//
// # Advanced Cluster Creation
//
// For tests requiring specific cluster configurations, use [Harness.Cluster] to start
// a [ClusterBuilder]. This provides fine-grained control over cluster properties:
//
//	harness.New(t).
//		Cluster("test", "my-cluster").
//		WithProvider("aws").
//		WithPhase("Provisioned").
//		WithVersion("v1.28.0").
//		WithMachines(3, 2).
//		Create().
//		Execute()
//
// # ClusterBuilder
//
// [ClusterBuilder] configures cluster properties before creation:
//
//   - [ClusterBuilder.WithProvider]: Set infrastructure provider (aws, azure, gcp, vsphere, vcd)
//   - [ClusterBuilder.WithPhase]: Set cluster phase (Pending, Provisioning, Provisioned, etc.)
//   - [ClusterBuilder.WithVersion]: Set Kubernetes version in topology spec
//   - [ClusterBuilder.WithMachines]: Set total and ready machine counts
//   - [ClusterBuilder.WithCondition]: Start a [ConditionBuilder] for cluster conditions
//   - [ClusterBuilder.WithKubeadmControlPlane]: Start a [ControlPlaneBuilder]
//   - [ClusterBuilder.Create]: Queue cluster creation and return to harness
//
// # ConditionBuilder
//
// [ConditionBuilder] configures cluster conditions:
//
//   - [ConditionBuilder.True], [ConditionBuilder.False], [ConditionBuilder.Unknown]: Set condition status
//   - [ConditionBuilder.Reason]: Set condition reason
//   - [ConditionBuilder.Message]: Set condition message
//   - [ConditionBuilder.Done]: Return to [ClusterBuilder]
//
// Example with conditions:
//
//	harness.New(t).
//		Cluster("test", "failing-cluster").
//		WithCondition("Ready").False().Reason("NotReady").Message("Cluster not ready").Done().
//		Create().
//		Execute()
//
// # ControlPlaneBuilder
//
// [ControlPlaneBuilder] configures KubeadmControlPlane resources:
//
//   - [ControlPlaneBuilder.Version]: Set Kubernetes version
//   - [ControlPlaneBuilder.Replicas]: Set number of control plane replicas
//   - [ControlPlaneBuilder.Done]: Return to [ClusterBuilder]
//
// Example with control plane:
//
//	harness.New(t).
//		Cluster("test", "ha-cluster").
//		WithKubeadmControlPlane().
//			Replicas(3).
//			Version("v1.28.0").
//			Done().
//		Create().
//		Execute()
//
// # Public API
//
// The package exports the following types for writing tests:
//
//   - [Harness]: Main test harness orchestrator
//   - [New]: Constructor for Harness
//   - [ToolCall]: Fluent builder for MCP tool calls
//   - [ClusterBuilder]: Fluent builder for cluster configuration
//   - [ConditionBuilder]: Fluent builder for cluster conditions
//   - [ControlPlaneBuilder]: Fluent builder for control plane configuration
//
// Key methods:
//
//   - [Harness.CreateNamespace], [Harness.CreateCluster], [Harness.CreateClusters]: Queue resource creation
//   - [Harness.Cluster]: Start a cluster builder for advanced configuration
//   - [Harness.ToolCall]: Start a tool call builder
//   - [Harness.Execute]: Execute all queued operations
//   - [ToolCall.WithArg], [ToolCall.WithArgs]: Add arguments to tool call
//   - [ToolCall.AssertContent]: Queue tool call and assertion, return to harness
//
// Internal implementation details (testEnv, mcpClient, callToolResult, operation types)
// remain unexported to keep the API surface minimal and allow implementation changes
// without breaking test code.
package harness
