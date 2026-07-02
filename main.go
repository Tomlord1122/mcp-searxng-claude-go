package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

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
	cache := NewCache(cacheTTLSeconds(), cacheMaxEntries())
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

// cacheTTLSeconds reads CACHE_TTL from the environment (seconds).
// Falls back to 60s if unset or invalid. This was previously a
// hardcoded value even though README documented it as configurable.
func cacheTTLSeconds() int {
	const defaultTTL = 60
	v := os.Getenv("CACHE_TTL")
	if v == "" {
		return defaultTTL
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		log.Printf("warning: invalid CACHE_TTL=%q, falling back to %ds", v, defaultTTL)
		return defaultTTL
	}
	return n
}

// cacheMaxEntries reads CACHE_MAX_ENTRIES from the environment.
// Bounds the in-memory url-read cache so it cannot grow unboundedly
// between TTL cleanup cycles. Falls back to 500 if unset or invalid.
func cacheMaxEntries() int {
	const defaultMax = 500
	v := os.Getenv("CACHE_MAX_ENTRIES")
	if v == "" {
		return defaultMax
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		log.Printf("warning: invalid CACHE_MAX_ENTRIES=%q, falling back to %d", v, defaultMax)
		return defaultMax
	}
	return n
}

func registerTools(server *mcp.Server, client *SearXNGClient, reader *URLReader) {
	// Web search tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "web_search",
		Description: "Performs a web search using the SearXNG API, ideal for general queries, news, articles, and online content.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args WebSearchArgs) (*mcp.CallToolResult, any, error) {
		return handleWebSearch(ctx, req, client, args)
	})

	// URL read tool
	mcp.AddTool(server, &mcp.Tool{
		Name:        "url_read",
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
