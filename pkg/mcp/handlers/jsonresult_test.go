package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestPaginatedResult_OmitsEmptyCursor(t *testing.T) {
	res, err := paginatedResult([]string{"a", "b"}, "")
	if err != nil {
		t.Fatalf("paginatedResult: %v", err)
	}
	body := textBody(t, res)

	var got map[string]any
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("payload not JSON: %v\n%s", err, body)
	}
	if _, hasCursor := got["nextCursor"]; hasCursor {
		t.Errorf("nextCursor should be omitted when empty, got: %s", body)
	}
	if items, ok := got["items"].([]any); !ok || len(items) != 2 {
		t.Errorf("items should be a 2-element array, got: %v", got["items"])
	}
}

func TestPaginatedResult_IncludesCursor(t *testing.T) {
	res, err := paginatedResult([]int{1, 2, 3}, "page-2-token")
	if err != nil {
		t.Fatalf("paginatedResult: %v", err)
	}
	body := textBody(t, res)
	if !strings.Contains(body, `"nextCursor":"page-2-token"`) {
		t.Errorf("nextCursor missing from payload: %s", body)
	}
}

func TestPageLimit_ClampsAndDefaults(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want int64
	}{
		{"zero applies default", 0, 50},
		{"negative applies default", -1, 50},
		{"in-range passes through", 25, 25},
		{"over max clamps to max", 9999, 200},
		{"max boundary", 200, 200},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := pageLimit(c.in, 50, 200)
			if got != c.want {
				t.Errorf("pageLimit(%v) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func textBody(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("expected 1 content entry, got %d", len(res.Content))
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	return tc.Text
}
