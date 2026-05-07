package mcp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/giantswarm/mcp-toolkit/health"
	"github.com/giantswarm/mcp-toolkit/httpx"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// RunStdioServer runs the server with STDIO transport. Lifecycle is driven
// by stdin EOF + SIGINT/SIGTERM; no HTTP surface, no /healthz endpoint.
func RunStdioServer(mcpSrv *mcpserver.MCPServer, stdin io.Reader, stdout io.Writer, logger *slog.Logger) error {
	s := mcpserver.NewStdioServer(mcpSrv)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		cancel()
	}()

	if err := s.Listen(ctx, stdin, stdout); err != nil {
		return fmt.Errorf("stdio server: %w", err)
	}
	logger.Info("stdio server stopped")
	return nil
}

// RunSSEServer runs the SSE transport on its own *http.Server with /healthz
// and /readyz mounted on the same mux at fixed paths. Graceful shutdown is
// driven by ctx through httpx.Run.
func RunSSEServer(ctx context.Context, mcpSrv *mcpserver.MCPServer, hc *health.Health, addr, sseEndpoint, messageEndpoint string, logger *slog.Logger) error {
	sseServer := mcpserver.NewSSEServer(mcpSrv,
		mcpserver.WithSSEEndpoint(sseEndpoint),
		mcpserver.WithMessageEndpoint(messageEndpoint),
	)

	mux := http.NewServeMux()
	mux.Handle(sseEndpoint, sseServer)
	mux.Handle(messageEndpoint, sseServer)
	hc.Mount(mux)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	logger.Info("SSE server starting", "addr", addr, "sse", sseEndpoint, "message", messageEndpoint)
	return httpx.Run(ctx, srv, 30*time.Second)
}

// RunStreamableHTTPServer runs the streamable-HTTP transport on its own
// *http.Server with /healthz and /readyz mounted on the same mux. Graceful
// shutdown is driven by ctx through httpx.Run.
func RunStreamableHTTPServer(ctx context.Context, mcpSrv *mcpserver.MCPServer, hc *health.Health, addr, endpoint string, logger *slog.Logger) error {
	streamServer := mcpserver.NewStreamableHTTPServer(mcpSrv,
		mcpserver.WithEndpointPath(endpoint),
	)

	mux := http.NewServeMux()
	mux.Handle(endpoint, streamServer)
	hc.Mount(mux)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	logger.Info("streamable-HTTP server starting", "addr", addr, "endpoint", endpoint)
	return httpx.Run(ctx, srv, 30*time.Second)
}
