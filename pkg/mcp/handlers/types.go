package handlers

import (
	"context"
	"errors"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/giantswarm/mcp-capi/pkg/oauth"
)

const (
	textContentType = "text"
	infraAPIV1Beta1 = "infrastructure.cluster.x-k8s.io/v1beta1"
)

// ErrNoCallerIdentity is returned by ServerContext.Client when the server acts
// as the caller but the request carries no OIDC ID token. There is no
// fallback to a ServiceAccount: without a person, no cluster access.
var ErrNoCallerIdentity = errors.New("authentication required: no identity token for this session (the MCP server acts as the caller and holds no credentials of its own)")

// ServerContext holds shared resources for the MCP server handlers and hands
// out the CAPI client to use for a request.
//
// Two modes exist:
//
//   - a static client (kubeconfig or in-cluster ServiceAccount), the
//     original behaviour for local/stdio use;
//   - caller identity: every request gets a client that authenticates with
//     the caller's own OIDC ID token (see oauth.IDTokenFromContext), so the
//     Kubernetes API server applies the person's RBAC.
type ServerContext struct {
	capiClient *capi.Client
	factory    *capi.BearerClientFactory
}

// NewServerContext creates a ServerContext around one static CAPI client.
func NewServerContext(capiClient *capi.Client) *ServerContext {
	return &ServerContext{capiClient: capiClient}
}

// NewCallerIdentityServerContext creates a ServerContext that acts as the
// authenticated caller of each request.
func NewCallerIdentityServerContext(factory *capi.BearerClientFactory) *ServerContext {
	return &ServerContext{factory: factory}
}

// Client returns the CAPI client for this request.
func (sc *ServerContext) Client(ctx context.Context) (*capi.Client, error) {
	if sc.factory != nil {
		token, ok := oauth.IDTokenFromContext(ctx)
		if !ok {
			return nil, ErrNoCallerIdentity
		}
		return sc.factory.ClientFor(token)
	}
	if sc.capiClient == nil {
		return nil, errors.New("no CAPI client configured")
	}
	return sc.capiClient, nil
}

// ActsAsCaller reports whether requests are served with the caller's identity.
func (sc *ServerContext) ActsAsCaller() bool {
	return sc.factory != nil
}

// ToolRegistration holds a tool definition and its handler
type ToolRegistration struct {
	Tool    mcp.Tool
	Handler server.ToolHandlerFunc
}
