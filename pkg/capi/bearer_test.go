package capi

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func jwtWithExp(t *testing.T, exp int64) string {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"sub": "alice", "exp": exp})
	if err != nil {
		t.Fatal(err)
	}
	return "eyJhbGciOiJSUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}

func TestTokenExpiry(t *testing.T) {
	exp := time.Now().Add(time.Hour).Unix()
	got, ok := tokenExpiry(jwtWithExp(t, exp))
	if !ok || got.Unix() != exp {
		t.Fatalf("tokenExpiry = %v %v, want %d", got, ok, exp)
	}
	if _, ok := tokenExpiry("opaque-token"); ok {
		t.Fatal("opaque token must not yield an expiry")
	}
	if _, ok := tokenExpiry("a.b.c"); ok {
		t.Fatal("undecodable payload must not yield an expiry")
	}
}

func TestClientExpiry(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if got := clientExpiry("opaque", now); got != now.Add(defaultClientTTL) {
		t.Fatalf("opaque: %v", got)
	}
	soon := now.Add(2 * time.Minute)
	if got := clientExpiry(jwtWithExp(t, soon.Unix()), now); !got.Equal(soon) {
		t.Fatalf("short-lived token must cap at exp, got %v", got)
	}
	if got := clientExpiry(jwtWithExp(t, now.Add(24*time.Hour).Unix()), now); got != now.Add(maxClientTTL) {
		t.Fatalf("long-lived token must cap at maxClientTTL, got %v", got)
	}
}

func TestBearerClientFactoryClientFor(t *testing.T) {
	f := NewBearerClientFactory("https://kubernetes.example:6443", "", nil)
	if _, err := f.ClientFor(""); err != ErrNoBearerToken {
		t.Fatalf("empty token: err = %v, want ErrNoBearerToken", err)
	}

	alice := jwtWithExp(t, time.Now().Add(time.Hour).Unix())
	bob := jwtWithExp(t, time.Now().Add(2*time.Hour).Unix())

	c1, err := f.ClientFor(alice)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := f.ClientFor(alice)
	if err != nil {
		t.Fatal(err)
	}
	if c1 != c2 {
		t.Fatal("same token must reuse the cached client")
	}
	c3, err := f.ClientFor(bob)
	if err != nil {
		t.Fatal(err)
	}
	if c3 == c1 {
		t.Fatal("different tokens must get different clients")
	}
	if got := c1.config.BearerToken; got != alice {
		t.Fatalf("client authenticates with %q, want the caller's token", got)
	}
	if c1.config.BearerTokenFile != "" || c1.config.Impersonate.UserName != "" {
		t.Fatalf("client must carry no ServiceAccount token file or impersonation: %+v", c1.config)
	}
	if c1.config.Host != f.Host() {
		t.Fatalf("host = %q", c1.config.Host)
	}
}

func TestBearerClientFactoryCacheExpires(t *testing.T) {
	f := NewBearerClientFactory("https://kubernetes.example:6443", "", nil)
	now := time.Now()
	f.now = func() time.Time { return now }
	tok := jwtWithExp(t, now.Add(time.Minute).Unix())
	c1, _ := f.ClientFor(tok)
	now = now.Add(2 * time.Minute)
	c2, _ := f.ClientFor(tok)
	if c1 == c2 {
		t.Fatal("expired cache entry must not be reused")
	}
}

func TestNewBearerClientFactoryFromKubeconfig(t *testing.T) {
	dir := t.TempDir()
	ca := filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(ca, []byte("placeholder"), 0o600); err != nil {
		t.Fatal(err)
	}
	kubeconfig := filepath.Join(dir, "kubeconfig")
	content := `apiVersion: v1
kind: Config
clusters:
- name: c
  cluster:
    server: https://kubernetes.example:6443
    certificate-authority: ` + ca + `
users:
- name: u
  user:
    token: static-admin-token
contexts:
- name: c
  context: {cluster: c, user: u}
current-context: c
`
	if err := os.WriteFile(kubeconfig, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := NewBearerClientFactoryFromKubeconfig(kubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	if f.Host() != "https://kubernetes.example:6443" {
		t.Fatalf("host = %q", f.Host())
	}
	cfg := f.restConfig("caller-token")
	if cfg.BearerToken != "caller-token" {
		t.Fatalf("the kubeconfig's credentials must be discarded, got %q", cfg.BearerToken)
	}
	if cfg.CAFile != ca {
		t.Fatalf("CA file not taken from kubeconfig: %q", cfg.CAFile)
	}
}

func TestNewInClusterBearerClientFactoryOutsideCluster(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
	if _, err := NewInClusterBearerClientFactory(); err == nil {
		t.Fatal("expected an error outside a cluster")
	}
}
