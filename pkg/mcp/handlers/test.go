package handlers

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// testToolResult is the result of the test tool.
type testToolResult struct {
	Echo string `json:"echo"`
}

// TestToolHandler handles the test tool: it echoes the message back as JSON.
func TestToolHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	message, ok := arguments["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message argument is required and must be a string")
	}

	return jsonResult(testToolResult{Echo: message})
}
