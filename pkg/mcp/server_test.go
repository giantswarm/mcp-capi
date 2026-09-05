package mcp

import (
	"strings"
	"testing"

	"github.com/giantswarm/mcp-capi/pkg/oauth"
)

func TestNewServerRejectsInvalidAuthCombinations(t *testing.T) {
	cases := map[string]struct {
		opts ServerOptions
		want string
	}{
		"caller identity without oauth": {
			opts: ServerOptions{Transport: TransportStreamableHTTP, CallerIdentity: true},
			want: "requires OAuth",
		},
		"oauth on stdio": {
			opts: ServerOptions{Transport: TransportStdio, OAuth: &oauth.Config{}},
			want: "stdio",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewServer(tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("NewServer() error = %v, want %q", err, tc.want)
			}
		})
	}
}
