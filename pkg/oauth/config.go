package oauth

import (
	"os"
	"strings"
)

// Config holds everything needed to run the OAuth 2.1 resource server.
// Every field maps to an environment variable (see [ConfigFromEnv]) so the
// Helm chart can configure the server without flags.
type Config struct {
	// Issuer is the public base URL of this MCP server, e.g.
	// "https://mcp-capi.example.com". It is the OAuth issuer identifier and
	// the base of every OAuth endpoint URL.
	Issuer string

	// EncryptionKey is the AES-256-GCM key for tokens at rest: 32 bytes,
	// base64- or hex-encoded. Empty stores tokens unencrypted (development).
	EncryptionKey string

	// AllowPublicRegistration permits unauthenticated dynamic client
	// registration (RFC 7591). Development / MCP Inspector only.
	AllowPublicRegistration bool

	// StorageType selects the token store: "memory" (default) or "valkey".
	StorageType string

	// ValkeyURL is the Valkey address ("host:port"); required for "valkey".
	ValkeyURL string

	// ValkeyPassword is the optional Valkey password.
	ValkeyPassword string

	// ValkeyTLS enables TLS for the Valkey connection.
	ValkeyTLS bool

	// ValkeyKeyPrefix namespaces the keys (default "mcp:").
	ValkeyKeyPrefix string

	// Provider is the upstream identity provider: [ProviderDex] (default) or
	// [ProviderGoogle].
	Provider string

	// RedirectURL is the callback registered at the identity provider, e.g.
	// "https://mcp-capi.example.com/oauth/callback".
	RedirectURL string

	// GoogleClientID / GoogleClientSecret configure the Google provider.
	GoogleClientID     string
	GoogleClientSecret string

	// DexIssuerURL, DexClientID and DexClientSecret configure the Dex provider.
	DexIssuerURL    string
	DexClientID     string
	DexClientSecret string

	// KubernetesAudience is the Dex client ID the Kubernetes API server
	// accepts as token audience (Giant Swarm: "dex-k8s-authenticator"). When
	// set, interactive logins through this server request the cross-client
	// scope "audience:server:client_id:<id>" so the resulting ID token is
	// valid at the apiserver. Requires this server's Dex client to be a
	// trusted peer of that client. Forwarded tokens are unaffected: they
	// already carry the audiences the aggregator requested.
	KubernetesAudience string

	// TrustedAudiences lists the OAuth client IDs whose ID tokens are
	// accepted as bearers (single sign-on). When muster forwards a person's
	// ID token, it is accepted if its audience matches an entry here; the
	// token must still come from the configured issuer and verify against
	// its JWKS.
	TrustedAudiences []string

	// AllowPrivateURLs permits the identity provider (issuer and JWKS) to
	// resolve to private/internal IP addresses, e.g. dex.<mc>.<base> on a
	// private management cluster. TLS verification is still enforced.
	AllowPrivateURLs bool

	// DexCAFile is a PEM file with the CA that signed the Dex certificate.
	// Used for OIDC discovery, the token endpoint and JWKS. Empty means the
	// system trust store.
	DexCAFile string
}

// Supported values for [Config.Provider].
const (
	// ProviderDex uses a Dex OIDC issuer. Default.
	ProviderDex = "dex"
	// ProviderGoogle uses Google as the OIDC provider.
	ProviderGoogle = "google"
)

// StorageTypeValkey selects the Valkey token store.
const StorageTypeValkey = "valkey"

const envTrue = "true"

// ConfigFromEnv reads the configuration from the environment. The variable
// names match mcp-prometheus and mcp-kubernetes so one platform contract
// configures every MCP server the same way.
func ConfigFromEnv() Config {
	return configFromLookup(os.LookupEnv)
}

func configFromLookup(lookup func(string) (string, bool)) Config {
	get := func(k string) string {
		v, _ := lookup(k)
		return strings.TrimSpace(v)
	}
	cfg := Config{
		Issuer:                  get("MCP_OAUTH_ISSUER"),
		EncryptionKey:           get("MCP_OAUTH_ENCRYPTION_KEY"),
		AllowPublicRegistration: get("MCP_OAUTH_ALLOW_PUBLIC_REGISTRATION") == envTrue,
		StorageType:             get("OAUTH_STORAGE"),
		ValkeyURL:               get("VALKEY_URL"),
		ValkeyPassword:          get("VALKEY_PASSWORD"),
		ValkeyTLS:               get("VALKEY_TLS_ENABLED") == envTrue,
		ValkeyKeyPrefix:         get("VALKEY_KEY_PREFIX"),
		Provider:                strings.ToLower(get("MCP_OAUTH_PROVIDER")),
		RedirectURL:             get("OAUTH_REDIRECT_URL"),
		GoogleClientID:          get("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:      get("GOOGLE_CLIENT_SECRET"),
		DexIssuerURL:            get("DEX_ISSUER_URL"),
		DexClientID:             get("DEX_CLIENT_ID"),
		DexClientSecret:         get("DEX_CLIENT_SECRET"),
		KubernetesAudience:      get("DEX_K8S_AUTHENTICATOR_CLIENT_ID"),
		AllowPrivateURLs:        get("MCP_OAUTH_ALLOW_PRIVATE_URLS") == envTrue,
		DexCAFile:               get("DEX_CA_FILE"),
	}
	if cfg.Provider == "" {
		cfg.Provider = ProviderDex
	}
	if cfg.RedirectURL == "" {
		// Legacy name from the Dex-only days, still honoured.
		cfg.RedirectURL = get("DEX_REDIRECT_URL")
	}
	for a := range strings.SplitSeq(get("OAUTH_TRUSTED_AUDIENCES"), ",") {
		if a = strings.TrimSpace(a); a != "" {
			cfg.TrustedAudiences = append(cfg.TrustedAudiences, a)
		}
	}
	return cfg
}
