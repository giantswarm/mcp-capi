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
// It initializes the test environment (k8senv, MCP server/client) and
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

// CreateSecret queues creation of a Kubernetes secret in the given namespace.
// The data map contains the secret's key-value pairs.
func (h *Harness) CreateSecret(namespace, name string, data map[string][]byte) *Harness {
	h.t.Helper()
	h.operations = append(h.operations, &secretOp{
		namespace: namespace,
		name:      name,
		data:      data,
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
	customInfraRef  *customRef // custom InfrastructureRef (overrides provider)
	customCPRef     *customRef // custom ControlPlaneRef (overrides controlPlane)
}

// customRef holds a custom object reference for InfrastructureRef or ControlPlaneRef.
type customRef struct {
	kind string
	name string
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

// WithCustomInfraRef sets a custom InfrastructureRef with an arbitrary Kind.
// This overrides any provider set via WithProvider.
func (cb *ClusterBuilder) WithCustomInfraRef(kind, name string) *ClusterBuilder {
	cb.customInfraRef = &customRef{kind: kind, name: name}
	return cb
}

// WithControlPlaneRef sets a custom ControlPlaneRef with an arbitrary Kind and name.
// Use this to test non-KubeadmControlPlane control plane types or references
// to control plane resources that don't exist.
func (cb *ClusterBuilder) WithControlPlaneRef(kind, name string) *ClusterBuilder {
	cb.customCPRef = &customRef{kind: kind, name: name}
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
		namespace:      cb.namespace,
		name:           cb.name,
		provider:       cb.provider,
		phase:          cb.phase,
		version:        cb.version,
		totalMachines:  cb.totalMachines,
		readyMachines:  cb.readyMachines,
		controlPlane:   cb.controlPlane,
		conditions:     cb.conditions,
		customInfraRef: cb.customInfraRef,
		customCPRef:    cb.customCPRef,
	})
	return cb.harness
}

// MachineDeploymentBuilder provides a fluent API for building MachineDeployment resources.
type MachineDeploymentBuilder struct {
	harness           *Harness
	namespace         string
	name              string
	clusterName       string
	replicas          int
	version           string
	phase             string
	statusReplicas    int
	readyReplicas     int
	updatedReplicas   int
	availableReplicas int
}

// MachineDeployment starts a new MachineDeployment builder.
func (h *Harness) MachineDeployment(namespace, name string) *MachineDeploymentBuilder {
	return &MachineDeploymentBuilder{
		harness:   h,
		namespace: namespace,
		name:      name,
		replicas:  1,
	}
}

// ForCluster sets the cluster name for this MachineDeployment.
func (mdb *MachineDeploymentBuilder) ForCluster(clusterName string) *MachineDeploymentBuilder {
	mdb.clusterName = clusterName
	return mdb
}

// WithReplicas sets the desired replica count.
func (mdb *MachineDeploymentBuilder) WithReplicas(replicas int) *MachineDeploymentBuilder {
	mdb.replicas = replicas
	return mdb
}

// WithVersion sets the Kubernetes version.
func (mdb *MachineDeploymentBuilder) WithVersion(version string) *MachineDeploymentBuilder {
	mdb.version = version
	return mdb
}

// WithPhase sets the phase on the MachineDeployment status.
func (mdb *MachineDeploymentBuilder) WithPhase(phase string) *MachineDeploymentBuilder {
	mdb.phase = phase
	return mdb
}

// WithStatus sets the status replica counts.
func (mdb *MachineDeploymentBuilder) WithStatus(total, ready, updated, available int) *MachineDeploymentBuilder {
	mdb.statusReplicas = total
	mdb.readyReplicas = ready
	mdb.updatedReplicas = updated
	mdb.availableReplicas = available
	return mdb
}

// Create queues the MachineDeployment creation and returns to the harness.
func (mdb *MachineDeploymentBuilder) Create() *Harness {
	mdb.harness.t.Helper()
	mdb.harness.operations = append(mdb.harness.operations, &machineDeploymentOp{
		namespace:         mdb.namespace,
		name:              mdb.name,
		clusterName:       mdb.clusterName,
		replicas:          mdb.replicas,
		version:           mdb.version,
		phase:             mdb.phase,
		statusReplicas:    mdb.statusReplicas,
		readyReplicas:     mdb.readyReplicas,
		updatedReplicas:   mdb.updatedReplicas,
		availableReplicas: mdb.availableReplicas,
	})
	return mdb.harness
}

// MachineSetBuilder provides a fluent API for building MachineSet resources.
type MachineSetBuilder struct {
	harness           *Harness
	namespace         string
	name              string
	clusterName       string
	replicas          int
	version           string
	statusReplicas    int
	readyReplicas     int
	availableReplicas int
	infraRefKind      string
	infraRefName      string
	bootstrapKind     string
	bootstrapName     string
	ownerMDName       string
}

// MachineSet starts a new MachineSet builder.
func (h *Harness) MachineSet(namespace, name string) *MachineSetBuilder {
	return &MachineSetBuilder{
		harness:   h,
		namespace: namespace,
		name:      name,
		replicas:  1,
	}
}

// ForCluster sets the cluster name for this MachineSet.
func (msb *MachineSetBuilder) ForCluster(clusterName string) *MachineSetBuilder {
	msb.clusterName = clusterName
	return msb
}

// WithReplicas sets the desired replica count.
func (msb *MachineSetBuilder) WithReplicas(replicas int) *MachineSetBuilder {
	msb.replicas = replicas
	return msb
}

// WithVersion sets the Kubernetes version.
func (msb *MachineSetBuilder) WithVersion(version string) *MachineSetBuilder {
	msb.version = version
	return msb
}

// WithStatus sets the status replica counts.
func (msb *MachineSetBuilder) WithStatus(total, ready, available int) *MachineSetBuilder {
	msb.statusReplicas = total
	msb.readyReplicas = ready
	msb.availableReplicas = available
	return msb
}

// WithInfraRef sets the infrastructure reference.
func (msb *MachineSetBuilder) WithInfraRef(kind, name string) *MachineSetBuilder {
	msb.infraRefKind = kind
	msb.infraRefName = name
	return msb
}

// WithBootstrapRef sets the bootstrap config reference.
func (msb *MachineSetBuilder) WithBootstrapRef(kind, name string) *MachineSetBuilder {
	msb.bootstrapKind = kind
	msb.bootstrapName = name
	return msb
}

// OwnedBy sets the owner MachineDeployment name.
func (msb *MachineSetBuilder) OwnedBy(mdName string) *MachineSetBuilder {
	msb.ownerMDName = mdName
	return msb
}

// Create queues the MachineSet creation and returns to the harness.
func (msb *MachineSetBuilder) Create() *Harness {
	msb.harness.t.Helper()
	msb.harness.operations = append(msb.harness.operations, &machineSetOp{
		namespace:         msb.namespace,
		name:              msb.name,
		clusterName:       msb.clusterName,
		replicas:          msb.replicas,
		version:           msb.version,
		statusReplicas:    msb.statusReplicas,
		readyReplicas:     msb.readyReplicas,
		availableReplicas: msb.availableReplicas,
		infraRefKind:      msb.infraRefKind,
		infraRefName:      msb.infraRefName,
		bootstrapKind:     msb.bootstrapKind,
		bootstrapName:     msb.bootstrapName,
		ownerMDName:       msb.ownerMDName,
	})
	return msb.harness
}

// NodeBuilder provides a fluent API for building Kubernetes Node resources with custom properties.
type NodeBuilder struct {
	harness       *Harness
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

// nodeCondition holds a node condition configuration.
type nodeCondition struct {
	condType string
	status   string
	reason   string
	message  string
}

// nodeAddress holds a node address configuration.
type nodeAddress struct {
	addrType string
	address  string
}

// nodeTaint holds a node taint configuration.
type nodeTaint struct {
	key    string
	value  string
	effect string
}

// nodeResources holds resource quantities for a node.
type nodeResources struct {
	cpu    string
	memory string
	pods   string
}

// nodeInfoConfig holds node system information.
type nodeInfoConfig struct {
	os                      string
	osImage                 string
	kernelVersion           string
	containerRuntimeVersion string
	kubeletVersion          string
	architecture            string
}

// Node starts a new node builder with sensible defaults.
func (h *Harness) Node(name string) *NodeBuilder {
	return &NodeBuilder{
		harness: h,
		name:    name,
		nodeInfo: nodeInfoConfig{
			os:                      "linux",
			osImage:                 "Ubuntu 22.04.3 LTS",
			kernelVersion:           "5.15.0-91-generic",
			containerRuntimeVersion: "containerd://1.7.2",
			kubeletVersion:          "v1.29.0",
			architecture:            "amd64",
		},
		capacity: nodeResources{
			cpu:    "4",
			memory: "8Gi",
			pods:   "110",
		},
		allocatable: nodeResources{
			cpu:    "3500m",
			memory: "7Gi",
			pods:   "110",
		},
	}
}

// WithProviderID sets the provider ID on the node.
func (nb *NodeBuilder) WithProviderID(providerID string) *NodeBuilder {
	nb.providerID = providerID
	return nb
}

// WithUnschedulable sets the node as unschedulable (cordoned).
func (nb *NodeBuilder) WithUnschedulable(unschedulable bool) *NodeBuilder {
	nb.unschedulable = unschedulable
	return nb
}

// NodeConditionBuilder provides a fluent API for configuring a node condition.
type NodeConditionBuilder struct {
	nodeBuilder *NodeBuilder
	condType    string
	status      string
	reason      string
	message     string
}

// WithCondition starts configuring a condition for this node.
func (nb *NodeBuilder) WithCondition(condType string) *NodeConditionBuilder {
	return &NodeConditionBuilder{
		nodeBuilder: nb,
		condType:    condType,
	}
}

// Status sets the condition status ("True", "False", "Unknown").
func (ncb *NodeConditionBuilder) Status(status string) *NodeConditionBuilder {
	ncb.status = status
	return ncb
}

// Reason sets the reason for this condition.
func (ncb *NodeConditionBuilder) Reason(reason string) *NodeConditionBuilder {
	ncb.reason = reason
	return ncb
}

// Message sets the message for this condition.
func (ncb *NodeConditionBuilder) Message(message string) *NodeConditionBuilder {
	ncb.message = message
	return ncb
}

// Done returns to the NodeBuilder to continue configuration.
func (ncb *NodeConditionBuilder) Done() *NodeBuilder {
	ncb.nodeBuilder.conditions = append(ncb.nodeBuilder.conditions, nodeCondition{
		condType: ncb.condType,
		status:   ncb.status,
		reason:   ncb.reason,
		message:  ncb.message,
	})
	return ncb.nodeBuilder
}

// WithAddress adds an address to the node.
func (nb *NodeBuilder) WithAddress(addrType, address string) *NodeBuilder {
	nb.addresses = append(nb.addresses, nodeAddress{addrType: addrType, address: address})
	return nb
}

// WithTaint adds a taint to the node.
func (nb *NodeBuilder) WithTaint(key, value, effect string) *NodeBuilder {
	nb.taints = append(nb.taints, nodeTaint{key: key, value: value, effect: effect})
	return nb
}

// WithKubeletVersion sets the kubelet version.
func (nb *NodeBuilder) WithKubeletVersion(version string) *NodeBuilder {
	nb.nodeInfo.kubeletVersion = version
	return nb
}

// Create queues the node creation operation and returns to the harness.
func (nb *NodeBuilder) Create() *Harness {
	nb.harness.t.Helper()
	nb.harness.operations = append(nb.harness.operations, &nodeBuilderOp{
		name:          nb.name,
		providerID:    nb.providerID,
		unschedulable: nb.unschedulable,
		conditions:    nb.conditions,
		addresses:     nb.addresses,
		taints:        nb.taints,
		capacity:      nb.capacity,
		allocatable:   nb.allocatable,
		nodeInfo:      nb.nodeInfo,
	})
	return nb.harness
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

// AssertError queues the tool call and error assertion, then returns to the harness.
// Use this for tool calls that are expected to fail (protocol errors or tool errors).
// The goldenPath is relative to testdata/<toolName>/.
func (tc *ToolCall) AssertError(goldenPath string) *Harness {
	tc.harness.t.Helper()
	// Queue the tool call operation
	tc.harness.operations = append(tc.harness.operations, &toolCallOp{
		toolName: tc.toolName,
		args:     tc.args,
	})
	// Queue the error assertion operation
	tc.harness.operations = append(tc.harness.operations, &assertErrorOp{
		toolName:   tc.toolName,
		goldenPath: goldenPath,
	})
	return tc.harness
}
