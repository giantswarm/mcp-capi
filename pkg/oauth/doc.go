// Package oauth turns mcp-capi into an OAuth 2.1 resource server backed by the
// github.com/giantswarm/mcp-oauth library and resolves the identity of the
// person behind every request.
//
// Two kinds of bearer reach the MCP endpoint:
//
//   - An ID token forwarded by an upstream aggregator (muster with
//     forwardToken: true). mcp-oauth validates it against the identity
//     provider's JWKS when its audience is listed in TrustedAudiences. The
//     bearer itself is the OIDC ID token.
//   - An access token issued by this server's own authorization server after
//     an interactive login. The provider token (with the ID token) is looked
//     up in the token store.
//
// Either way the ID token ends up in the request context
// ([ContextWithIDToken]) so the CAPI client can present it to the Kubernetes
// API server: the apiserver's OIDC authenticator resolves the person and the
// person's RBAC governs every CAPI operation. mcp-capi never acts with a
// ServiceAccount in this mode.
package oauth
