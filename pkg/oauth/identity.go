package oauth

import (
	"context"
	"net/http"
	"strings"

	"github.com/giantswarm/mcp-oauth/handler"
	"github.com/giantswarm/mcp-oauth/providers"
)

type contextKey string

//nolint:gosec // G101: a context key name, not a credential
const idTokenKey contextKey = "mcp-capi/oauth-id-token"

// UserInfo is the authenticated caller as resolved by mcp-oauth.
type UserInfo = providers.UserInfo

// ContextWithIDToken stores the caller's OIDC ID token.
func ContextWithIDToken(ctx context.Context, idToken string) context.Context {
	return context.WithValue(ctx, idTokenKey, idToken)
}

// IDTokenFromContext returns the caller's OIDC ID token, if the request was
// authenticated and an ID token could be resolved for it.
func IDTokenFromContext(ctx context.Context) (string, bool) {
	tok, ok := ctx.Value(idTokenKey).(string)
	return tok, ok && tok != ""
}

// UserInfoFromContext returns the authenticated caller, set by the token
// validation middleware.
func UserInfoFromContext(ctx context.Context) (*UserInfo, bool) {
	u, ok := handler.UserInfoFromContext(ctx)
	return u, ok && u != nil
}

// HTTPContextFunc copies the caller's identity from the HTTP request context
// into the context mcp-go hands to tool handlers. mcp-go derives that context
// from the request already; this keeps the propagation explicit and robust
// against transports that do not.
func HTTPContextFunc(ctx context.Context, r *http.Request) context.Context {
	if u, ok := UserInfoFromContext(r.Context()); ok {
		ctx = handler.ContextWithUserInfo(ctx, u)
	}
	if tok, ok := IDTokenFromContext(r.Context()); ok {
		ctx = ContextWithIDToken(ctx, tok)
	}
	return ctx
}

// identity runs after ValidateToken and resolves the OIDC ID token that
// represents the caller towards the Kubernetes API server:
//
//   - a forwarded ID token (single sign-on through an aggregator such as
//     muster) IS the bearer;
//   - an access token issued by this server maps to the provider token stored
//     at login, whose id_token is used.
//
// The request proceeds even when no ID token could be resolved: protocol
// handshakes (initialize, tools/list) need no cluster access. Tool calls fail
// closed in handlers.ServerContext.Client when the context has no token.
func (s *Server) identity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, ok := UserInfoFromContext(ctx)
		if !ok {
			// ValidateToken always sets it; treat its absence as unauthenticated.
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		bearer := bearerToken(r)
		idToken := ""
		switch {
		case user.IsExternalIssuer():
			// Trusted-issuer tokens are not configured for mcp-capi; a token
			// of that class carries no identity the apiserver would accept.
			s.logger.Warn("rejecting external-issuer token: not supported by mcp-capi", "issuer", user.Issuer)
			http.Error(w, "forbidden: external-issuer tokens are not supported", http.StatusForbidden)
			return
		case user.IsSSO():
			idToken = bearer
			s.logger.Info("forwarded_id_token_accepted", "email", user.Email, "issuer", user.Issuer)
		default:
			idToken = s.storedIDToken(ctx, bearer, user)
			if idToken == "" {
				s.logger.Warn("no id_token stored for this session; Kubernetes calls will be refused", "email", user.Email)
			}
		}
		if idToken != "" {
			ctx = ContextWithIDToken(ctx, idToken)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// storedIDToken looks up the provider token this server stored at login. The
// store is keyed by the access token; older records were keyed by e-mail.
func (s *Server) storedIDToken(ctx context.Context, accessToken string, user *UserInfo) string {
	if s.server == nil {
		return ""
	}
	store := s.server.TokenStore()
	if store == nil {
		return ""
	}
	tok, err := store.GetToken(ctx, accessToken)
	if (err != nil || tok == nil) && user.Email != "" {
		tok, err = store.GetToken(ctx, user.Email)
	}
	if err != nil || tok == nil {
		return ""
	}
	if id, ok := tok.Extra("id_token").(string); ok {
		return id
	}
	return ""
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}
