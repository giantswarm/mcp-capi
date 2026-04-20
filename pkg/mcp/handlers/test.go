package handlers

import (
	"context"
	"errors"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestToolHandler handles the test tool
func TestToolHandler(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	message, ok := arguments["message"].(string)
	if !ok {
		return nil, errors.New("message argument is required and must be a string")
	}

	response := "Echo from CAPI MCP Server: " + message
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: response,
			},
		},
	}, nil
}
