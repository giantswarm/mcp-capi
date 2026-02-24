package harness

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// operation represents a single deferred operation in the lazy execution model.
// Operations are queued when harness methods are called and executed when
// Execute() is invoked.
type operation interface {
	// execute performs the operation within the given execution context.
	// The context.Context is passed as a separate parameter following Go conventions
	// (contexts should not be stored in structs).
	// Errors are reported via the test interface (t.Fatal/t.Fatalf).
	execute(ctx context.Context, ec *executionContext)

	// describe returns a human-readable description of the operation for logging.
	describe() string
}

// executionContext holds the state needed during operation execution.
// It is created when Execute() is called and passed to each operation.
//
// Design note: This struct intentionally combines infrastructure references
// (k8sEnv, mcpClient) with execution state (lastToolResult) for simplicity.
// Operations need both infrastructure access and cross-operation state sharing.
type executionContext struct {
	t              TestingT
	k8sEnv         *testEnv
	mcpClient      *mcpClient
	lastToolResult *callToolResult // stores the result of the last tool call for assertions
}

// requireLastToolResult returns the last tool call result, or fatals if no
// preceding tool call has been executed. The opName parameter identifies the
// calling operation for the error message.
func (ec *executionContext) requireLastToolResult(opName string) *callToolResult {
	ec.t.Helper()
	if ec.lastToolResult == nil {
		ec.t.Fatalf("%s called without a preceding tool call", opName)
	}
	return ec.lastToolResult
}

// namespaceOp creates a Kubernetes namespace.
type namespaceOp struct {
	name string
}

func (op *namespaceOp) execute(ctx context.Context, ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating namespace: %s", op.name)
	ec.k8sEnv.createNamespace(ctx, op.name)
}

func (op *namespaceOp) describe() string {
	return fmt.Sprintf("create namespace %q", op.name)
}

// secretOp creates a Kubernetes Secret resource.
type secretOp struct {
	namespace string
	name      string
	data      map[string][]byte
}

func (op *secretOp) execute(ctx context.Context, ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating secret '%s' in namespace '%s'", op.name, op.namespace)
	ec.k8sEnv.createSecret(ctx, op.namespace, op.name, op.data)
}

func (op *secretOp) describe() string {
	return fmt.Sprintf("create secret %q in namespace %q", op.name, op.namespace)
}

// clusterOp creates a CAPI Cluster resource.
type clusterOp struct {
	namespace string
	name      string
}

func (op *clusterOp) execute(ctx context.Context, ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating cluster '%s' in namespace '%s'", op.name, op.namespace)
	ec.k8sEnv.createCluster(ctx, op.namespace, op.name)
}

func (op *clusterOp) describe() string {
	return fmt.Sprintf("create cluster %q in namespace %q", op.name, op.namespace)
}

// clusterBuilderOp creates a cluster with optional provider, phase, version, machine settings, and conditions.
// It embeds clusterCreateOptions for the core cluster fields and adds extra fields
// for machines and control planes that are handled in execute().
type clusterBuilderOp struct {
	clusterCreateOptions
	totalMachines int
	readyMachines int
	controlPlane  *controlPlaneConfig
	customCPRef   *customRef // custom ControlPlaneRef (overrides controlPlane)
}

func (op *clusterBuilderOp) execute(ctx context.Context, ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating cluster '%s/%s' (provider=%s, phase=%s, version=%s)", op.namespace, op.name, op.provider, op.phase, op.version)

	// Create the cluster with all spec fields and status in minimal API calls
	ec.k8sEnv.createClusterFull(ctx, op.clusterCreateOptions)

	// Create machines if specified
	for i := 0; i < op.totalMachines; i++ {
		machineName := fmt.Sprintf("%s-machine-%d", op.name, i)
		ready := i < op.readyMachines // First N machines are ready
		ec.k8sEnv.createMachine(ctx, op.namespace, op.name, machineName, ready)
	}

	// Create control plane if specified
	if op.controlPlane != nil && op.controlPlane.kind == "KubeadmControlPlane" {
		kcpName := op.name + "-control-plane"
		ec.k8sEnv.createKubeadmControlPlane(ctx, op.namespace, kcpName, op.controlPlane.version, op.controlPlane.replicas)
		ec.k8sEnv.setClusterControlPlaneRef(ctx, op.namespace, op.name, kcpName)
	}

	// Set custom ControlPlaneRef if specified (overrides controlPlane)
	if op.customCPRef != nil {
		ec.k8sEnv.setClusterControlPlaneRefCustom(ctx, op.namespace, op.name, op.customCPRef.kind, op.customCPRef.name)
	}
}

func (op *clusterBuilderOp) describe() string {
	desc := fmt.Sprintf("create cluster %q in namespace %q", op.name, op.namespace)
	if op.provider != "" {
		desc += fmt.Sprintf(" (provider: %s)", op.provider)
	}
	if op.phase != "" {
		desc += fmt.Sprintf(" (phase: %s)", op.phase)
	}
	if op.version != "" {
		desc += fmt.Sprintf(" (version: %s)", op.version)
	}
	if op.totalMachines > 0 {
		desc += fmt.Sprintf(" (machines: %d/%d ready)", op.readyMachines, op.totalMachines)
	}
	return desc
}

