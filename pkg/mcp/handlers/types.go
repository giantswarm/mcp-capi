package handlers

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-capi/pkg/capi"
)

// ServerContext holds shared resources for the MCP server handlers.
// It provides access to the CAPI client and other shared resources needed
// by handler functions.
type ServerContext struct {
	// CAPIClient is the CAPI client used for cluster operations
	CAPIClient *capi.Client
}

// NewServerContext creates a new ServerContext with the given CAPI client
func NewServerContext(capiClient *capi.Client) *ServerContext {
	return &ServerContext{
		CAPIClient: capiClient,
	}
}

// ToolRegistration holds a tool definition and its handler
type ToolRegistration struct {
	Tool    mcp.Tool
	Handler server.ToolHandlerFunc
}
