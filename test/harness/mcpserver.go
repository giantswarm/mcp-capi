package harness

import (
	"context"
	"errors"
	"io"
	"time"

	mcppkg "github.com/giantswarm/mcp-capi/pkg/mcp"
)

const (
	// serverName identifies the test MCP server.
	serverName = "mcp-capi-test"
	// serverVersion is the semantic version of the test server.
	serverVersion = "0.1.0"
	// serverShutdownTimeout is the maximum time to wait for server shutdown.
	// 5 seconds is generous for a local test server that should exit quickly.
	serverShutdownTimeout = 5 * time.Second
)

// initializeMCPServer creates and starts the MCP server with cleanup registration.
func initializeMCPServer(t TestingT, kubeconfigPath string, input io.Reader, output io.WriteCloser) {
	t.Helper()

	// Create MCP server with stdio transport
	t.Log("creating MCP server with stdio transport")
	opts := mcppkg.ServerOptions{
		KubeconfigPath: kubeconfigPath,
		Transport:      mcppkg.TransportStdio,
		StdioInput:     input,
		StdioOutput:    output,
		ServerName:     serverName,
		ServerVersion:  serverVersion,
	}

	srv, err := mcppkg.NewServer(opts)
	if err != nil {
		t.Fatalf("failed to create MCP server: %v", err)
	}

	// Start MCP server in goroutine. Server errors are captured via channel
	// and logged during cleanup (while the test is still valid) rather than
	// from the goroutine directly, which could cause panics if the goroutine
	// outlives the test.
	t.Log("starting MCP server")
	done := make(chan struct{})
	errCh := make(chan error, 1)
	go func() {
		defer close(done)
		if err := srv.Run(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			errCh <- err
		}
	}()

	t.Cleanup(func() {
		srv.Shutdown()
		// Wait for server goroutine to complete with timeout to prevent deadlock
		select {
		case <-done:
			// Log any server error while the test is still valid
			select {
			case err := <-errCh:
				t.Logf("server error: %v", err)
			default:
			}
		case <-time.After(serverShutdownTimeout):
			t.Error("Timeout waiting for server shutdown")
		}
	})
}
