package harness

import (
	"context"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
)

// Harness holds all resources for an isolated test environment.
// Operations are queued when methods are called and executed when Execute() is invoked.
type Harness struct {
	t          TestingT    // test interface for logging and cleanup
	operations []operation // queued operations
	executed   bool        // true after Execute() has been called
}

// New creates an isolated test harness for a single test.
// Operations are queued and not executed until Execute() is called.
func New(t TestingT) *Harness {
	t.Helper()

	h := &Harness{
		t:          t,
		operations: nil,
	}

	// Register cleanup to check for forgotten Execute()
	t.Cleanup(func() {
		if !h.executed && len(h.operations) > 0 {
			t.Errorf("harness has %d queued operations but Execute() was never called", len(h.operations))
		}
	})

	return h
}

// Execute runs all queued operations.
// It initializes the test environment (envtest, MCP server/client) and
// executes each operation in order.
// Returns the harness for chaining.
func (h *Harness) Execute() *Harness {
	h.t.Helper()

	if h.executed {
		h.t.Fatal("Execute() called twice on same harness - operations already executed")
	}
	h.executed = true

	if len(h.operations) == 0 {
		h.t.Log("Execute() called with no operations queued")
		return h
	}

	// Create test context
	ctx, cancel := context.WithCancel(context.Background())
	h.t.Cleanup(func() { cancel() })

	// Initialize test environment
	k8sEnv := h.initializeEnvironment()
	mcpClient := initializeMCP(ctx, h.t, k8sEnv.kubeconfigPath)

	// Create execution context
	execCtx := &executionContext{
		t:         h.t,
		k8sEnv:    k8sEnv,
		mcpClient: mcpClient,
		ctx:       ctx,
	}

	// Execute all operations in order
	h.t.Logf("executing %d operations", len(h.operations))
	for i, op := range h.operations {
		h.t.Logf("[%d/%d] %s", i+1, len(h.operations), op.describe())
		op.execute(execCtx)
	}

	h.t.Log("all operations executed successfully")
	return h
}

// initializeEnvironment bootstraps the K8s test environment.
func (h *Harness) initializeEnvironment() *testEnv {
	h.t.Helper()
	h.t.Log("bootstrapping K8s environment")

	k8sEnv := newTestEnv(h.t)
	h.t.Cleanup(func() { k8sEnv.teardown() })
	return k8sEnv
}

// initializeMCP creates and initializes the MCP server and client.
// It sets up pipes for bidirectional stdio communication and coordinates
// the initialization of both server and client.
func initializeMCP(ctx context.Context, t TestingT, kubeconfigPath string) *mcpClient {
	t.Helper()

	// Create pipes for bidirectional communication
	// Server writes to serverOutput -> client reads from clientInput
	// Client writes to clientOutput -> server reads from serverInput
	serverInput, clientOutput := io.Pipe()
	clientInput, serverOutput := io.Pipe()
	t.Cleanup(func() {
		if err := clientOutput.Close(); err != nil {
			t.Logf("failed to close client output pipe: %v", err)
		}
		if err := serverOutput.Close(); err != nil {
			t.Logf("failed to close server output pipe: %v", err)
		}
	})

	// Initialize server and client
	initializeMCPServer(t, kubeconfigPath, serverInput, serverOutput)
	mcpClient := initializeMCPClient(ctx, t, clientInput, clientOutput)

	t.Log("MCP ready")
	return mcpClient
}

// CreateClusters queues creation of multiple clusters in the given namespace.
func (h *Harness) CreateClusters(namespace string, names ...string) *Harness {
	h.t.Helper()
	for _, name := range names {
		h.operations = append(h.operations, &clusterOp{
			namespace: namespace,
			name:      name,
		})
	}
	return h
}

// CreateCluster queues creation of a cluster in the given namespace.
func (h *Harness) CreateCluster(namespace, name string) *Harness {
	h.t.Helper()
	h.operations = append(h.operations, &clusterOp{
		namespace: namespace,
		name:      name,
	})
	return h
}

// CreateNamespace queues creation of a namespace with the given name.
func (h *Harness) CreateNamespace(name string) *Harness {
	h.t.Helper()
	h.operations = append(h.operations, &namespaceOp{
		name: name,
	})
	return h
}

// controlPlaneConfig holds the configuration for a control plane resource.
type controlPlaneConfig struct {
	kind     string // e.g., "KubeadmControlPlane"
	version  string
	replicas int32
}

// ClusterBuilder provides a fluent API for building cluster resources with custom properties.
// Similar to ToolCall, it accumulates configuration and queues the operation when finalized.
type ClusterBuilder struct {
	harness       *Harness
	namespace     string
	name          string
	provider      string // "", "aws", "azure", "gcp", "vsphere", "vcd"
	phase         string // cluster phase to set after creation
	version       string // kubernetes version to set after creation
	totalMachines int    // number of machines to create
	readyMachines int    // number of machines with NodeRef (ready)
	controlPlane  *controlPlaneConfig
	conditions    []clusterv1.Condition // conditions to set on the cluster
}

// Cluster starts a new cluster builder.
func (h *Harness) Cluster(namespace, name string) *ClusterBuilder {
	return &ClusterBuilder{
		harness:   h,
		namespace: namespace,
		name:      name,
	}
}

// WithProvider sets the infrastructure provider (aws, azure, gcp, vsphere, vcd).
func (cb *ClusterBuilder) WithProvider(provider string) *ClusterBuilder {
	cb.provider = provider
	return cb
}

// WithPhase sets the cluster phase to apply after creation.
func (cb *ClusterBuilder) WithPhase(phase string) *ClusterBuilder {
	cb.phase = phase
	return cb
}

