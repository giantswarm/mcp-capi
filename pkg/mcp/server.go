// Package mcp implements the Model Context Protocol (MCP) server for CAPI.
package mcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/giantswarm/mcp-capi/pkg/mcp/handlers"
	"github.com/mark3labs/mcp-go/server"
)

// Server represents an MCP CAPI server instance
type Server struct {
	options       ServerOptions
	mcpServer     *server.MCPServer
	serverContext *handlers.ServerContext
	cancel        context.CancelFunc
}

// NewServer creates a new MCP CAPI server with the given options
func NewServer(opts ServerOptions) (*Server, error) {
	// Initialize CAPI client
	log.Println("Initializing CAPI client...")
	capiClient, err := capi.NewClient(opts.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create CAPI client: %w", err)
	}

	// Initialize providers
	if err := capiClient.InitializeProviders(); err != nil {
		log.Printf("Warning: Failed to initialize providers: %v", err)
		log.Printf("Server will continue but some provider-specific tools may not work")
	}
	// Create server context
	serverCtx := handlers.NewServerContext(capiClient)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		opts.ServerName,
		opts.ServerVersion,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true), // subscribe, list
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	s := &Server{
		options:       opts,
		mcpServer:     mcpServer,
		serverContext: serverCtx,
	}

	// Register all tools
	if err := s.RegisterTools(handlers.BuildAllTools); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return s, nil
}

// RegisterTools registers MCP tools using the provided registration function
func (s *Server) RegisterTools(registerFunc func(*handlers.ServerContext) ([]handlers.ToolRegistration, error)) error {
	tools, err := registerFunc(s.serverContext)
	if err != nil {
		return err
	}

	for i := range tools {
		s.mcpServer.AddTool(tools[i].Tool, tools[i].Handler)
	}

	return nil
}

// Run starts the server with the configured transport
func (s *Server) Run() error {
	// Create a cancellable context for the server lifetime
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	defer cancel()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received, closing server...")
		cancel()
	}()

	fmt.Printf("Starting MCP CAPI server with %s transport...\n", s.options.Transport)

	// Start the appropriate server based on transport type
	switch s.options.Transport {
	case TransportStdio:
		return RunStdioServer(ctx, s.mcpServer, s.options.StdioInput, s.options.StdioOutput)
	case TransportSSE:
		return RunSSEServer(ctx, s.mcpServer, s.options.HTTPAddr, s.options.SSEEndpoint, s.options.MessageEndpoint)
	case TransportStreamableHTTP:
		return RunStreamableHTTPServer(ctx, s.mcpServer, s.options.HTTPAddr, s.options.HTTPEndpoint)
	default:
		return fmt.Errorf(
			"unsupported transport type: %s (supported: stdio, sse, streamable-http)",
			s.options.Transport,
		)
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	if s.cancel != nil {
		s.cancel()
	}
}
