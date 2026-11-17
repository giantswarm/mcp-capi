package oauth

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	k8sAudience      = "dex-k8s-authenticator"
	testIssuer       = "https://mcp-capi.example.com"
	testRedirectURL  = testIssuer + "/oauth/callback"
	testClientID     = "client"
	testClientSecret = "secret"
	envFalse         = "false"
)

func TestConfigFromLookup(t *testing.T) {
	env := map[string]string{
		"MCP_OAUTH_ISSUER":                testIssuer,
		"OAUTH_REDIRECT_URL":              testRedirectURL,
		"DEX_ISSUER_URL":                  "https://dex.example.com",
		"DEX_CLIENT_ID":                   "mcp-capi",
		"DEX_CLIENT_SECRET":               "s3cr3t",
		"DEX_K8S_AUTHENTICATOR_CLIENT_ID": k8sAudience,
		"OAUTH_TRUSTED_AUDIENCES":         " muster-client, dex-k8s-authenticator ,,",
		"MCP_OAUTH_ALLOW_PRIVATE_IP_JWKS": envTrue,
		"OAUTH_STORAGE":                   "valkey",
		"VALKEY_URL":                      "valkey:6379",
		"MCP_OAUTH_PROVIDER":              "Dex",
	}
	cfg, err := configFromLookup(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	require.NoError(t, err)

	if cfg.Issuer != env["MCP_OAUTH_ISSUER"] || cfg.RedirectURL != env["OAUTH_REDIRECT_URL"] {
		t.Fatalf("issuer/redirect not read: %+v", cfg)
	}
	if cfg.Provider != ProviderDex {
		t.Fatalf("provider should be lower-cased dex, got %q", cfg.Provider)
	}
	if want := []string{"muster-client", k8sAudience}; !reflect.DeepEqual(cfg.TrustedAudiences, want) {
		t.Fatalf("trusted audiences = %v, want %v", cfg.TrustedAudiences, want)
	}
	if !cfg.AllowPrivateIPJWKS || cfg.DeprecatedPrivateURLsEnv != "" || cfg.StorageType != StorageTypeValkey || cfg.ValkeyURL != "valkey:6379" {
		t.Fatalf("flags not read: %+v", cfg)
	}
	if cfg.KubernetesAudience != k8sAudience || cfg.DexClientSecret != "s3cr3t" {
		t.Fatalf("dex settings not read: %+v", cfg)
	}
}

func TestConfigFromLookupDefaultsAndLegacyRedirect(t *testing.T) {
	env := map[string]string{"DEX_REDIRECT_URL": "https://legacy/oauth/callback"}
	cfg, err := configFromLookup(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	require.NoError(t, err)
	if cfg.Provider != ProviderDex {
		t.Fatalf("default provider = %q, want dex", cfg.Provider)
	}
	if cfg.RedirectURL != "https://legacy/oauth/callback" {
		t.Fatalf("legacy DEX_REDIRECT_URL not honoured: %q", cfg.RedirectURL)
	}
	if len(cfg.TrustedAudiences) != 0 {
		t.Fatalf("expected no trusted audiences, got %v", cfg.TrustedAudiences)
	}
}

func TestConfigFromEnvPrivateIPJWKSAlias(t *testing.T) {
	tests := []struct {
		name           string
		canonical      string
		deprecated     string
		want           bool
		wantDeprecated string
		wantError      string
	}{
		{name: "unset is off"},
		{name: "new true", canonical: envTrue, want: true},
		{name: "new 1", canonical: "1", want: true},
		{name: "new false", canonical: envFalse},
		{name: "new invalid", canonical: "yes", wantError: envAllowPrivateIPJWKS},
		{name: "old true", deprecated: envTrue, want: true, wantDeprecated: envAllowPrivateURLs},
		{name: "old false", deprecated: envFalse, wantDeprecated: envAllowPrivateURLs},
		{name: "old invalid", deprecated: "yes", wantError: envAllowPrivateURLs},
		{name: "old true wins over new false", canonical: envFalse, deprecated: envTrue, want: true, wantDeprecated: envAllowPrivateURLs},
		{name: "new true wins over old false", canonical: envTrue, deprecated: envFalse, want: true, wantDeprecated: envAllowPrivateURLs},
		{name: "old and new true", canonical: envTrue, deprecated: envTrue, want: true, wantDeprecated: envAllowPrivateURLs},
		{name: "old invalid with new true", canonical: envTrue, deprecated: "nope", wantError: envAllowPrivateURLs},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv(envAllowPrivateIPJWKS, testCase.canonical)
			t.Setenv(envAllowPrivateURLs, testCase.deprecated)
			cfg, err := ConfigFromEnv()
			if testCase.wantError != "" {
				require.ErrorContains(t, err, testCase.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.want, cfg.AllowPrivateIPJWKS)
			require.Equal(t, testCase.wantDeprecated, cfg.DeprecatedPrivateURLsEnv)
		})
	}
}
