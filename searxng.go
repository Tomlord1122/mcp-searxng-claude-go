package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

type SearXNGClient struct {
	baseURL    string
	httpClient *http.Client
}

type SearXNGResult struct {
	Title   string  `json:"title"`
	Content string  `json:"content"`
	URL     string  `json:"url"`
	Score   float64 `json:"score"`
}

type SearXNGResponse struct {
	Results []SearXNGResult `json:"results"`
}

func NewSearXNGClient(baseURL string, proxyConfig *ProxyConfig) *SearXNGClient {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	if proxyConfig != nil && proxyConfig.Transport != nil {
		client.Transport = proxyConfig.Transport
	}

	return &SearXNGClient{
		baseURL:    baseURL,
		httpClient: client,
	}
}

func (c *SearXNGClient) Search(ctx context.Context, query string, pageno int, timeRange, language, safesearch string) (*SearXNGResponse, error) {
	// Build URL
	searchURL, err := url.Parse(c.baseURL + "/search")
	if err != nil {
		return nil, fmt.Errorf("invalid SearXNG URL: %w", err)
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "json")
	params.Set("pageno", strconv.Itoa(pageno))

	if timeRange != "" && (timeRange == "day" || timeRange == "month" || timeRange == "year") {
		params.Set("time_range", timeRange)
	}

	if language != "" && language != "all" {
		params.Set("language", language)
	}

	if safesearch != "" && (safesearch == "0" || safesearch == "1" || safesearch == "2") {
		params.Set("safesearch", safesearch)
	}

	searchURL.RawQuery = params.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add required headers to prevent bot detection
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	req.Header.Set("X-Real-IP", "127.0.0.1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MCP-SearXNG-Go/1.0)")

	// Add basic auth if configured
	username := os.Getenv("AUTH_USERNAME")
	password := os.Getenv("AUTH_PASSWORD")
	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SearXNG returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result SearXNGResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func handleWebSearch(ctx context.Context, request mcp.CallToolRequest, client *SearXNGClient) (*mcp.CallToolResult, error) {
	// Extract parameters
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	pageno := request.GetInt("pageno", 1)
	timeRange := request.GetString("time_range", "")
	language := request.GetString("language", "all")
	safesearch := request.GetString("safesearch", "0")

	// Perform search
	startTime := time.Now()
	results, err := client.Search(ctx, query, pageno, timeRange, language, safesearch)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Search failed: %v", err)), nil
	}

	duration := time.Since(startTime)

	// Format output
	if len(results.Results) == 0 {
		output := fmt.Sprintf("# No Results Found\n\nNo results found for query: \"%s\"\n\nTry:\n- Different keywords\n- Broader search terms\n- Checking spelling", query)
		return mcp.NewToolResultText(output), nil
	}

	output := fmt.Sprintf("# Search Results for \"%s\"\n\n", query)
	output += fmt.Sprintf("Found %d results (page %d) in %dms\n\n", len(results.Results), pageno, duration.Milliseconds())

	for i, result := range results.Results {
		output += fmt.Sprintf("## %d. %s\n\n", i+1, result.Title)
		output += fmt.Sprintf("**URL:** %s\n\n", result.URL)
		output += fmt.Sprintf("%s\n\n", result.Content)
		output += "---\n\n"
	}

	return mcp.NewToolResultText(output), nil
}
