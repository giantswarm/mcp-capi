package cmd

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"

	mcppkg "github.com/giantswarm/mcp-capi/pkg/mcp"
	"github.com/spf13/cobra"
)

const (
	serverName    = "mcp-capi"
	serverVersion = "0.1.0"
)

// newServeCmd creates the Cobra command for starting the MCP server.
func newServeCmd() *cobra.Command {
	var (
		// Transport options
		transport       string
		httpAddr        string
		sseEndpoint     string
		messageEndpoint string
		httpEndpoint    string
		kubeconfigPath  string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP CAPI server",
		Long: `Start the MCP CAPI server to provide tools for interacting
with Cluster API (CAPI) clusters via the Model Context Protocol.

Supports multiple transport types:
  - stdio: Standard input/output (default)
  - sse: Server-Sent Events over HTTP
  - streamable-http: Streamable HTTP transport`,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Validate input parameters before starting server
			if err := validateServeFlags(transport, httpAddr, sseEndpoint, messageEndpoint, httpEndpoint); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}
			return RunServe(kubeconfigPath, transport, httpAddr, sseEndpoint, messageEndpoint, httpEndpoint)
		},
	}

	// Transport flags
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type: stdio, sse, or streamable-http")
	cmd.Flags().
		StringVar(&httpAddr, "http-addr", ":8080", "HTTP server address (for sse and streamable-http transports)")
	cmd.Flags().StringVar(&sseEndpoint, "sse-endpoint", "/sse", "SSE endpoint path (for sse transport)")
	cmd.Flags().StringVar(&messageEndpoint, "message-endpoint", "/message", "Message endpoint path (for sse transport)")
	cmd.Flags().StringVar(&httpEndpoint, "http-endpoint", "/mcp", "HTTP endpoint path (for streamable-http transport)")
	cmd.Flags().
		StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file (defaults to KUBECONFIG env var or ~/.kube/config)")

	return cmd
}

// validateServeFlags validates the input parameters for the serve command
func validateServeFlags(transport, httpAddr, sseEndpoint, messageEndpoint, httpEndpoint string) error {
	// Validate transport type
	validTransports := []string{"stdio", "sse", "streamable-http"}
	isValidTransport := slices.Contains(validTransports, transport)
	if !isValidTransport {
		return fmt.Errorf(
			"unsupported transport type: %s (supported: %s)",
			transport,
			strings.Join(validTransports, ", "),
		)
	}

	// Validate HTTP address for non-stdio transports
	if transport != "stdio" {
		if err := validateHTTPAddr(httpAddr); err != nil {
			return fmt.Errorf("invalid http-addr: %w", err)
		}
	}

	// Validate endpoint paths
	endpoints := map[string]string{
		"sse-endpoint":     sseEndpoint,
		"message-endpoint": messageEndpoint,
		"http-endpoint":    httpEndpoint,
	}

	for name, endpoint := range endpoints {
		if err := validateEndpointPath(endpoint); err != nil {
			return fmt.Errorf("invalid %s: %w", name, err)
		}
	}

	return nil
}

// validateHTTPAddr validates HTTP address format
func validateHTTPAddr(addr string) error {
	if addr == "" {
		return errors.New("address cannot be empty")
	}

	// Parse the address
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	// Validate port
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid port number: %w", err)
		}
		if port < 1 || port > 65535 {
			return fmt.Errorf("port number must be between 1 and 65535, got: %d", port)
		}
	}

	// Validate host (if specified)
	if host != "" && net.ParseIP(host) == nil && host != "localhost" {
		return fmt.Errorf("invalid host address: %s", host)
	}

	return nil
}

// validateEndpointPath validates HTTP endpoint paths
func validateEndpointPath(path string) error {
	if path == "" {
		return errors.New("endpoint path cannot be empty")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("endpoint path must start with '/', got: %s", path)
	}
	if strings.Contains(path, " ") {
		return fmt.Errorf("endpoint path cannot contain spaces, got: %s", path)
	}
	return nil
}

// RunServe contains the main server logic with support for multiple transports
// This function is exported to allow testing
func RunServe(kubeconfigPath, transport, httpAddr, sseEndpoint, messageEndpoint, httpEndpoint string) error {
	// Create server options
	opts := mcppkg.ServerOptions{
		KubeconfigPath:  kubeconfigPath,
		Transport:       mcppkg.TransportType(transport),
		HTTPAddr:        httpAddr,
		SSEEndpoint:     sseEndpoint,
		MessageEndpoint: messageEndpoint,
		HTTPEndpoint:    httpEndpoint,
		ServerName:      serverName,
		ServerVersion:   rootCmd.Version,
	}

	// Create server
	srv, err := mcppkg.NewServer(opts)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Run server
	return srv.Run()
}
