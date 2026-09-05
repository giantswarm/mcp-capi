package capi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// DefaultInClusterCAFile is where the pod's projected kube-root-ca.crt is
	// mounted. Only the CA is needed: the caller's token, never a
	// ServiceAccount token, authenticates the request.
	DefaultInClusterCAFile = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	defaultClientTTL  = 5 * time.Minute
	maxClientTTL      = 10 * time.Minute
	defaultMaxClients = 1000
)

// ErrNoBearerToken is returned when a client is requested without a token.
var ErrNoBearerToken = errors.New("bearer token is required")

// BearerClientFactory builds a [Client] per caller that authenticates to the
// Kubernetes API server with the caller's own OIDC ID token. The pod's
// ServiceAccount is never used: the factory only knows the API server
// address and its CA.
//
// Clients are cached by token digest until the token expires (or for a short
// TTL when the token carries no exp claim) so a chatty session does not pay
// for discovery on every call.
type BearerClientFactory struct {
	host    string
	caFile  string
	caData  []byte
	qps     float32
	burst   int
	timeout time.Duration

	mu         sync.Mutex
	cache      map[string]cacheEntry
	maxEntries int
	now        func() time.Time
}

type cacheEntry struct {
	client  *Client
	expires time.Time
}

// NewBearerClientFactory creates a factory for the API server at host. caFile
// or caData carry the server CA (either may be empty to use the system pool).
func NewBearerClientFactory(host, caFile string, caData []byte) *BearerClientFactory {
	return &BearerClientFactory{
		host:       host,
		caFile:     caFile,
		caData:     caData,
		qps:        20,
		burst:      30,
		timeout:    30 * time.Second,
		cache:      map[string]cacheEntry{},
		maxEntries: defaultMaxClients,
		now:        time.Now,
	}
}

// NewInClusterBearerClientFactory derives the API server address from the
// pod's KUBERNETES_SERVICE_HOST/PORT environment and the CA from
// [DefaultInClusterCAFile]. Unlike rest.InClusterConfig it does not require a
// ServiceAccount token to be mounted.
func NewInClusterBearerClientFactory() (*BearerClientFactory, error) {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, errors.New("not running in a cluster: KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT are unset")
	}
	if _, err := os.Stat(DefaultInClusterCAFile); err != nil {
		return nil, fmt.Errorf("cluster CA %s: %w", DefaultInClusterCAFile, err)
	}
	return NewBearerClientFactory("https://"+net.JoinHostPort(host, port), DefaultInClusterCAFile, nil), nil
}

// NewBearerClientFactoryFromKubeconfig takes the API server address and CA
// from a kubeconfig and discards the credentials it contains. Useful outside
// a cluster (development, tests): the caller's token still authenticates.
func NewBearerClientFactoryFromKubeconfig(path string) (*BearerClientFactory, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig %s: %w", path, err)
	}
	return NewBearerClientFactory(cfg.Host, cfg.CAFile, cfg.CAData), nil
}

// Host returns the API server address the factory dials.
func (f *BearerClientFactory) Host() string { return f.host }

// ClientFor returns a client that authenticates as the bearer of token.
func (f *BearerClientFactory) ClientFor(token string) (*Client, error) {
	if token == "" {
		return nil, ErrNoBearerToken
	}
	key := digest(token)
	now := f.now()

	f.mu.Lock()
	if e, ok := f.cache[key]; ok && now.Before(e.expires) {
		f.mu.Unlock()
		return e.client, nil
	}
	f.mu.Unlock()

	c, err := NewClientFromConfig(f.restConfig(token))
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.cache) >= f.maxEntries {
		for k, e := range f.cache {
			if !now.Before(e.expires) {
				delete(f.cache, k)
			}
		}
	}
	if len(f.cache) < f.maxEntries {
		f.cache[key] = cacheEntry{client: c, expires: clientExpiry(token, now)}
	}
	return c, nil
}

func (f *BearerClientFactory) restConfig(token string) *rest.Config {
	return &rest.Config{
		Host:        f.host,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: f.caFile,
			CAData: f.caData,
		},
		QPS:     f.qps,
		Burst:   f.burst,
		Timeout: f.timeout,
	}
}

// clientExpiry caps the cache lifetime at the token's exp claim and at
// maxClientTTL, falling back to defaultClientTTL for opaque tokens.
func clientExpiry(token string, now time.Time) time.Time {
	limit := now.Add(maxClientTTL)
	exp, ok := tokenExpiry(token)
	if !ok {
		return now.Add(defaultClientTTL)
	}
	if exp.Before(limit) {
		return exp
	}
	return limit
}

// tokenExpiry reads the exp claim of a JWT without verifying it; the token
// was already verified by the OAuth layer, this only sizes the cache entry.
func tokenExpiry(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}

func digest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
