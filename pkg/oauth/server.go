package oauth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	mcpoauth "github.com/giantswarm/mcp-oauth"
	"github.com/giantswarm/mcp-oauth/handler"
	"github.com/giantswarm/mcp-oauth/providers"
	"github.com/giantswarm/mcp-oauth/providers/dex"
	"github.com/giantswarm/mcp-oauth/providers/google"
	mcpoidc "github.com/giantswarm/mcp-oauth/providers/oidc"
	"github.com/giantswarm/mcp-oauth/security"
	"github.com/giantswarm/mcp-oauth/storage"
	"github.com/giantswarm/mcp-oauth/storage/memory"
	"github.com/giantswarm/mcp-oauth/storage/valkey"
)

// Server is the OAuth 2.1 resource/authorization server of mcp-capi.
type Server struct {
	handler *handler.Handler
	server  *mcpoauth.Server
	logger  *slog.Logger
	cleanup func()
}

// NewServer builds the OAuth server from cfg. Call [Server.Close] on shutdown.
func NewServer(ctx context.Context, cfg Config, logger *slog.Logger) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("oauth: MCP_OAUTH_ISSUER must be set")
	}
	if cfg.RedirectURL == "" {
		return nil, fmt.Errorf("oauth: OAUTH_REDIRECT_URL must be set")
	}

	var (
		provider providers.Provider
		rootCAs  *x509.CertPool
		err      error
	)
	switch cfg.Provider {
	case "", ProviderDex:
		provider, rootCAs, err = newDexProvider(cfg, logger)
	case ProviderGoogle:
		provider, err = newGoogleProvider(cfg, logger)
	default:
		return nil, fmt.Errorf("oauth: unsupported MCP_OAUTH_PROVIDER %q (supported: %s, %s)", cfg.Provider, ProviderDex, ProviderGoogle)
	}
	if err != nil {
		return nil, err
	}
	return newServerWithProvider(ctx, provider, cfg, rootCAs, logger)
}

// newServerWithProvider wires a pre-built provider; tests inject a fake one.
func newServerWithProvider(_ context.Context, provider providers.Provider, cfg Config, rootCAs *x509.CertPool, logger *slog.Logger) (*Server, error) {
	enc, err := buildEncryptor(cfg, logger)
	if err != nil {
		return nil, err
	}
	store, cleanup, err := newStore(cfg, enc)
	if err != nil {
		return nil, err
	}

	serverCfg := &mcpoauth.ServerConfig{
		Issuer:                        cfg.Issuer,
		AllowPublicClientRegistration: cfg.AllowPublicRegistration,
		AllowRefreshTokenRotation:     true,
		TrustedAudiences:              cfg.TrustedAudiences,
		JWKSRootCAs:                   rootCAs,
		// A private Dex serves its JWKS from the same private address as its
		// issuer, so the one knob covers both.
		AllowPrivateIPJWKS: cfg.AllowPrivateURLs,
	}
	srv, err := mcpoauth.NewServer(provider, store, store, store, serverCfg, logger)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("oauth: create server: %w", err)
	}
	return &Server{
		handler: handler.New(srv, logger),
		server:  srv,
		logger:  logger,
		cleanup: cleanup,
	}, nil
}

// Close releases the token store.
func (s *Server) Close() {
	if s.cleanup != nil {
		s.cleanup()
	}
}

// RegisterRoutes mounts the OAuth 2.1 endpoints (authorize, callback, token,
// register, revoke) and the RFC 9728 / RFC 8414 discovery documents for the
// MCP endpoint at mcpPath.
func (s *Server) RegisterRoutes(mux *http.ServeMux, mcpPath string) {
	s.handler.RegisterOAuthRoutes(mux, handler.OAuthRoutesOptions{
		MCPPath:         mcpPath,
		IncludeMetadata: true,
	})
}

// Protect wraps an MCP handler: the bearer is validated (401 with RFC 9728
// challenge otherwise) and the caller's ID token is resolved into the request
// context for downstream Kubernetes access.
func (s *Server) Protect(next http.Handler) http.Handler {
	return s.handler.ValidateToken(s.identity(next))
}

