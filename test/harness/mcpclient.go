package harness

import (
	"bytes"
	"context"
	"io"
	"path/filepath"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	// protocolVersion is the MCP protocol version this client implements.
	// See: https://modelcontextprotocol.io/specification
	protocolVersion = "2024-11-05"
	// clientName identifies this test client in MCP handshakes.
	clientName = "mcp-capi-test-client"
	// clientVersion is the semantic version of this test client.
	clientVersion = "0.1.0"
	// testdataDir is the directory containing test fixture files.
	testdataDir = "testdata"
)

// mcpClient wraps an MCP client for testing purposes
type mcpClient struct {
	t         TestingT
	client    *client.Client
	stderrBuf *bytes.Buffer
}

// callToolResult wraps an MCP CallToolResult for testing purposes
type callToolResult struct {
	t      TestingT
	Result *mcp.CallToolResult
}

// initializeMCPClient creates and initializes the MCP client with cleanup registration.
func initializeMCPClient(ctx context.Context, t TestingT, input io.Reader, output io.WriteCloser) *mcpClient {
	t.Helper()

	t.Log("creating and initializing stdio MCP client")
	client := newMCPClient(ctx, t, input, output)
	t.Cleanup(func() {
		client.close()
		if stderr := client.stderr(); stderr != "" {
			t.Logf("MCP client stderr:\n%s", stderr)
		}
	})

	return client
}

// newMCPClient creates and initializes a new test MCP client using stdio transport.
func newMCPClient(ctx context.Context, t TestingT, input io.Reader, output io.WriteCloser) *mcpClient {
	t.Helper()

	// Create buffer to capture stderr for debugging
	stderrBuf := &bytes.Buffer{}

	// Create stdio transport using NewIO for in-process communication
	// Note: NewIO takes (input, output, stderr) where:
	// - input: what the client reads from (server's output)
	// - output: what the client writes to (server's input)
	// - stderr: logging stream (captured for debugging)
	stdioTransport := transport.NewIO(input, output, io.NopCloser(stderrBuf))

	// Create client with the transport
	c := client.NewClient(stdioTransport)

	// Start the client
	if err := c.Start(ctx); err != nil {
		t.Fatalf("failed to start MCP client: %v", err)
	}

	// Initialize the connection
	initRequest := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: protocolVersion,
			ClientInfo: mcp.Implementation{
				Name:    clientName,
				Version: clientVersion,
			},
			Capabilities: mcp.ClientCapabilities{
				Experimental: map[string]any{},
			},
		},
	}

	_, err := c.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("failed to initialize MCP client: %v", err)
	}

	return &mcpClient{
		t:         t,
		client:    c,
		stderrBuf: stderrBuf,
	}
}

// CallTool calls an MCP tool with the given name and arguments
func (c *mcpClient) CallTool(ctx context.Context, toolName string, args map[string]any) *callToolResult {
	c.t.Helper()

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}

	result, err := c.client.CallTool(ctx, request)
	if err != nil {
		c.t.Fatalf("failed to call tool %s: %v", toolName, err)
	}

	return &callToolResult{
		t:      c.t,
		Result: result,
	}
}

// close closes the underlying MCP client
func (c *mcpClient) close() {
	if err := c.client.Close(); err != nil {
		c.t.Logf("failed to close MCP client: %v", err)
	}
}

// stderr returns the captured stderr output from the MCP transport.
func (c *mcpClient) stderr() string {
	return c.stderrBuf.String()
}

// extractText extracts the first text content from the MCP response.
// If multiple text contents exist, only the first one is returned.
// Use Result.Content directly to access all content items.
func (res *callToolResult) extractText() string {
	res.t.Helper()

	if res.Result == nil {
		res.t.Fatal("result is nil")
	}
	if len(res.Result.Content) == 0 {
		res.t.Fatal("result has no content")
	}

	for _, content := range res.Result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			return textContent.Text
		}
	}

	res.t.Fatal("no text content found in result")
	panic("unreachable") // Fatal calls runtime.Goexit but compiler needs a return
}

// assertContent compares the extracted text content with the specified golden file.
// The goldenPath is relative to the testdata directory.
func (res *callToolResult) assertContent(goldenPath string) {
	res.t.Helper()
	text := res.extractText()
	fullPath := filepath.Join(testdataDir, goldenPath)
	err := compareWithGolden(text, fullPath)
	if err != nil {
		res.t.Fatalf("golden file comparison failed: %v", err)
	}
}