// WithVersion sets the kubernetes version to apply after creation.
func (cb *ClusterBuilder) WithVersion(version string) *ClusterBuilder {
	cb.version = version
	return cb
}

// WithMachines sets the number of machines to create and how many should be ready.
// Ready machines will have a NodeRef set in their status.
func (cb *ClusterBuilder) WithMachines(total, ready int) *ClusterBuilder {
	cb.totalMachines = total
	cb.readyMachines = ready
	return cb
}

// ConditionBuilder provides a fluent API for configuring a cluster condition.
type ConditionBuilder struct {
	clusterBuilder *ClusterBuilder
	condType       string
	status         corev1.ConditionStatus
	reason         string
	message        string
}

// WithCondition starts configuring a condition for this cluster.
func (cb *ClusterBuilder) WithCondition(condType string) *ConditionBuilder {
	return &ConditionBuilder{
		clusterBuilder: cb,
		condType:       condType,
	}
}

// True sets the condition status to True.
func (cob *ConditionBuilder) True() *ConditionBuilder {
	cob.status = corev1.ConditionTrue
	return cob
}

// False sets the condition status to False.
func (cob *ConditionBuilder) False() *ConditionBuilder {
	cob.status = corev1.ConditionFalse
	return cob
}

// Unknown sets the condition status to Unknown.
func (cob *ConditionBuilder) Unknown() *ConditionBuilder {
	cob.status = corev1.ConditionUnknown
	return cob
}

// Reason sets the reason for this condition.
func (cob *ConditionBuilder) Reason(reason string) *ConditionBuilder {
	cob.reason = reason
	return cob
}

// Message sets the message for this condition.
func (cob *ConditionBuilder) Message(message string) *ConditionBuilder {
	cob.message = message
	return cob
}

// Done returns to the ClusterBuilder to continue configuration.
func (cob *ConditionBuilder) Done() *ClusterBuilder {
	cob.clusterBuilder.conditions = append(cob.clusterBuilder.conditions, clusterv1.Condition{
		Type:               clusterv1.ConditionType(cob.condType),
		Status:             cob.status,
		Reason:             cob.reason,
		Message:            cob.message,
		LastTransitionTime: metav1.Now(),
	})
	return cob.clusterBuilder
}

// ControlPlaneBuilder provides a fluent API for configuring the control plane.
type ControlPlaneBuilder struct {
	clusterBuilder *ClusterBuilder
	kind           string // e.g., "KubeadmControlPlane"
	version        string
	replicas       int32
}

// WithKubeadmControlPlane starts configuring a KubeadmControlPlane for this cluster.
func (cb *ClusterBuilder) WithKubeadmControlPlane() *ControlPlaneBuilder {
	return &ControlPlaneBuilder{
		clusterBuilder: cb,
		kind:           "KubeadmControlPlane",
		replicas:       1, // sensible default
	}
}

// Version sets the Kubernetes version on the control plane.
func (cpb *ControlPlaneBuilder) Version(version string) *ControlPlaneBuilder {
	cpb.version = version
	return cpb
}

// Replicas sets the number of control plane replicas.
func (cpb *ControlPlaneBuilder) Replicas(replicas int32) *ControlPlaneBuilder {
	cpb.replicas = replicas
	return cpb
}

// Done returns to the ClusterBuilder to continue configuration.
func (cpb *ControlPlaneBuilder) Done() *ClusterBuilder {
	cpb.clusterBuilder.controlPlane = &controlPlaneConfig{
		kind:     cpb.kind,
		version:  cpb.version,
		replicas: cpb.replicas,
	}
	return cpb.clusterBuilder
}

// Create queues the cluster creation operation and returns to the harness.
func (cb *ClusterBuilder) Create() *Harness {
	cb.harness.t.Helper()
	cb.harness.operations = append(cb.harness.operations, &clusterBuilderOp{
		namespace:     cb.namespace,
		name:          cb.name,
		provider:      cb.provider,
		phase:         cb.phase,
		version:       cb.version,
		totalMachines: cb.totalMachines,
		readyMachines: cb.readyMachines,
		controlPlane:  cb.controlPlane,
		conditions:    cb.conditions,
	})
	return cb.harness
}

// ToolCall provides a fluent API for MCP tool call testing.
// It accumulates tool name and arguments, then queues operations
// when finalized via AssertContent or bridge methods.
type ToolCall struct {
	harness  *Harness       // parent harness for tool execution
	toolName string         // name of the MCP tool to call
	args     map[string]any // input arguments for the tool
}

// ToolCall starts a new tool call builder.
func (h *Harness) ToolCall(toolName string) *ToolCall {
	return &ToolCall{
		harness:  h,
		toolName: toolName,
		args:     make(map[string]any),
	}
}

// WithArg adds a single argument (chainable)
func (tc *ToolCall) WithArg(key string, value any) *ToolCall {
	tc.args[key] = value
	return tc
}

// WithArgs merges the provided arguments with existing arguments (chainable).
// Arguments set via WithArg are preserved; conflicting keys are overwritten.
func (tc *ToolCall) WithArgs(args map[string]any) *ToolCall {
	for k, v := range args {
		tc.args[k] = v
	}
	return tc
}

// AssertContent queues the tool call and assertion, then returns to the harness.
// This enables continued chaining after the assertion.
// The goldenPath is relative to testdata/<toolName>/.
func (tc *ToolCall) AssertContent(goldenPath string) *Harness {
	tc.harness.t.Helper()
	// Queue the tool call operation
	tc.harness.operations = append(tc.harness.operations, &toolCallOp{
		toolName: tc.toolName,
		args:     tc.args,
	})
	// Queue the assertion operation
	tc.harness.operations = append(tc.harness.operations, &assertContentOp{
		toolName:   tc.toolName,
		goldenPath: goldenPath,
	})
	return tc.harness
}