func newDexProvider(cfg Config, logger *slog.Logger) (providers.Provider, *x509.CertPool, error) {
	if cfg.DexIssuerURL == "" || cfg.DexClientID == "" || cfg.DexClientSecret == "" {
		return nil, nil, fmt.Errorf("oauth: DEX_ISSUER_URL, DEX_CLIENT_ID and DEX_CLIENT_SECRET must be set for MCP_OAUTH_PROVIDER=%s", ProviderDex)
	}
	scopes := []string{"openid", "profile", "email", "groups", "offline_access"}
	if cfg.KubernetesAudience != "" {
		scopes = append(scopes, "audience:server:client_id:"+cfg.KubernetesAudience)
	}
	dexCfg := &dex.Config{
		IssuerURL:    cfg.DexIssuerURL,
		ClientID:     cfg.DexClientID,
		ClientSecret: cfg.DexClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
	}
	rootCAs, err := loadRootCAs(cfg.DexCAFile)
	if err != nil {
		return nil, nil, fmt.Errorf("oauth: %w", err)
	}
	if rootCAs != nil {
		logger.Info("Using custom CA for Dex and JWKS TLS verification", "caFile", cfg.DexCAFile)
	}
	switch {
	case cfg.AllowPrivateURLs:
		dexCfg.HTTPClient = mcpoidc.NewPrivateIPAllowedHTTPClient(30*time.Second, rootCAs)
	case rootCAs != nil:
		dexCfg.HTTPClient = &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: rootCAs, MinVersion: tls.VersionTLS12}},
			Timeout:   30 * time.Second,
		}
	}
	provider, err := dex.NewProvider(dexCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("oauth: create Dex provider: %w", err)
	}
	logger.Info("Using Dex OIDC provider", "issuer", cfg.DexIssuerURL, "trustedAudiences", cfg.TrustedAudiences)
	return provider, rootCAs, nil
}

func newGoogleProvider(cfg Config, logger *slog.Logger) (providers.Provider, error) {
	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		return nil, fmt.Errorf("oauth: GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET must be set for MCP_OAUTH_PROVIDER=%s", ProviderGoogle)
	}
	if cfg.DexCAFile != "" || cfg.AllowPrivateURLs {
		logger.Warn("DEX_CA_FILE and MCP_OAUTH_ALLOW_PRIVATE_URLS are ignored for the Google provider (public endpoints)")
	}
	provider, err := google.NewProvider(&google.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
	})
	if err != nil {
		return nil, fmt.Errorf("oauth: create Google provider: %w", err)
	}
	logger.Info("Using Google OAuth provider", "trustedAudiences", cfg.TrustedAudiences)
	return provider, nil
}

// loadRootCAs returns the system pool plus the PEM certificates in caFile, or
// (nil, nil) for an empty caFile.
func loadRootCAs(caFile string) (*x509.CertPool, error) {
	if caFile == "" {
		return nil, nil
	}
	pem, err := os.ReadFile(caFile) // #nosec G304 -- operator-provided path
	if err != nil {
		return nil, fmt.Errorf("read CA file %s: %w", caFile, err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("parse CA certificate from %s", caFile)
	}
	return pool, nil
}

// buildEncryptor decodes the 32-byte key (base64 or hex). (nil, nil) when
// no key is configured.
func buildEncryptor(cfg Config, logger *slog.Logger) (*security.Encryptor, error) {
	if cfg.EncryptionKey == "" {
		logger.Warn("MCP_OAUTH_ENCRYPTION_KEY is not set: OAuth tokens are stored unencrypted. Generate a key with: openssl rand -base64 32")
		return nil, nil
	}
	key, err := decodeKey(cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("oauth: decode MCP_OAUTH_ENCRYPTION_KEY: %w", err)
	}
	enc, err := security.NewEncryptor(key)
	if err != nil {
		return nil, fmt.Errorf("oauth: create encryptor: %w", err)
	}
	return enc, nil
}

// decodeKey accepts a hex-encoded (64 chars) or base64-encoded key.
func decodeKey(s string) ([]byte, error) {
	if len(s) == 64 {
		if b, err := hex.DecodeString(s); err == nil {
			return b, nil
		}
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("key must be base64 or hex encoded: %w", err)
	}
	return b, nil
}

type combinedStore interface {
	storage.TokenStore
	storage.ClientStore
	storage.FlowStore
}

func newStore(cfg Config, enc *security.Encryptor) (combinedStore, func(), error) {
	if cfg.StorageType == StorageTypeValkey {
		if cfg.ValkeyURL == "" {
			return nil, nil, fmt.Errorf("oauth: VALKEY_URL must be set when OAUTH_STORAGE=valkey")
		}
		vcfg := valkey.Config{Address: cfg.ValkeyURL, Password: cfg.ValkeyPassword}
		if cfg.ValkeyKeyPrefix != "" {
			vcfg.KeyPrefix = cfg.ValkeyKeyPrefix
		}
		if cfg.ValkeyTLS {
			vcfg.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
		}
		var opts []valkey.Option
		if enc != nil {
			opts = append(opts, valkey.WithEncryptor(enc))
		}
		s, err := valkey.New(vcfg, opts...)
		if err != nil {
			return nil, nil, fmt.Errorf("oauth: connect to Valkey at %s: %w", cfg.ValkeyURL, err)
		}
		return s, func() { s.Close() }, nil
	}
	var opts []memory.Option
	if enc != nil {
		opts = append(opts, memory.WithEncryptor(enc))
	}
	s := memory.New(opts...)
	return s, s.Stop, nil
}
