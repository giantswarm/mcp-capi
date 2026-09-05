package cmd

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	mcppkg "github.com/giantswarm/mcp-capi/pkg/mcp"
	"github.com/giantswarm/mcp-capi/pkg/oauth"
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

		// Authentication options
		enableOAuth     bool
		downstreamOAuth bool

		// Write policy
		readOnly    bool
		gitopsGuard bool

		debug bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP CAPI server",
		Long: `Start the MCP CAPI server to provide tools for interacting
with Cluster API (CAPI) clusters via the Model Context Protocol.

Supports multiple transport types:
  - stdio: Standard input/output (default)
  - sse: Server-Sent Events over HTTP
  - streamable-http: Streamable HTTP transport

OAuth 2.1 (--enable-oauth, sse/streamable-http only) turns the server into an
OAuth resource server. Configuration comes from the environment:
  MCP_OAUTH_ISSUER                  public base URL of this server (required)
  OAUTH_REDIRECT_URL                callback URL, e.g. <issuer>/oauth/callback (required)
  MCP_OAUTH_PROVIDER                dex (default) or google
  DEX_ISSUER_URL, DEX_CLIENT_ID, DEX_CLIENT_SECRET
  DEX_K8S_AUTHENTICATOR_CLIENT_ID   Dex client the apiserver accepts as audience (cross-client scope)
  DEX_CA_FILE                       CA of a private Dex
  GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET
  OAUTH_TRUSTED_AUDIENCES           comma-separated client IDs whose ID tokens are accepted (SSO)
  MCP_OAUTH_ALLOW_PRIVATE_URLS      allow the identity provider on private addresses
  MCP_OAUTH_ENCRYPTION_KEY          32-byte key (base64 or hex) for tokens at rest
  MCP_OAUTH_ALLOW_PUBLIC_REGISTRATION
  OAUTH_STORAGE                     memory (default) or valkey
  VALKEY_URL, VALKEY_PASSWORD, VALKEY_TLS_ENABLED, VALKEY_KEY_PREFIX

With --downstream-oauth every Kubernetes API call authenticates with the
caller's own OIDC ID token; the server holds no credentials of its own and
the apiserver applies the person's RBAC.

Write policy (both on by default):
  --read-only      only the tools that read are registered; every mutating
                   Kubernetes call is refused. Pass --read-only=false to
                   offer the create/scale/upgrade/pause/delete tools.
  --gitops-guard   mutating calls on objects owned by Flux, Argo CD or a
                   Helm release are refused: the change would be reverted
                   on the next reconciliation and belongs in Git.
An object labelled giantswarm.io/prevent-deletion is never deleted.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate input parameters before starting server
			if err := validateServeFlags(transport, httpAddr, sseEndpoint, messageEndpoint, httpEndpoint); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}
			cfg := ServeConfig{
				KubeconfigPath:  kubeconfigPath,
				Transport:       transport,
				HTTPAddr:        httpAddr,
				SSEEndpoint:     sseEndpoint,
				MessageEndpoint: messageEndpoint,
				HTTPEndpoint:    httpEndpoint,
				DownstreamOAuth: downstreamOAuth,
				ReadOnly:        readOnly,
				GitOpsGuard:     gitopsGuard,
				Debug:           debug,
			}
			if enableOAuth {
				oauthCfg := oauth.ConfigFromEnv()
				cfg.OAuth = &oauthCfg
			}
			return RunServe(cfg)
		},
	}

	// Transport flags
	cmd.Flags().StringVar(&transport, "transport", "stdio", "Transport type: stdio, sse, or streamable-http")
	cmd.Flags().StringVar(&httpAddr, "http-addr", ":8080", "HTTP server address (for sse and streamable-http transports)")
	cmd.Flags().StringVar(&sseEndpoint, "sse-endpoint", "/sse", "SSE endpoint path (for sse transport)")
	cmd.Flags().StringVar(&messageEndpoint, "message-endpoint", "/message", "Message endpoint path (for sse transport)")
	cmd.Flags().StringVar(&httpEndpoint, "http-endpoint", "/mcp", "HTTP endpoint path (for streamable-http transport)")
	cmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to kubeconfig file (defaults to KUBECONFIG env var or ~/.kube/config)")

	// Authentication flags
	cmd.Flags().BoolVar(&enableOAuth, "enable-oauth", false, "Enable OAuth 2.1 authentication (sse/streamable-http only; configured via MCP_OAUTH_* and DEX_*/GOOGLE_* environment variables)")
	cmd.Flags().BoolVar(&downstreamOAuth, "downstream-oauth", false, "Authenticate every Kubernetes API call with the caller's own OIDC ID token; no kubeconfig or ServiceAccount credentials are used (requires --enable-oauth)")

	// Write policy flags
	cmd.Flags().BoolVar(&readOnly, "read-only", true, "Register only the tools that read and refuse every mutating Kubernetes call (default: true; pass --read-only=false to offer the mutating tools)")
	cmd.Flags().BoolVar(&gitopsGuard, "gitops-guard", true, "Refuse mutating calls on objects owned by a GitOps controller (Flux, Argo CD) or a Helm release; the change belongs in Git (default: true)")

	cmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")

	return cmd
}

// validateServeFlags validates the input parameters for the serve command
func validateServeFlags(transport, httpAddr, sseEndpoint, messageEndpoint, httpEndpoint string) error {
	// Validate transport type
	validTransports := []string{"stdio", "sse", "streamable-http"}
	isValidTransport := false
	for _, valid := range validTransports {
		if transport == valid {
			isValidTransport = true
			break
		}
	}
	if !isValidTransport {
		return fmt.Errorf("unsupported transport type: %s (supported: %s)", transport, strings.Join(validTransports, ", "))
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
		return fmt.Errorf("address cannot be empty")
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
		return fmt.Errorf("endpoint path cannot be empty")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("endpoint path must start with '/', got: %s", path)
	}
	if strings.Contains(path, " ") {
		return fmt.Errorf("endpoint path cannot contain spaces, got: %s", path)
	}
	return nil
}

// ServeConfig is the resolved configuration of the serve command.
type ServeConfig struct {
	KubeconfigPath  string
	Transport       string
	HTTPAddr        string
	SSEEndpoint     string
	MessageEndpoint string
	HTTPEndpoint    string
	// OAuth is nil when authentication is disabled.
	OAuth *oauth.Config
	// DownstreamOAuth makes the server act as the caller against Kubernetes.
	DownstreamOAuth bool
	// ReadOnly registers only reading tools and refuses every mutating call.
	ReadOnly bool
	// GitOpsGuard refuses mutating calls on GitOps- or Helm-owned objects.
	GitOpsGuard bool
	Debug       bool
}

// RunServe contains the main server logic with support for multiple transports
// This function is exported to allow testing
func RunServe(cfg ServeConfig) error {
	if cfg.DownstreamOAuth && cfg.OAuth == nil {
		return fmt.Errorf("invalid configuration: --downstream-oauth requires --enable-oauth")
	}
	if cfg.OAuth != nil && cfg.Transport == string(mcppkg.TransportStdio) {
		return fmt.Errorf("invalid configuration: --enable-oauth is not supported with the stdio transport")
	}

	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	// Create server options
	opts := mcppkg.ServerOptions{
		KubeconfigPath:  cfg.KubeconfigPath,
		Transport:       mcppkg.TransportType(cfg.Transport),
		HTTPAddr:        cfg.HTTPAddr,
		SSEEndpoint:     cfg.SSEEndpoint,
		MessageEndpoint: cfg.MessageEndpoint,
		HTTPEndpoint:    cfg.HTTPEndpoint,
		ServerName:      serverName,
		ServerVersion:   rootCmd.Version,
		OAuth:           cfg.OAuth,
		CallerIdentity:  cfg.DownstreamOAuth,
		ReadOnly:        cfg.ReadOnly,
		GitOpsGuard:     cfg.GitOpsGuard,
		Logger:          logger,
	}

	// Create server
	srv, err := mcppkg.NewServer(opts)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Run server
	return srv.Run()
}
