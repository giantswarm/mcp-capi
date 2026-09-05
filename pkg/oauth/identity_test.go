package oauth

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/giantswarm/mcp-oauth/handler"
	"github.com/giantswarm/mcp-oauth/providers"
)

const alice = "alice@example.com"

// withUser simulates ValidateToken having stamped the caller on the request.
func withUser(r *http.Request, u *providers.UserInfo) *http.Request {
	return r.WithContext(handler.ContextWithUserInfo(r.Context(), u))
}

func TestIdentityMiddleware(t *testing.T) {
	srv := &Server{logger: slog.New(slog.NewTextHandler(discard{}, nil))}
	const forwarded = "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJhbGljZSJ9.sig"

	var seenToken string
	var seenOK bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenToken, seenOK = IDTokenFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	h := srv.identity(next)

	t.Run("forwarded ID token is the bearer", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		r.Header.Set("Authorization", "Bearer "+forwarded)
		r = withUser(r, &providers.UserInfo{Email: alice, TokenSource: providers.TokenSourceSSO, Issuer: "https://dex"})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusNoContent || !seenOK || seenToken != forwarded {
			t.Fatalf("status=%d token=%q ok=%v", rec.Code, seenToken, seenOK)
		}
	})

	t.Run("no user info is unauthorized", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("external-issuer token is forbidden", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		r.Header.Set("Authorization", "Bearer x")
		r = withUser(r, &providers.UserInfo{Email: "svc@example.com", TokenSource: providers.TokenSourceTrustedIssuer})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("own access token without a stored id_token proceeds without identity", func(t *testing.T) {
		seenToken, seenOK = "", true
		r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		r.Header.Set("Authorization", "Bearer opaque")
		r = withUser(r, &providers.UserInfo{Email: "bob@example.com", TokenSource: providers.TokenSourceOAuth})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusNoContent || seenOK {
			t.Fatalf("status=%d ok=%v: handshake must pass, but no ID token may be attached", rec.Code, seenOK)
		}
	})
}

func TestHTTPContextFunc(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	ctx := ContextWithIDToken(r.Context(), "tok")
	ctx = handler.ContextWithUserInfo(ctx, &providers.UserInfo{Email: alice})
	r = r.WithContext(ctx)

	got := HTTPContextFunc(context.Background(), r)
	if tok, ok := IDTokenFromContext(got); !ok || tok != "tok" {
		t.Fatalf("ID token not propagated: %q %v", tok, ok)
	}
	if u, ok := UserInfoFromContext(got); !ok || u.Email != alice {
		t.Fatalf("user info not propagated: %+v %v", u, ok)
	}
}
