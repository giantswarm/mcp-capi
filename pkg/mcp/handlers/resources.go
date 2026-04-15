package handlers

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestResourceHandler handles the test resource
func TestResourceHandler(_ context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "text/plain",
			Text:     "This is a test resource from the CAPI MCP server.",
		},
	}, nil
}
