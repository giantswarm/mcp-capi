package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "mcp-capi"
	serverVersion = "0.1.0"
)

func main() {
	// Create context that cancels on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received, closing server...")
		cancel()
	}()

	// Create MCP server
	mcpServer := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true), // subscribe, list
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	// Add a simple test tool
	testTool := mcp.NewTool(
		"capi_test",
		mcp.WithDescription("Test tool to verify MCP server is working"),
		mcp.WithString("message", 
			mcp.Required(), 
			mcp.Description("Test message to echo"),
		),
	)

	mcpServer.AddTool(testTool, testToolHandler)

	// Add a simple test resource
	testResource := mcp.NewResource(
		"capi://test",
		"Test Resource",
		mcp.WithMIMEType("text/plain"),
	)
	
	mcpServer.AddResource(testResource, testResourceHandler)

	// Start server based on transport type
	transport := os.Getenv("MCP_TRANSPORT")
	if transport == "" {
		transport = "stdio"
	}

	// Set up signal handling for graceful shutdown
	go func() {
		<-ctx.Done()
		log.Println("Context cancelled, shutting down...")
		os.Exit(0)
	}()

	switch transport {
	case "stdio":
		log.Println("Starting MCP CAPI server with stdio transport...")
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	default:
		log.Fatalf("Unsupported transport: %s", transport)
	}
}

func testToolHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	arguments := request.GetArguments()
	message, ok := arguments["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message argument is required and must be a string")
	}

	response := fmt.Sprintf("Echo from CAPI MCP Server: %s", message)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: response,
			},
		},
	}, nil
}

func testResourceHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "text/plain",
			Text:     "This is a test resource from the CAPI MCP server.",
		},
	}, nil
} 