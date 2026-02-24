package harness

import (
	"context"
	"fmt"
	"path/filepath"

	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
)

// operation represents a single deferred operation in the lazy execution model.
// Operations are queued when harness methods are called and executed when
// Execute() is invoked.
type operation interface {
	// execute performs the operation within the given execution context.
	// Errors are reported via the test interface (t.Fatal/t.Fatalf).
	execute(ctx *executionContext)

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
	ctx            context.Context
	lastToolResult *callToolResult // stores the result of the last tool call for assertions
}

// namespaceOp creates a Kubernetes namespace.
type namespaceOp struct {
	name string
}

func (op *namespaceOp) execute(ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating namespace: %s", op.name)
	ec.k8sEnv.createNamespace(ec.ctx, op.name)
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

func (op *secretOp) execute(ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating secret '%s' in namespace '%s'", op.name, op.namespace)
	ec.k8sEnv.createSecret(ec.ctx, op.namespace, op.name, op.data)
}

func (op *secretOp) describe() string {
	return fmt.Sprintf("create secret %q in namespace %q", op.name, op.namespace)
}

// clusterOp creates a CAPI Cluster resource.
type clusterOp struct {
	namespace string
	name      string
}

func (op *clusterOp) execute(ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating cluster '%s' in namespace '%s'", op.name, op.namespace)
	ec.k8sEnv.createCluster(ec.ctx, op.namespace, op.name)
}

func (op *clusterOp) describe() string {
	return fmt.Sprintf("create cluster %q in namespace %q", op.name, op.namespace)
}

// clusterBuilderOp creates a cluster with optional provider, phase, version, machine settings, and conditions.
type clusterBuilderOp struct {
	namespace         string
	name              string
	provider          string
	phase             string
	version           string
	totalMachines     int
	readyMachines     int
	controlPlane      *controlPlaneConfig
	conditions        []clusterv1.Condition
	customInfraRef    *customRef // custom InfrastructureRef (overrides provider)
	customCPRef       *customRef // custom ControlPlaneRef (overrides controlPlane)
	controlPlaneReady *bool      // explicit ControlPlaneReady status
	infraReady        *bool      // explicit InfrastructureReady status
}

func (op *clusterBuilderOp) execute(ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating cluster '%s/%s' (provider=%s, phase=%s, version=%s)", op.namespace, op.name, op.provider, op.phase, op.version)

	// Create the cluster with all spec fields and status in minimal API calls
	ec.k8sEnv.createClusterFull(ec.ctx, clusterCreateOptions{
		namespace:         op.namespace,
		name:              op.name,
		provider:          op.provider,
		phase:             op.phase,
		version:           op.version,
		conditions:        op.conditions,
		customInfraRef:    op.customInfraRef,
		controlPlaneReady: op.controlPlaneReady,
		infraReady:        op.infraReady,
	})

	// Create machines if specified
	for i := 0; i < op.totalMachines; i++ {
		machineName := fmt.Sprintf("%s-machine-%d", op.name, i)
		ready := i < op.readyMachines // First N machines are ready
		ec.k8sEnv.createMachine(ec.ctx, op.namespace, op.name, machineName, ready)
	}

	// Create control plane if specified
	if op.controlPlane != nil && op.controlPlane.kind == "KubeadmControlPlane" {
		kcpName := op.name + "-control-plane"
		ec.k8sEnv.createKubeadmControlPlane(ec.ctx, op.namespace, kcpName, op.controlPlane.version, op.controlPlane.replicas)
		ec.k8sEnv.setClusterControlPlaneRef(ec.ctx, op.namespace, op.name, kcpName)
	}

	// Set custom ControlPlaneRef if specified (overrides controlPlane)
	if op.customCPRef != nil {
		ec.k8sEnv.setClusterControlPlaneRefCustom(ec.ctx, op.namespace, op.name, op.customCPRef.kind, op.customCPRef.name)
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
	name          string
	providerID    string
	unschedulable bool
	conditions    []nodeCondition
	addresses     []nodeAddress
	taints        []nodeTaint
	capacity      nodeResources
	allocatable   nodeResources
	nodeInfo      nodeInfoConfig
}

func (op *nodeBuilderOp) execute(ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating node '%s'", op.name)
	ec.k8sEnv.createNode(ec.ctx, nodeCreateOptions{
		name:          op.name,
		providerID:    op.providerID,
		unschedulable: op.unschedulable,
		conditions:    op.conditions,
		addresses:     op.addresses,
		taints:        op.taints,
		capacity:      op.capacity,
		allocatable:   op.allocatable,
		nodeInfo:      op.nodeInfo,
	})
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
	namespace         string
	name              string
	clusterName       string
	replicas          int
	nilReplicas       bool
	version           string
	phase             string
	statusReplicas    int
	readyReplicas     int
	updatedReplicas   int
	availableReplicas int
}

func (op *machineDeploymentOp) execute(ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating MachineDeployment '%s/%s' for cluster '%s'", op.namespace, op.name, op.clusterName)
	ec.k8sEnv.createMachineDeployment(ec.ctx, machineDeploymentCreateOptions{
		namespace:         op.namespace,
		name:              op.name,
		clusterName:       op.clusterName,
		replicas:          op.replicas,
		nilReplicas:       op.nilReplicas,
		version:           op.version,
		phase:             op.phase,
		statusReplicas:    op.statusReplicas,
		readyReplicas:     op.readyReplicas,
		updatedReplicas:   op.updatedReplicas,
		availableReplicas: op.availableReplicas,
	})
}

func (op *machineDeploymentOp) describe() string {
	return fmt.Sprintf("create MachineDeployment %q in namespace %q for cluster %q", op.name, op.namespace, op.clusterName)
}

// machineSetOp creates a CAPI MachineSet resource.
type machineSetOp struct {
	namespace         string
	name              string
	clusterName       string
	replicas          int
	nilReplicas       bool
	version           string
	statusReplicas    int
	readyReplicas     int
	availableReplicas int
	infraRefKind      string
	infraRefName      string
	bootstrapKind     string
	bootstrapName     string
	ownerMDName       string
	ownerKind         string
	ownerName         string
	conditions        []machineSetCondition
	failureReason     string
	failureMessage    string
}

func (op *machineSetOp) execute(ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating MachineSet '%s/%s' for cluster '%s'", op.namespace, op.name, op.clusterName)
	ec.k8sEnv.createMachineSet(ec.ctx, machineSetCreateOptions{
		namespace:         op.namespace,
		name:              op.name,
		clusterName:       op.clusterName,
		replicas:          op.replicas,
		nilReplicas:       op.nilReplicas,
		version:           op.version,
		statusReplicas:    op.statusReplicas,
		readyReplicas:     op.readyReplicas,
		availableReplicas: op.availableReplicas,
		infraRefKind:      op.infraRefKind,
		infraRefName:      op.infraRefName,
		bootstrapKind:     op.bootstrapKind,
		bootstrapName:     op.bootstrapName,
		ownerMDName:       op.ownerMDName,
		ownerKind:         op.ownerKind,
		ownerName:         op.ownerName,
		conditions:        op.conditions,
		failureReason:     op.failureReason,
		failureMessage:    op.failureMessage,
	})
}

func (op *machineSetOp) describe() string {
	return fmt.Sprintf("create MachineSet %q in namespace %q for cluster %q", op.name, op.namespace, op.clusterName)
}

// machineBuilderOp creates a CAPI Machine resource with fine-grained field control.
type machineBuilderOp struct {
	namespace     string
	name          string
	clusterName   string
	phase         string
	version       string
	providerID    string
	nodeRefName   string
	configRefKind string
	configRefName string
	infraRefKind  string
	infraRefName  string
	conditions    []machineCondition
	addresses     []machineAddress
}

func (op *machineBuilderOp) execute(ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("creating Machine '%s/%s' for cluster '%s'", op.namespace, op.name, op.clusterName)
	ec.k8sEnv.createMachineCustom(ec.ctx, machineCustomCreateOptions{
		namespace:     op.namespace,
		name:          op.name,
		clusterName:   op.clusterName,
		phase:         op.phase,
		version:       op.version,
		providerID:    op.providerID,
		nodeRefName:   op.nodeRefName,
		configRefKind: op.configRefKind,
		configRefName: op.configRefName,
		infraRefKind:  op.infraRefKind,
		infraRefName:  op.infraRefName,
		conditions:    op.conditions,
		addresses:     op.addresses,
	})
}

func (op *machineBuilderOp) describe() string {
	return fmt.Sprintf("create Machine %q in namespace %q for cluster %q", op.name, op.namespace, op.clusterName)
}

// toolCallOp executes an MCP tool call.
type toolCallOp struct {
	toolName string
	args     map[string]any
}

func (op *toolCallOp) execute(ec *executionContext) {
	ec.t.Helper()
	ec.t.Logf("calling %s MCP tool", op.toolName)
	ec.lastToolResult = ec.mcpClient.CallTool(ec.ctx, op.toolName, op.args)
}

func (op *toolCallOp) describe() string {
	return fmt.Sprintf("call MCP tool %q", op.toolName)
}

// assertContentOp compares the last tool call result with a golden file.
// IMPORTANT: This operation must be preceded by a toolCallOp that sets lastToolResult.
// The AssertContent() method on ToolCall enforces this by queuing both operations together.
type assertContentOp struct {
	toolName   string
	goldenPath string
}

func (op *assertContentOp) execute(ec *executionContext) {
	ec.t.Helper()
	if ec.lastToolResult == nil {
		ec.t.Fatal("assertContent called without a preceding tool call")
	}
	fullGoldenPath := filepath.Join(op.toolName, op.goldenPath)
	ec.lastToolResult.assertContent(fullGoldenPath)
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

func (op *assertContentNormalizedOp) execute(ec *executionContext) {
	ec.t.Helper()
	if ec.lastToolResult == nil {
		ec.t.Fatal("assertContentNormalized called without a preceding tool call")
	}
	fullGoldenPath := filepath.Join(op.toolName, op.goldenPath)
	ec.lastToolResult.assertContentNormalized(fullGoldenPath, op.normalizers)
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

func (op *assertErrorOp) execute(ec *executionContext) {
	ec.t.Helper()
	if ec.lastToolResult == nil {
		ec.t.Fatal("assertError called without a preceding tool call")
	}
	fullGoldenPath := filepath.Join(op.toolName, op.goldenPath)
	ec.lastToolResult.assertError(fullGoldenPath)
}

func (op *assertErrorOp) describe() string {
	return fmt.Sprintf("assert error matches %q", filepath.Join(op.toolName, op.goldenPath))
}
