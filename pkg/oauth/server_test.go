package oauth

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewServerValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	base := Config{Issuer: "https://mcp-capi.example.com", RedirectURL: testRedirectURL}

	cases := map[string]struct {
		mutate func(*Config)
		want   string
	}{
		"missing issuer":       {func(c *Config) { c.Issuer = "" }, "MCP_OAUTH_ISSUER"},
		"missing redirect":     {func(c *Config) { c.RedirectURL = "" }, "OAUTH_REDIRECT_URL"},
		"unknown provider":     {func(c *Config) { c.Provider = "okta" }, "unsupported MCP_OAUTH_PROVIDER"},
		"dex without client":   {func(c *Config) { c.Provider = ProviderDex; c.DexIssuerURL = "https://dex" }, "DEX_CLIENT_SECRET"},
		"google without creds": {func(c *Config) { c.Provider = ProviderGoogle }, "GOOGLE_CLIENT_ID"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := base
			tc.mutate(&cfg)
			_, err := NewServer(t.Context(), cfg, logger)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("NewServer() error = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestDecodeKey(t *testing.T) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	for name, encoded := range map[string]string{
		"hex":    hex.EncodeToString(raw),
		"base64": base64.StdEncoding.EncodeToString(raw),
	} {
		got, err := decodeKey(encoded)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if string(got) != string(raw) {
			t.Fatalf("%s: decoded key differs", name)
		}
	}
	if _, err := decodeKey("not*base64*at*all"); err == nil {
		t.Fatal("expected an error for an undecodable key")
	}
}

func TestBuildEncryptorRejectsShortKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	if _, err := buildEncryptor(Config{EncryptionKey: base64.StdEncoding.EncodeToString([]byte("short"))}, logger); err == nil {
		t.Fatal("expected an error for a key shorter than 32 bytes")
	}
	enc, err := buildEncryptor(Config{}, logger)
	if err != nil || enc != nil {
		t.Fatalf("no key must yield (nil, nil), got (%v, %v)", enc, err)
	}
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

func TestServerConfigPrivateIPJWKSHosts(t *testing.T) {
	const issuer = "https://dex.mc.example.internal:8443/dex"
	tests := []struct {
		name string
		cfg  Config
		want []string
	}{
		{name: "dex, flag off keeps the guard", cfg: Config{Provider: ProviderDex, DexIssuerURL: issuer}},
		{name: "dex, flag on allows the issuer host only", cfg: Config{Provider: ProviderDex, DexIssuerURL: issuer, AllowPrivateIPJWKS: true}, want: []string{"dex.mc.example.internal"}},
		{name: "default provider is dex", cfg: Config{DexIssuerURL: issuer, AllowPrivateIPJWKS: true}, want: []string{"dex.mc.example.internal"}},
		{name: "google ignores the flag", cfg: Config{Provider: ProviderGoogle, DexIssuerURL: issuer, AllowPrivateIPJWKS: true}},
		{name: "no issuer host", cfg: Config{Provider: ProviderDex, AllowPrivateIPJWKS: true}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			serverCfg := serverConfig(testCase.cfg, nil)
			require.Equal(t, testCase.want, serverCfg.AllowPrivateIPJWKSHosts)
			require.False(t, serverCfg.AllowPrivateIPJWKS) //nolint:staticcheck // The deprecated blanket switch must stay off.
		})
	}
}

func TestDexProviderConfigLeavesClientToMCPOAuth(t *testing.T) {
	rootCAs := x509.NewCertPool()
	dexCfg := dexProviderConfig(Config{
		DexIssuerURL:       "https://dex.example.com",
		DexClientID:        "client",
		DexClientSecret:    "secret",
		RedirectURL:        testRedirectURL,
		KubernetesAudience: k8sAudience,
		AllowPrivateIPJWKS: true,
	}, rootCAs)
	require.Nil(t, dexCfg.HTTPClient)
	require.Same(t, rootCAs, dexCfg.RootCAs)
	require.Contains(t, dexCfg.Scopes, "audience:server:client_id:"+k8sAudience)
}