// nodeBuilderOp creates a Kubernetes Node resource with optional properties.
type nodeBuilderOp struct {
	nodeCreateOptions
}

func (op *nodeBuilderOp) execute(ctx context.Context, ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating node '%s'", op.name)
	ec.k8sEnv.createNode(ctx, op.nodeCreateOptions)
}

func (op *nodeBuilderOp) describe() string {
	desc := fmt.Sprintf("create node %q", op.name)
	if op.unschedulable {
		desc += " (cordoned)"
	}
	return desc
}

// machineDeploymentOp creates a CAPI MachineDeployment resource.
type machineDeploymentOp struct {
	machineDeploymentCreateOptions
}

func (op *machineDeploymentOp) execute(ctx context.Context, ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating MachineDeployment '%s/%s' for cluster '%s'", op.namespace, op.name, op.clusterName)
	ec.k8sEnv.createMachineDeployment(ctx, op.machineDeploymentCreateOptions)
}

func (op *machineDeploymentOp) describe() string {
	return fmt.Sprintf("create MachineDeployment %q in namespace %q for cluster %q", op.name, op.namespace, op.clusterName)
}

// machineSetOp creates a CAPI MachineSet resource.
type machineSetOp struct {
	machineSetCreateOptions
}

func (op *machineSetOp) execute(ctx context.Context, ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating MachineSet '%s/%s' for cluster '%s'", op.namespace, op.name, op.clusterName)
	ec.k8sEnv.createMachineSet(ctx, op.machineSetCreateOptions)
}

func (op *machineSetOp) describe() string {
	return fmt.Sprintf("create MachineSet %q in namespace %q for cluster %q", op.name, op.namespace, op.clusterName)
}

// machineBuilderOp creates a CAPI Machine resource with fine-grained field control.
type machineBuilderOp struct {
	machineCustomCreateOptions
}

func (op *machineBuilderOp) execute(ctx context.Context, ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating Machine '%s/%s' for cluster '%s'", op.namespace, op.name, op.clusterName)
	ec.k8sEnv.createMachineCustom(ctx, op.machineCustomCreateOptions)
}

func (op *machineBuilderOp) describe() string {
	return fmt.Sprintf("create Machine %q in namespace %q for cluster %q", op.name, op.namespace, op.clusterName)
}

// toolCallOp executes an MCP tool call.
type toolCallOp struct {
	toolName string
	args     map[string]any
}

func (op *toolCallOp) execute(ctx context.Context, ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("calling %s MCP tool", op.toolName)
	ec.lastToolResult = ec.mcpClient.CallTool(ctx, op.toolName, op.args)
}

func (op *toolCallOp) describe() string {
	if len(op.args) == 0 {
		return fmt.Sprintf("call MCP tool %q", op.toolName)
	}
	keys := make([]string, 0, len(op.args))
	for k := range op.args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return fmt.Sprintf("call MCP tool %q with args [%s]", op.toolName, strings.Join(keys, ", "))
}

// assertContentOp compares the last tool call result with a golden file.
// IMPORTANT: This operation must be preceded by a toolCallOp that sets lastToolResult.
// The AssertContent() method on ToolCall enforces this by queuing both operations together.
type assertContentOp struct {
	toolName   string
	goldenPath string
}

func (op *assertContentOp) execute(_ context.Context, ec *executionContext) {
	ec.t.Helper()
	result := ec.requireLastToolResult("assertContent")
	fullGoldenPath := filepath.Join(op.toolName, op.goldenPath)
	result.assertContent(fullGoldenPath)
}

func (op *assertContentOp) describe() string {
	return fmt.Sprintf("assert content matches %q", filepath.Join(op.toolName, op.goldenPath))
}

// assertContentNormalizedOp compares the last tool call result with a golden file
// after applying normalizers to both the actual output and golden file content.
type assertContentNormalizedOp struct {
	toolName    string
	goldenPath  string
	normalizers []Normalizer
}

func (op *assertContentNormalizedOp) execute(_ context.Context, ec *executionContext) {
	ec.t.Helper()
	result := ec.requireLastToolResult("assertContentNormalized")
	fullGoldenPath := filepath.Join(op.toolName, op.goldenPath)
	result.assertContentNormalized(fullGoldenPath, op.normalizers)
}

func (op *assertContentNormalizedOp) describe() string {
	return fmt.Sprintf("assert normalized content matches %q", filepath.Join(op.toolName, op.goldenPath))
}

// assertErrorOp compares the last tool call error with a golden file.
// IMPORTANT: This operation must be preceded by a toolCallOp that sets lastToolResult.
// The AssertError() method on ToolCall enforces this by queuing both operations together.
type assertErrorOp struct {
	toolName   string
	goldenPath string
}

func (op *assertErrorOp) execute(_ context.Context, ec *executionContext) {
	ec.t.Helper()
	result := ec.requireLastToolResult("assertError")
	fullGoldenPath := filepath.Join(op.toolName, op.goldenPath)
	result.assertError(fullGoldenPath)
}

func (op *assertErrorOp) describe() string {
	return fmt.Sprintf("assert error matches %q", filepath.Join(op.toolName, op.goldenPath))
}
