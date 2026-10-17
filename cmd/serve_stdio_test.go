package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// serveHelperEnv makes the test binary run `mcp-capi serve` with the
// arguments after "--" instead of the tests, so the stdio transport is
// exercised on the process's real stdin and stdout, as a stdio client sees it.
const serveHelperEnv = "MCP_CAPI_SERVE_HELPER"

func TestMain(m *testing.M) {
	if os.Getenv(serveHelperEnv) == "1" {
		for i, arg := range os.Args {
			if arg == "--" {
				os.Args = append([]string{"mcp-capi"}, os.Args[i+1:]...)
				break
			}
		}
		Execute()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// unreachableKubeconfig points at a closed port: the stdio handshake and
// tools/list never call the Kubernetes API.
const unreachableKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://127.0.0.1:1
users:
- name: test
  user:
    token: test
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
`

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

// TestServeStdio drives `mcp-capi serve` over stdio with initialize and
// tools/list and requires a JSON-RPC answer to both, with nothing but
// JSON-RPC messages on stdout.
func TestServeStdio(t *testing.T) {
	kubeconfig := filepath.Join(t.TempDir(), "kubeconfig.yaml")
	if err := os.WriteFile(kubeconfig, []byte(unreachableKubeconfig), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, self, "-test.run=^$", "--", "serve", "--kubeconfig", kubeconfig) //#nosec G204 -- test re-executes its own binary
	cmd.Env = append(os.Environ(), serveHelperEnv+"=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	requests := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	}
	for _, r := range requests {
		if _, err := io.WriteString(stdin, r+"\n"); err != nil {
			t.Fatalf("write %s: %v", r, err)
		}
	}

	responses := map[int]rpcResponse{}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1<<20), 16<<20)
	for len(responses) < 2 && scanner.Scan() {
		var resp rpcResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil || resp.JSONRPC != "2.0" {
			t.Fatalf("stdout carries a line that is not a JSON-RPC message: %q", scanner.Text())
		}
		if resp.ID != nil {
			responses[*resp.ID] = resp
		}
	}
	_ = stdin.Close()
	waitErr := cmd.Wait()
	if len(responses) < 2 {
		t.Fatalf("got %d of 2 responses (scan: %v, exit: %v); stderr:\n%s", len(responses), scanner.Err(), waitErr, stderr.String())
	}
	if waitErr != nil {
		t.Fatalf("serve exited with %v on stdin EOF; stderr:\n%s", waitErr, stderr.String())
	}

	var initResult struct {
		ServerInfo struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
	}
	if err := decodeResult(responses[1], &initResult); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if initResult.ServerInfo.Name != serverName {
		t.Errorf("initialize serverInfo.name = %q, want %q", initResult.ServerInfo.Name, serverName)
	}

	var toolsResult struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := decodeResult(responses[2], &toolsResult); err != nil {
		t.Fatalf("tools/list: %v", err)
	}
	if len(toolsResult.Tools) == 0 {
		t.Error("tools/list returned no tools")
	}
}

func decodeResult(resp rpcResponse, v any) error {
	if len(resp.Error) > 0 {
		return fmt.Errorf("error response: %s", resp.Error)
	}
	return json.Unmarshal(resp.Result, v)
}
