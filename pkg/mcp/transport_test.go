package mcp

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// A body beyond the bound is refused, and the endpoint keeps answering.
func TestStreamableHTTPBodyIsBounded(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- RunStreamableHTTPServer(mcpserver.NewMCPServer("test", "0"), addr, "/mcp", ctx, nil) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()

	post := func(body string) int {
		req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/mcp", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		var resp *http.Response
		for range 50 {
			if resp, err = http.DefaultClient.Do(req); err == nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
			req.Body, _ = req.GetBody()
		}
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}

	if got := post(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`); got != http.StatusOK {
		t.Errorf("initialize: status %d, want 200", got)
	}
	big := `{"jsonrpc":"2.0","id":2,"method":"ping","params":{"pad":"` + strings.Repeat("x", maxBodyBytes) + `"}}`
	if got := post(big); got < 400 {
		t.Errorf("oversized body: status %d, want an error", got)
	}
}
