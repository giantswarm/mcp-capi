package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/giantswarm/mcp-toolkit/health"
	"github.com/giantswarm/mcp-toolkit/middleware/responsecap"
	"github.com/giantswarm/mcp-toolkit/middleware/timeout"
	"github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/giantswarm/mcp-capi/pkg/mcp/handlers"
)

// Server represents an MCP CAPI server instance
type Server struct {
	options       ServerOptions
	logger        *slog.Logger
	mcpServer     *server.MCPServer
	serverContext *handlers.ServerContext
	health        *health.Health
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewServer creates a new MCP CAPI server with the given options
func NewServer(opts ServerOptions) (*Server, error) {
	if opts.Logger == nil {
		return nil, fmt.Errorf("server: Logger is required")
	}
	logger := opts.Logger

	ctx, cancel := context.WithCancel(context.Background())

	hc := health.New()

	logger.Info("initializing CAPI client")
	capiClient, err := capi.NewClient(opts.KubeconfigPath)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create CAPI client: %w", err)
	}

	if err := capiClient.InitializeProviders(); err != nil {
		logger.Warn("provider initialization partial; some provider-specific tools may not work", "error", err)
	}
	serverCtx := handlers.NewServerContext(capiClient)

	mcpServer := server.NewMCPServer(
		opts.ServerName,
		opts.ServerVersion,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
		server.WithToolHandlerMiddleware(timeout.New(30*time.Second)),
		server.WithToolHandlerMiddleware(responsecap.New(responsecap.Options{})),
	)

	s := &Server{
		options:       opts,
		logger:        logger,
		mcpServer:     mcpServer,
		serverContext: serverCtx,
		health:        hc,
		ctx:           ctx,
		cancel:        cancel,
	}

	if err := s.RegisterTools(handlers.BuildAllTools); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	hc.SetReady(true)
	return s, nil
}

// RegisterTools registers MCP tools using the provided registration function
func (s *Server) RegisterTools(registerFunc func(*handlers.ServerContext) ([]handlers.ToolRegistration, error)) error {
	tools, err := registerFunc(s.serverContext)
	if err != nil {
		return err
	}

	for _, toolReg := range tools {
		s.mcpServer.AddTool(toolReg.Tool, toolReg.Handler)
	}

	return nil
}

// Run starts the server with the configured transport
func (s *Server) Run() error {
	defer s.cancel()
	defer s.health.SetReady(false)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		s.logger.Info("shutdown signal received, closing server")
		s.cancel()
	}()

	s.logger.Info("starting MCP CAPI server", "transport", s.options.Transport)

	switch s.options.Transport {
	case TransportStdio:
		return RunStdioServer(s.mcpServer, s.options.StdioInput, s.options.StdioOutput, s.logger)
	case TransportSSE:
		return RunSSEServer(s.ctx, s.mcpServer, s.health, s.options.HTTPAddr, s.options.SSEEndpoint, s.options.MessageEndpoint, s.logger)
	case TransportStreamableHTTP:
		return RunStreamableHTTPServer(s.ctx, s.mcpServer, s.health, s.options.HTTPAddr, s.options.HTTPEndpoint, s.logger)
	default:
		return fmt.Errorf("unsupported transport type: %s (supported: stdio, sse, streamable-http)", s.options.Transport)
	}
}

// Context returns the server's context
func (s *Server) Context() context.Context {
	return s.ctx
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	s.cancel()
}
