package handlers

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration

	"github.com/giantswarm/mcp-capi/pkg/capi"
)

// resultText returns the text content of a tool result.
func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result.IsError {
		t.Fatalf("unexpected error result: %+v", result.Content)
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected one content item, got %d", len(result.Content))
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", result.Content[0])
	}
	return text.Text
}

func TestJSONResultIsTextOnly(t *testing.T) {
	result, err := jsonResult(map[string]any{"b": 1, "a": "x"})
	if err != nil {
		t.Fatalf("jsonResult: %v", err)
	}
	if result.StructuredContent != nil {
		t.Fatalf("structuredContent must stay unset, got %v", result.StructuredContent)
	}
	if got, want := resultText(t, result), `{"a":"x","b":1}`; got != want {
		t.Fatalf("text = %s, want %s", got, want)
	}
}

func TestListResultEncodesEmptySliceAsArray(t *testing.T) {
	result, err := listResult(make([]ClusterSummary, 0))
	if err != nil {
		t.Fatalf("listResult: %v", err)
	}
	if got, want := resultText(t, result), `{"items":[]}`; got != want {
		t.Fatalf("text = %s, want %s", got, want)
	}
}

func TestGetClusterResultInlinesSummaryOrCandidates(t *testing.T) {
	summary := ClusterSummary{Name: "c1", Namespace: "ns", Provider: "aws"}

	single, err := json.Marshal(getClusterResult{ClusterSummary: &summary})
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]any
	if err := json.Unmarshal(single, &top); err != nil {
		t.Fatal(err)
	}
	if top["name"] != "c1" || top["namespace"] != "ns" {
		t.Fatalf("summary fields must be inlined at the top level, got %s", single)
	}
	if _, ok := top["candidates"]; ok {
		t.Fatalf("candidates must be omitted for a single match, got %s", single)
	}

	multi, err := json.Marshal(getClusterResult{MatchedBy: "labelValue", Candidates: []ClusterSummary{summary}})
	if err != nil {
		t.Fatal(err)
	}
	top = map[string]any{}
	if err := json.Unmarshal(multi, &top); err != nil {
		t.Fatal(err)
	}
	if _, ok := top["name"]; ok {
		t.Fatalf("no cluster fields expected at the top level for multiple matches, got %s", multi)
	}
	if candidates, ok := top["candidates"].([]any); !ok || len(candidates) != 1 {
		t.Fatalf("expected one candidate, got %s", multi)
	}
}

func TestClusterSummaryDropsSystemLabelsAndKeepsConditions(t *testing.T) {
	status := &capi.ClusterStatus{
		Name:      "c1",
		Namespace: "ns",
		Provider:  capi.ProviderAWS,
		Labels: map[string]string{
			"cluster.x-k8s.io/cluster-name": "c1",
			"env":                           "prod",
		},
		Conditions: clusterv1.Conditions{
			{Type: clusterv1.ReadyCondition, Status: "True", Reason: "ClusterReady", Message: "all good"},
		},
	}

	got := clusterSummary(status)
	if _, ok := got.Labels["cluster.x-k8s.io/cluster-name"]; ok {
		t.Fatalf("system labels must be dropped, got %v", got.Labels)
	}
	if got.Labels["env"] != "prod" {
		t.Fatalf("user labels must be kept, got %v", got.Labels)
	}
	if len(got.Conditions) != 1 || got.Conditions[0].Message != "all good" {
		t.Fatalf("conditions must carry reason and message, got %+v", got.Conditions)
	}
	if got.Provider != "aws" {
		t.Fatalf("provider = %q, want aws", got.Provider)
	}
}
