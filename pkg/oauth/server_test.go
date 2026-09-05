package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"log/slog"
	"strings"
	"testing"
)

func TestNewServerValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(discard{}, nil))
	base := Config{Issuer: "https://mcp-capi.example.com", RedirectURL: "https://mcp-capi.example.com/oauth/callback"}

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
			_, err := NewServer(context.Background(), cfg, logger)
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
