package handlers

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-capi/pkg/capi"
)

const (
	textContentType = "text"
	infraAPIV1Beta1 = "infrastructure.cluster.x-k8s.io/v1beta1"
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
