package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/giantswarm/mcp-capi/pkg/oauth"
)

func TestServerContextClientCallerIdentity(t *testing.T) {
	factory := capi.NewBearerClientFactory("https://kubernetes.example:6443", "", nil)
	sc := NewCallerIdentityServerContext(factory)
	if !sc.ActsAsCaller() {
		t.Fatal("expected caller-identity mode")
	}

	if _, err := sc.Client(context.Background()); !errors.Is(err, ErrNoCallerIdentity) {
		t.Fatalf("without an ID token the client must be refused, got %v", err)
	}

	ctx := oauth.ContextWithIDToken(context.Background(), "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJhbGljZSJ9.sig")
	c, err := sc.Client(ctx)
	if err != nil || c == nil {
		t.Fatalf("Client() = %v, %v", c, err)
	}
}

func TestServerContextClientStatic(t *testing.T) {
	sc := NewServerContext(nil)
	if sc.ActsAsCaller() {
		t.Fatal("static context must not act as caller")
	}
	if _, err := sc.Client(context.Background()); err == nil {
		t.Fatal("a nil static client must be reported")
	}
	static := &capi.Client{}
	sc = NewServerContext(static)
	c, err := sc.Client(context.Background())
	if err != nil || c != static {
		t.Fatalf("Client() = %v, %v; want the static client", c, err)
	}
}
