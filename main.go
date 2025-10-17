package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const VERSION = "0.7.0"

func main() {
	// Load environment variables from .env file if it exists
	_ = godotenv.Load()

	// Validate environment
	if err := validateEnvironment(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Create MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-searxng-go",
		Version: VERSION,
	}, nil)

	// Initialize services
	cache := NewCache(60) // 60 seconds TTL
	defer cache.Destroy()

	proxyConfig := LoadProxyConfig()
	searxngClient := NewSearXNGClient(os.Getenv("SEARXNG_URL"), proxyConfig)
	urlReader := NewURLReader(cache, proxyConfig)

	// Register tools
	registerTools(server, searxngClient, urlReader)

	// Register resources
	registerResources(server)

	// Start server with stdio transport
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func validateEnvironment() error {
	searxngURL := os.Getenv("SEARXNG_URL")
	if searxngURL == "" {
		return fmt.Errorf("SEARXNG_URL environment variable is required")
	}
	return nil
}

func registerTools(server *mcp.Server, client *SearXNGClient, reader *URLReader) {
	// Web search tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "searxng_web_search",
		Description: "Performs a web search using the SearXNG API, ideal for general queries, news, articles, and online content.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args WebSearchArgs) (*mcp.CallToolResult, any, error) {
		return handleWebSearch(ctx, req, client, args)
	})

	// URL read tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "web_url_read",
		Description: "Read the content from a URL. Use this for further information retrieving.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args URLReadArgs) (*mcp.CallToolResult, any, error) {
		return handleURLRead(ctx, req, reader, args)
	})
}

func registerResources(server *mcp.Server) {
	// Config resource
	server.AddResource(&mcp.Resource{
		Name:        "Server Configuration",
		URI:         "config://mcp-searxng",
		Description: "Current server configuration",
		MIMEType:    "application/json",
	}, createConfigResourceHandler)

	// Help resource
	server.AddResource(&mcp.Resource{
		Name:        "Usage Guide",
		URI:         "help://mcp-searxng",
		Description: "MCP SearXNG usage guide",
		MIMEType:    "text/markdown",
	}, createHelpResourceHandler)
}
