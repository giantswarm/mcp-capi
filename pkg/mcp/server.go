package mcp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/giantswarm/mcp-capi/pkg/mcp/handlers"
	"github.com/giantswarm/mcp-capi/pkg/oauth"
)

// Server represents an MCP CAPI server instance
type Server struct {
	options       ServerOptions
	mcpServer     *server.MCPServer
	serverContext *handlers.ServerContext
	oauth         *oauth.Server
	logger        *slog.Logger
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewServer creates a new MCP CAPI server with the given options
func NewServer(opts ServerOptions) (*Server, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	// Create context that cancels on interrupt
	ctx, cancel := context.WithCancel(context.Background())

	serverCtx, err := newServerContext(opts, logger)
	if err != nil {
		cancel()
		return nil, err
	}

	var oauthSrv *oauth.Server
	if opts.OAuth != nil {
		oauthSrv, err = oauth.NewServer(ctx, *opts.OAuth, logger)
		if err != nil {
			cancel()
			return nil, err
		}
		logger.Info("OAuth 2.1 enabled", "issuer", opts.OAuth.Issuer, "provider", opts.OAuth.Provider, "callerIdentity", opts.CallerIdentity)
	}

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
		oauth:         oauthSrv,
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Register all tools
	if err := s.RegisterTools(handlers.BuildAllTools); err != nil {
		s.Shutdown()
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return s, nil
}

func validateOptions(opts ServerOptions) error {
	if opts.CallerIdentity && opts.OAuth == nil {
		return errors.New("caller identity requires OAuth to be enabled")
	}
	if opts.OAuth != nil && opts.Transport == TransportStdio {
		return errors.New("OAuth is not supported with the stdio transport")
	}
	return nil
}

// newServerContext builds the handler context: a per-caller client factory
// when acting as the caller, else one static client.
func newServerContext(opts ServerOptions, logger *slog.Logger) (*handlers.ServerContext, error) {
	if opts.CallerIdentity {
		var (
			factory *capi.BearerClientFactory
			err     error
		)
		if opts.KubeconfigPath != "" {
			factory, err = capi.NewBearerClientFactoryFromKubeconfig(opts.KubeconfigPath)
		} else {
			factory, err = capi.NewInClusterBearerClientFactory()
		}
		if err != nil {
			return nil, fmt.Errorf("failed to prepare caller-identity Kubernetes access: %w", err)
		}
		logger.Info("Acting as the authenticated caller against the Kubernetes API; no ServiceAccount credentials are used", "apiServer", factory.Host())
		return handlers.NewCallerIdentityServerContext(factory), nil
	}

	// Initialize CAPI client
	log.Println("Initializing CAPI client...")
	capiClient, err := capi.NewClient(opts.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create CAPI client: %v", err)
	}

	// Initialize providers
	if err := capiClient.InitializeProviders(); err != nil {
		log.Printf("Warning: Failed to initialize providers: %v", err)
		log.Printf("Server will continue but some provider-specific tools may not work")
	}
	return handlers.NewServerContext(capiClient), nil
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
	defer s.Shutdown()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received, closing server...")
		s.cancel()
	}()

	fmt.Printf("Starting MCP CAPI server with %s transport...\n", s.options.Transport)

	// Start the appropriate server based on transport type
	switch s.options.Transport {
	case TransportStdio:
		return RunStdioServer(s.mcpServer, s.options.StdioInput, s.options.StdioOutput)
	case TransportSSE:
		return RunSSEServer(s.mcpServer, s.options.HTTPAddr, s.options.SSEEndpoint, s.options.MessageEndpoint, s.ctx, s.oauth)
	case TransportStreamableHTTP:
		return RunStreamableHTTPServer(s.mcpServer, s.options.HTTPAddr, s.options.HTTPEndpoint, s.ctx, s.oauth)
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
	if s.oauth != nil {
		s.oauth.Close()
		s.oauth = nil
	}
}
