package harness

import (
	"context"
	"io"
	"time"
)

// defaultExecuteTimeout is the maximum duration for all operations in a single
// Execute() call. It is generous enough for slow CI environments but catches
// genuine hangs that would otherwise block until the global test timeout.
const defaultExecuteTimeout = 2 * time.Minute

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
	//
	// Safety: reading h.executed without synchronization is safe here because
	// t.Cleanup functions run after the test function has returned, establishing
	// a happens-before edge. All writes to h.executed (in Execute()) occur during
	// the test body, so no concurrent writes are possible at cleanup time.
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultExecuteTimeout)
	h.t.Cleanup(func() { cancel() })

	// Initialize test environment
	k8sEnv := h.initializeEnvironment()
	mcpClient := initializeMCP(ctx, h.t, k8sEnv.kubeconfigPath)

	// Create execution context
	execCtx := &executionContext{
		t:         h.t,
		k8sEnv:    k8sEnv,
		mcpClient: mcpClient,
	}

	// Execute all operations in order
	h.t.Logf("executing %d operations", len(h.operations))
	for i, op := range h.operations {
		h.t.Logf("[%d/%d] %s", i+1, len(h.operations), op.describe())
		op.execute(ctx, execCtx)
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
