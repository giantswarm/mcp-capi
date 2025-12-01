package handlers

import (
	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServerContext holds shared resources for the MCP server handlers.
// It provides access to the CAPI client and other shared resources needed
// by handler functions.
type ServerContext struct {
	// CapiClient is the CAPI client used for cluster operations
	CapiClient *capi.Client
}

// NewServerContext creates a new ServerContext with the given CAPI client
func NewServerContext(capiClient *capi.Client) *ServerContext {
	return &ServerContext{
		CapiClient: capiClient,
	}
}

// ToolRegistration holds a tool definition and its handler
type ToolRegistration struct {
	Tool    mcp.Tool
	Handler server.ToolHandlerFunc
}
