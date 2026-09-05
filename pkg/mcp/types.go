package mcp

import (
	"io"
	"log/slog"
	"os"

	"github.com/giantswarm/mcp-capi/pkg/oauth"
)

// TransportType represents the type of transport for the MCP server
type TransportType string

const (
	// TransportStdio uses standard input/output
	TransportStdio TransportType = "stdio"
	// TransportSSE uses Server-Sent Events over HTTP
	TransportSSE TransportType = "sse"
	// TransportStreamableHTTP uses streamable HTTP transport
	TransportStreamableHTTP TransportType = "streamable-http"
)

// ServerOptions holds configuration options for the MCP server
type ServerOptions struct {
	// KubeconfigPath is the path to the kubeconfig file
	KubeconfigPath string

	// Transport is the type of transport to use
	Transport TransportType

	// HTTPAddr is the HTTP server address (for SSE and Streamable HTTP transports)
	HTTPAddr string

	// SSEEndpoint is the SSE endpoint path (for SSE transport)
	SSEEndpoint string

	// MessageEndpoint is the message endpoint path (for SSE transport)
	MessageEndpoint string

	// HTTPEndpoint is the HTTP endpoint path (for Streamable HTTP transport)
	HTTPEndpoint string

	// ServerName is the name of the MCP server
	ServerName string

	// ServerVersion is the version of the MCP server
	ServerVersion string

	// StdioInput is the input stream for stdio transport (default: os.Stdin)
	StdioInput io.Reader

	// StdioOutput is the output stream for stdio transport (default: os.Stdout)
	StdioOutput io.Writer

	// OAuth, when set, protects the MCP endpoint with OAuth 2.1 (sse and
	// streamable-http transports only) and exposes the authorization
	// endpoints next to it.
	OAuth *oauth.Config

	// CallerIdentity makes every Kubernetes API call authenticate with the
	// caller's own OIDC ID token instead of a kubeconfig or the pod's
	// ServiceAccount. Requires OAuth. When KubeconfigPath is set only its
	// server address and CA are used; its credentials are discarded.
	CallerIdentity bool

	// Logger receives structured logs (default: slog.Default()).
	Logger *slog.Logger
}

// DefaultServerOptions returns ServerOptions with default values
func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		Transport:       TransportStdio,
		HTTPAddr:        ":8080",
		SSEEndpoint:     "/sse",
		MessageEndpoint: "/message",
		HTTPEndpoint:    "/mcp",
		ServerName:      "mcp-capi",
		ServerVersion:   "0.1.0",
		StdioInput:      os.Stdin,
		StdioOutput:     os.Stdout,
	}
}
