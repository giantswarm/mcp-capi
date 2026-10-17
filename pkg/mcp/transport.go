package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-capi/pkg/oauth"
)

// RunStdioServer runs the server with STDIO transport
func RunStdioServer(mcpSrv *mcpserver.MCPServer, stdin io.Reader, stdout io.Writer) error {
	s := mcpserver.NewStdioServer(mcpSrv)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		cancel()
	}()

	// Start the server in a goroutine so we can handle shutdown signals
	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		if err := s.Listen(ctx, stdin, stdout); err != nil {
			serverDone <- err
		}
	}()

	// Wait for server completion
	if err := <-serverDone; err != nil {
		return fmt.Errorf("server stopped with error: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Server stopped normally")

	fmt.Fprintln(os.Stderr, "Server gracefully stopped")
	return nil
}

// RunSSEServer runs the server with SSE transport. When oauthSrv is non-nil
// the SSE and message endpoints require a valid bearer and the OAuth 2.1
// endpoints are served next to them.
func RunSSEServer(mcpSrv *mcpserver.MCPServer, addr, sseEndpoint, messageEndpoint string, ctx context.Context, oauthSrv *oauth.Server) error {
	// Create SSE server with custom endpoints
	sseOpts := []mcpserver.SSEOption{
		mcpserver.WithSSEEndpoint(sseEndpoint),
		mcpserver.WithMessageEndpoint(messageEndpoint),
	}
	if oauthSrv != nil {
		sseOpts = append(sseOpts, mcpserver.WithSSEContextFunc(oauth.HTTPContextFunc))
	}
	sseServer := mcpserver.NewSSEServer(mcpSrv, sseOpts...)

	fmt.Fprintf(os.Stderr, "SSE server starting on %s\n", addr)
	fmt.Fprintf(os.Stderr, "  SSE endpoint: %s\n", sseEndpoint)
	fmt.Fprintf(os.Stderr, "  Message endpoint: %s\n", messageEndpoint)

	if oauthSrv != nil {
		mux := http.NewServeMux()
		oauthSrv.RegisterRoutes(mux, sseEndpoint)
		mux.Handle(sseEndpoint, oauthSrv.Protect(sseServer.SSEHandler()))
		mux.Handle(messageEndpoint, oauthSrv.Protect(sseServer.MessageHandler()))
		fmt.Fprintln(os.Stderr, "  OAuth 2.1 endpoints: /oauth/*")
		return serveWithShutdown(ctx, addr, mux, "SSE")
	}

	mux := http.NewServeMux()
	mux.Handle(sseEndpoint, sseServer.SSEHandler())
	mux.Handle(messageEndpoint, sseServer.MessageHandler())
	return serveWithShutdown(ctx, addr, mux, "SSE")
}

// RunStreamableHTTPServer runs the server with Streamable HTTP transport.
// When oauthSrv is non-nil the MCP endpoint requires a valid bearer and the
// OAuth 2.1 endpoints are served next to it.
func RunStreamableHTTPServer(mcpSrv *mcpserver.MCPServer, addr, endpoint string, ctx context.Context, oauthSrv *oauth.Server) error {
	// Create Streamable HTTP server with custom endpoint
	httpOpts := []mcpserver.StreamableHTTPOption{
		mcpserver.WithEndpointPath(endpoint),
	}
	if oauthSrv != nil {
		httpOpts = append(httpOpts, mcpserver.WithHTTPContextFunc(oauth.HTTPContextFunc))
	}
	httpServer := mcpserver.NewStreamableHTTPServer(mcpSrv, httpOpts...)

	fmt.Fprintf(os.Stderr, "Streamable HTTP server starting on %s\n", addr)
	fmt.Fprintf(os.Stderr, "  HTTP endpoint: %s\n", endpoint)

	if oauthSrv != nil {
		mux := http.NewServeMux()
		oauthSrv.RegisterRoutes(mux, endpoint)
		mux.Handle(endpoint, oauthSrv.Protect(httpServer))
		fmt.Fprintln(os.Stderr, "  OAuth 2.1 endpoints: /oauth/*")
		return serveWithShutdown(ctx, addr, mux, "HTTP")
	}

	mux := http.NewServeMux()
	mux.Handle(endpoint, httpServer)
	return serveWithShutdown(ctx, addr, mux, "HTTP")
}

// maxBodyBytes bounds a request body. mcp-go reads a POST body whole before
// validating it (CVE-2026-108859); the bound keeps that read small.
const maxBodyBytes = 1 << 20

// serveWithShutdown serves handler on addr, every request body bounded at
// maxBodyBytes, until ctx is done, then drains connections for up to 30
// seconds.
func serveWithShutdown(ctx context.Context, addr string, handler http.Handler, name string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	srv := &http.Server{
		Handler:           http.MaxBytesHandler(handler, maxBodyBytes),
		ReadHeaderTimeout: 30 * time.Second,
	}
	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverDone <- err
		}
	}()

	select {
	case <-ctx.Done():
		fmt.Fprintf(os.Stderr, "Shutdown signal received, stopping %s server...\n", name)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("error shutting down %s server: %w", name, err)
		}
	case err := <-serverDone:
		if err != nil {
			return fmt.Errorf("%s server stopped with error: %w", name, err)
		}
		fmt.Fprintf(os.Stderr, "%s server stopped normally\n", name)
	}

	fmt.Fprintf(os.Stderr, "%s server gracefully stopped\n", name)
	return nil
}
