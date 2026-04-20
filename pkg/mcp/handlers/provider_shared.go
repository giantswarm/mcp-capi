package handlers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/giantswarm/mcp-capi/pkg/capi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// providerListConfig holds configuration for listing provider-specific clusters.
type providerListConfig struct {
	// header is the display header, e.g. "AWS Clusters:\n\n"
	header string
	// infraKinds are the InfrastructureRef.Kind values for this provider
	infraKinds []string
	// provider is the expected provider constant for confirmation
	provider capi.Provider
	// providerLabel is the provider name for the "confirmed" line, e.g. "AWS"
	providerLabel string
	// noneMsg is the message when no clusters found, e.g. "No AWS clusters found.\n"
	noneMsg string
	// totalFmt is the format string for the total line, e.g. "Total AWS clusters: %d\n"
	totalFmt string
}

// buildListClustersHandler returns a ToolHandlerFunc that lists clusters filtered
// by infrastructure provider. It is shared by all provider list-clusters handlers.
func buildListClustersHandler(serverCtx *ServerContext, cfg providerListConfig) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, _ := arguments["namespace"].(string)

		clusters, err := serverCtx.CAPIClient.ListClusters(ctx, namespace, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		var content strings.Builder
		content.WriteString(cfg.header)

		clusterCount := 0
		for i := range clusters.Items {
			cluster := &clusters.Items[i]
			if cluster.Spec.InfrastructureRef == nil {
				continue
			}
			if !slices.Contains(cfg.infraKinds, cluster.Spec.InfrastructureRef.Kind) {
				continue
			}

			clusterCount++
			fmt.Fprintf(&content, "Cluster: %s/%s\n", cluster.Namespace, cluster.Name)
			fmt.Fprintf(&content, "  Infrastructure: %s\n", cluster.Spec.InfrastructureRef.Kind)
			fmt.Fprintf(&content, "  Phase: %s\n", cluster.Status.Phase)
			fmt.Fprintf(&content, "  Ready: %v\n", cluster.Status.InfrastructureReady)

			provider, _ := serverCtx.CAPIClient.GetProviderForCluster(ctx, cluster.Namespace, cluster.Name)
			if provider == cfg.provider {
				fmt.Fprintf(&content, "  Provider: %s (confirmed)\n", cfg.providerLabel)
			}
			content.WriteString("\n")
		}

		if clusterCount == 0 {
			content.WriteString(cfg.noneMsg)
		} else {
			fmt.Fprintf(&content, cfg.totalFmt, clusterCount)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: content.String(),
				},
			},
		}, nil
	}
}

// providerGetConfig holds configuration for retrieving details of a provider-specific cluster.
type providerGetConfig struct {
	// infraKinds are the valid InfrastructureRef.Kind values for this provider
	infraKinds []string
	// notProviderFmt is used to format the "not a X cluster" error, e.g. "Cluster %s/%s is not an Azure cluster"
	notProviderFmt string
	// headerFmt is used for the response header, e.g. "Azure Cluster: %s/%s\n\n"
	headerFmt string
	// noteFmt is the closing note about detailed infrastructure information
	noteFmt string
}

// buildGetClusterHandler returns a ToolHandlerFunc that retrieves basic details of a
// provider-specific cluster. It is shared by Azure and GCP get-cluster handlers.
func buildGetClusterHandler(serverCtx *ServerContext, cfg providerGetConfig) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		namespace, ok := arguments["namespace"].(string)
		if !ok || namespace == "" {
			return nil, errors.New("namespace argument is required")
		}
		name, ok := arguments["name"].(string)
		if !ok || name == "" {
			return nil, errors.New("name argument is required")
		}

		cluster, err := serverCtx.CAPIClient.GetCluster(ctx, namespace, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster: %w", err)
		}

		// Verify the cluster belongs to the expected provider
		if cluster.Spec.InfrastructureRef == nil ||
			!slices.Contains(cfg.infraKinds, cluster.Spec.InfrastructureRef.Kind) {
			return mcp.NewToolResultError(fmt.Sprintf(cfg.notProviderFmt, namespace, name)), nil
		}

		var content strings.Builder
		fmt.Fprintf(&content, cfg.headerFmt, namespace, name)

		content.WriteString("Cluster Information:\n")
		fmt.Fprintf(&content, "  Phase: %s\n", cluster.Status.Phase)
		fmt.Fprintf(&content, "  Infrastructure Ready: %v\n", cluster.Status.InfrastructureReady)
		fmt.Fprintf(&content, "  Control Plane Ready: %v\n", cluster.Status.ControlPlaneReady)

		content.WriteString("\nInfrastructure:\n")
		fmt.Fprintf(&content, "  Kind: %s\n", cluster.Spec.InfrastructureRef.Kind)
		fmt.Fprintf(&content, "  Name: %s\n", cluster.Spec.InfrastructureRef.Name)

		content.WriteString(cfg.noteFmt)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: content.String(),
				},
			},
		}, nil
	}
}

// buildPlaceholderHandler returns a ToolHandlerFunc that returns a static placeholder text.
// It is used for provider-specific write operations that are not yet implemented.
func buildPlaceholderHandler(text string) server.ToolHandlerFunc {
	return func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: text,
				},
			},
		}, nil
	}
}
