package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const VERSION = "0.7.0"

func main() {
	// Load environment variables from .env file if it exists
	_ = godotenv.Load()

	// Validate environment
	if err := validateEnvironment(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Create MCP server with capabilities
	s := server.NewMCPServer(
		"mcp-searxng-go",
		VERSION,
		server.WithResourceCapabilities(false, false),
		server.WithLogging(),
	)

	// Initialize services
	cache := NewCache(60) // 60 seconds TTL
	defer cache.Destroy()

	proxyConfig := LoadProxyConfig()
	searxngClient := NewSearXNGClient(os.Getenv("SEARXNG_URL"), proxyConfig)
	urlReader := NewURLReader(cache, proxyConfig)

	// Register tools
	registerTools(s, searxngClient, urlReader)

	// Register resources
	registerResources(s)

	// Start server with stdio transport
	if err := server.ServeStdio(s); err != nil {
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

func registerTools(s *server.MCPServer, client *SearXNGClient, reader *URLReader) {
	// Web search tool
	webSearchTool := mcp.NewTool("searxng_web_search",
		mcp.WithDescription("Performs a web search using the SearXNG API, ideal for general queries, news, articles, and online content."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query")),
		mcp.WithNumber("pageno",
			mcp.Description("Search page number (starts at 1)"),
			mcp.DefaultNumber(1)),
		mcp.WithString("time_range",
			mcp.Description("Time range of search"),
			mcp.Enum("day", "month", "year")),
		mcp.WithString("language",
			mcp.Description("Language code for search results (e.g., 'en', 'fr', 'de')"),
			mcp.DefaultString("all")),
		mcp.WithString("safesearch",
			mcp.Description("Safe search filter level (0: None, 1: Moderate, 2: Strict)"),
			mcp.Enum("0", "1", "2"),
			mcp.DefaultString("0")),
	)

	s.AddTool(webSearchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleWebSearch(ctx, request, client)
	})

	// URL read tool
	urlReadTool := mcp.NewTool("web_url_read",
		mcp.WithDescription("Read the content from a URL. Use this for further information retrieving."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("URL to read")),
		mcp.WithNumber("startChar",
			mcp.Description("Starting character position for content extraction (default: 0)")),
		mcp.WithNumber("maxLength",
			mcp.Description("Maximum number of characters to return")),
		mcp.WithString("section",
			mcp.Description("Extract content under a specific heading")),
		mcp.WithString("paragraphRange",
			mcp.Description("Return specific paragraph ranges (e.g., '1-5', '3', '10-')")),
		mcp.WithBoolean("readHeadings",
			mcp.Description("Return only a list of headings instead of full content")),
	)

	s.AddTool(urlReadTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleURLRead(ctx, request, reader)
	})
}

func registerResources(s *server.MCPServer) {
	// Config resource
	configResource := mcp.NewResource(
		"config://mcp-searxng",
		"Server Configuration",
		mcp.WithResourceDescription("Current server configuration"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(configResource, createConfigResourceHandler)

	// Help resource
	helpResource := mcp.NewResource(
		"help://mcp-searxng",
		"Usage Guide",
		mcp.WithResourceDescription("MCP SearXNG usage guide"),
		mcp.WithMIMEType("text/markdown"),
	)

	s.AddResource(helpResource, createHelpResourceHandler)
}
