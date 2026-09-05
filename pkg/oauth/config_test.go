package oauth

import (
	"reflect"
	"testing"
)

const k8sAudience = "dex-k8s-authenticator"

func TestConfigFromLookup(t *testing.T) {
	env := map[string]string{
		"MCP_OAUTH_ISSUER":                "https://mcp-capi.example.com",
		"OAUTH_REDIRECT_URL":              "https://mcp-capi.example.com/oauth/callback",
		"DEX_ISSUER_URL":                  "https://dex.example.com",
		"DEX_CLIENT_ID":                   "mcp-capi",
		"DEX_CLIENT_SECRET":               "s3cr3t",
		"DEX_K8S_AUTHENTICATOR_CLIENT_ID": k8sAudience,
		"OAUTH_TRUSTED_AUDIENCES":         " muster-client, dex-k8s-authenticator ,,",
		"MCP_OAUTH_ALLOW_PRIVATE_URLS":    "true",
		"OAUTH_STORAGE":                   "valkey",
		"VALKEY_URL":                      "valkey:6379",
		"MCP_OAUTH_PROVIDER":              "Dex",
	}
	cfg := configFromLookup(func(k string) (string, bool) { v, ok := env[k]; return v, ok })

	if cfg.Issuer != env["MCP_OAUTH_ISSUER"] || cfg.RedirectURL != env["OAUTH_REDIRECT_URL"] {
		t.Fatalf("issuer/redirect not read: %+v", cfg)
	}
	if cfg.Provider != ProviderDex {
		t.Fatalf("provider should be lower-cased dex, got %q", cfg.Provider)
	}
	if want := []string{"muster-client", k8sAudience}; !reflect.DeepEqual(cfg.TrustedAudiences, want) {
		t.Fatalf("trusted audiences = %v, want %v", cfg.TrustedAudiences, want)
	}
	if !cfg.AllowPrivateURLs || cfg.StorageType != StorageTypeValkey || cfg.ValkeyURL != "valkey:6379" {
		t.Fatalf("flags not read: %+v", cfg)
	}
	if cfg.KubernetesAudience != k8sAudience || cfg.DexClientSecret != "s3cr3t" {
		t.Fatalf("dex settings not read: %+v", cfg)
	}
}

func TestConfigFromLookupDefaultsAndLegacyRedirect(t *testing.T) {
	env := map[string]string{"DEX_REDIRECT_URL": "https://legacy/oauth/callback"}
	cfg := configFromLookup(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
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
