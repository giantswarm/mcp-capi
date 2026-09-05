package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration

	"github.com/giantswarm/mcp-capi/pkg/capi"
)

// Every tool returns its result as one JSON document in the text content.
// Aggregators (muster), UIs and LLM clients parse it; nothing has to scrape
// prose. List tools follow the mcp-toolkit paginated shape {items, nextCursor}
// (https://github.com/giantswarm/mcp-toolkit/blob/main/docs/conventions.md);
// nextCursor is omitted because every list tool still returns all matches in
// one page -- server-side pagination is a follow-up.
//
// structuredContent is deliberately left unset. muster mirrors it both
// natively and inside its call_tool envelope, so a payload placed there
// travels twice; the text content is the one carrier every client reads.

// jsonResult encodes v as the JSON text content of a successful tool result.
func jsonResult(v any) (*mcp.CallToolResult, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to encode tool result: %w", err)
	}
	return mcp.NewToolResultText(string(body)), nil
}

// ListResult is the shape of every list tool's result.
type ListResult struct {
	Items      any        `json:"items"`
	NextCursor mcp.Cursor `json:"nextCursor,omitempty"`
}

// listResult wraps items in the {items} envelope. Callers pass a non-nil
// slice so an empty list encodes as [] rather than null.
func listResult(items any) (*mcp.CallToolResult, error) {
	return jsonResult(ListResult{Items: items})
}

// ObjectRef names a related Kubernetes object.
type ObjectRef struct {
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

// Condition is the digest of a CAPI or core Kubernetes condition.
type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

// Address is a machine or node address.
type Address struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

// ClusterSummary is the digest of a CAPI Cluster returned by the cluster tools.
type ClusterSummary struct {
	Name                string            `json:"name"`
	Namespace           string            `json:"namespace"`
	Phase               string            `json:"phase,omitempty"`
	Ready               bool              `json:"ready"`
	Paused              bool              `json:"paused,omitempty"`
	ControlPlaneReady   bool              `json:"controlPlaneReady"`
	InfrastructureReady bool              `json:"infrastructureReady"`
	Provider            string            `json:"provider,omitempty"`
	Version             string            `json:"version,omitempty"`
	TotalMachines       int               `json:"totalMachines"`
	ReadyMachines       int               `json:"readyMachines"`
	Labels              map[string]string `json:"labels,omitempty"`
	Annotations         map[string]string `json:"annotations,omitempty"`
	Conditions          []Condition       `json:"conditions,omitempty"`
}

// clusterSummary converts a capi.ClusterStatus into its JSON digest. Labels
// and annotations under the cluster.x-k8s.io domain are system-managed and
// left out.
func clusterSummary(status *capi.ClusterStatus) ClusterSummary {
	return ClusterSummary{
		Name:                status.Name,
		Namespace:           status.Namespace,
		Phase:               status.Phase,
		Ready:               status.Ready,
		Paused:              status.Paused,
		ControlPlaneReady:   status.ControlPlaneReady,
		InfrastructureReady: status.InfraReady,
		Provider:            string(status.Provider),
		Version:             status.Version,
		TotalMachines:       status.TotalMachines,
		ReadyMachines:       status.ReadyMachines,
		Labels:              capi.UserMetadata(status.Labels),
		Annotations:         capi.UserMetadata(status.Annotations),
		Conditions:          capiConditions(status.Conditions),
	}
}

// capiConditions converts CAPI conditions into their digest. Returns nil for
// an empty input so the field is omitted.
func capiConditions(conditions clusterv1.Conditions) []Condition {
	if len(conditions) == 0 {
		return nil
	}
	out := make([]Condition, 0, len(conditions))
	for _, c := range conditions {
		out = append(out, Condition{
			Type:    string(c.Type),
			Status:  string(c.Status),
			Reason:  c.Reason,
			Message: c.Message,
		})
	}
	return out
}

// nodeConditions converts core Node conditions into their digest.
func nodeConditions(conditions []corev1.NodeCondition) []Condition {
	if len(conditions) == 0 {
		return nil
	}
	out := make([]Condition, 0, len(conditions))
	for _, c := range conditions {
		out = append(out, Condition{
			Type:    string(c.Type),
			Status:  string(c.Status),
			Reason:  c.Reason,
			Message: c.Message,
		})
	}
	return out
}

// machineAddresses converts CAPI machine addresses into their digest.
func machineAddresses(addresses clusterv1.MachineAddresses) []Address {
	if len(addresses) == 0 {
		return nil
	}
	out := make([]Address, 0, len(addresses))
	for _, a := range addresses {
		out = append(out, Address{Type: string(a.Type), Address: a.Address})
	}
	return out
}

// nodeAddresses converts core Node addresses into their digest.
func nodeAddresses(addresses []corev1.NodeAddress) []Address {
	if len(addresses) == 0 {
		return nil
	}
	out := make([]Address, 0, len(addresses))
	for _, a := range addresses {
		out = append(out, Address{Type: string(a.Type), Address: a.Address})
	}
	return out
}

// objectRef converts a core ObjectReference into its digest; nil in, nil out.
func objectRef(ref *corev1.ObjectReference) *ObjectRef {
	if ref == nil {
		return nil
	}
	return &ObjectRef{Kind: ref.Kind, Name: ref.Name, APIVersion: ref.APIVersion}
}

// machineReady reports whether the machine carries a Ready condition set to True.
func machineReady(machine *clusterv1.Machine) bool {
	for _, condition := range machine.Status.Conditions {
		if condition.Type == clusterv1.ReadyCondition && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
