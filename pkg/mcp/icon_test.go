package mcp

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/giantswarm/mcp-toolkit/brand"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestInitializeDeclaresClusterAPIIcon(t *testing.T) {
	srv := newMCPServer("mcp-capi", "test")
	initReq := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`)

	raw, err := json.Marshal(srv.HandleMessage(t.Context(), initReq))
	if err != nil {
		t.Fatalf("marshal initialize response: %v", err)
	}
	var resp struct {
		Result mcp.InitializeResult `json:"result"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal initialize response: %v", err)
	}

	want := []mcp.Icon{brand.Icon(clusterAPILogo)}
	if got := resp.Result.ServerInfo.Icons; !reflect.DeepEqual(got, want) {
		t.Fatalf("serverInfo.icons = %+v, want %+v", got, want)
	}
}
