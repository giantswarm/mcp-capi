package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// paginatedResult renders a `{ items, nextCursor }` JSON payload as the
// canonical MCP-toolkit paginated tool result. nextCursor is omitted when
// empty (no more pages). items must marshal to a JSON array — typically a
// slice of digest structs from package capi.
func paginatedResult(items any, nextCursor string) (*mcp.CallToolResult, error) {
	payload := struct {
		Items      any        `json:"items"`
		NextCursor mcp.Cursor `json:"nextCursor,omitempty"`
	}{
		Items:      items,
		NextCursor: mcp.Cursor(nextCursor),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal paginated result: %w", err)
	}
	return mcp.NewToolResultText(string(body)), nil
}

// jsonResult renders any value as a JSON tool result. Used for non-paginated
// list tools (bounded enums) and for tools whose response is a single object.
func jsonResult(v any) (*mcp.CallToolResult, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return mcp.NewToolResultText(string(body)), nil
}

// pageLimit clamps a caller-supplied limit into a sensible range.
// Zero / negative → defaultLimit. Values above maxLimit → maxLimit. The
// returned int64 is what controller-runtime's client.Limit expects.
func pageLimit(raw float64, defaultLimit, maxLimit int64) int64 {
	if raw <= 0 {
		return defaultLimit
	}
	n := int64(raw)
	if n > maxLimit {
		return maxLimit
	}
	return n
}
