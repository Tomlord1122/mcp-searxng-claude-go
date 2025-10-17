package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type URLReader struct {
	cache      *Cache
	httpClient *http.Client
}

// URLReadArgs defines the parameters for URL reading
type URLReadArgs struct {
	URL            string `json:"url" jsonschema:"URL to read"`
	StartChar      int    `json:"startChar,omitempty" jsonschema:"starting character position for content extraction (default: 0)"`
	MaxLength      int    `json:"maxLength,omitempty" jsonschema:"maximum number of characters to return"`
	Section        string `json:"section,omitempty" jsonschema:"extract content under a specific heading"`
	ParagraphRange string `json:"paragraphRange,omitempty" jsonschema:"return specific paragraph ranges (e.g., '1-5', '3', '10-')"`
	ReadHeadings   bool   `json:"readHeadings,omitempty" jsonschema:"return only a list of headings instead of full content"`
}

func NewURLReader(cache *Cache, proxyConfig *ProxyConfig) *URLReader {
	client := &http.Client{
		Timeout: 30 * time.Second, // Increased timeout for large pages
	}

	if proxyConfig != nil && proxyConfig.Transport != nil {
		client.Transport = proxyConfig.Transport
	}

	return &URLReader{
		cache:      cache,
		httpClient: client,
	}
}

func (r *URLReader) FetchAndConvert(ctx context.Context, urlStr string) (string, error) {
	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("URL must use http or https scheme")
	}

	// Check cache
	if cached := r.cache.Get(urlStr); cached != "" {
		return cached, nil
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MCP-SearXNG-Go/1.0)")

	// Execute request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read body with size limit (10MB max to prevent memory issues)
	const maxBodySize = 10 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxBodySize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Convert HTML to Markdown (simplified conversion)
	markdown := htmlToMarkdown(string(body))

	// Cache result
	r.cache.Set(urlStr, markdown)

	return markdown, nil
}

func htmlToMarkdown(html string) string {
	// Simple HTML to Markdown conversion
	// Remove script and style tags (Go regex doesn't support backreferences like \1)
	re := regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
	content := re.ReplaceAllString(html, "")
	re = regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`)
	content = re.ReplaceAllString(content, "")

	// Convert headings (h1-h6)
	for i := 6; i >= 1; i-- {
		pattern := fmt.Sprintf(`(?s)<h%d[^>]*>(.*?)</h%d>`, i, i)
		re = regexp.MustCompile(pattern)
		replacement := strings.Repeat("#", i) + " $1\n\n"
		content = re.ReplaceAllString(content, replacement)
	}

	// Convert paragraphs
	re = regexp.MustCompile(`(?s)<p[^>]*>(.*?)</p>`)
	content = re.ReplaceAllString(content, "$1\n\n")

	// Convert links
	re = regexp.MustCompile(`(?s)<a[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	content = re.ReplaceAllString(content, "[$2]($1)")

	// Convert bold (both <b> and <strong>)
	re = regexp.MustCompile(`(?s)<b[^>]*>(.*?)</b>`)
	content = re.ReplaceAllString(content, "**$1**")
	re = regexp.MustCompile(`(?s)<strong[^>]*>(.*?)</strong>`)
	content = re.ReplaceAllString(content, "**$1**")

	// Convert italic (both <i> and <em>)
	re = regexp.MustCompile(`(?s)<i[^>]*>(.*?)</i>`)
	content = re.ReplaceAllString(content, "*$1*")
	re = regexp.MustCompile(`(?s)<em[^>]*>(.*?)</em>`)
	content = re.ReplaceAllString(content, "*$1*")

	// Convert lists
	re = regexp.MustCompile(`(?s)<li[^>]*>(.*?)</li>`)
	content = re.ReplaceAllString(content, "- $1\n")

	// Remove remaining HTML tags
	re = regexp.MustCompile(`<[^>]+>`)
	content = re.ReplaceAllString(content, "")

	// Clean up whitespace
	re = regexp.MustCompile(`\n{3,}`)
	content = re.ReplaceAllString(content, "\n\n")

	return strings.TrimSpace(content)
}

func handleURLRead(ctx context.Context, req *mcp.CallToolRequest, reader *URLReader, args URLReadArgs) (result *mcp.CallToolResult, _ any, err error) {
	// Add panic recovery to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in URL read: %v", r)
			result = &mcp.CallToolResult{
				IsError: true,
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Internal error: %v", r)},
				},
			}
		}
	}()

	// Validate required parameter
	if args.URL == "" {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: "url parameter is required"},
			},
		}, nil, nil
	}

	// Fetch content
	content, err := reader.FetchAndConvert(ctx, args.URL)
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to read URL: %v", err)},
			},
		}, nil, nil
	}

	// Apply pagination options
	content = applyPaginationOptions(content, args)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: content},
		},
	}, nil, nil
}

func applyPaginationOptions(content string, args URLReadArgs) string {
	// Read headings only
	if args.ReadHeadings {
		return extractHeadings(content)
	}

	// Extract specific section
	if args.Section != "" {
		content = extractSection(content, args.Section)
	}

	// Extract paragraph range
	if args.ParagraphRange != "" {
		content = extractParagraphRange(content, args.ParagraphRange)
	}

	// Character-level pagination
	if args.StartChar > 0 || args.MaxLength > 0 {
		content = applyCharacterPagination(content, args.StartChar, args.MaxLength)
	}

	return content
}

func extractHeadings(content string) string {
	lines := strings.Split(content, "\n")
	var headings []string

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			headings = append(headings, line)
		}
	}

	return strings.Join(headings, "\n")
}

func extractSection(content, sectionHeading string) string {
	lines := strings.Split(content, "\n")
	sectionRegex := regexp.MustCompile(`(?i)^#{1,6}\s*.*` + regexp.QuoteMeta(sectionHeading) + `.*$`)

	startIndex := -1
	currentLevel := 0

	// Find section start
	for i, line := range lines {
		if sectionRegex.MatchString(line) {
			startIndex = i
			// Count heading level
			currentLevel = len(strings.Split(line, " ")[0])
			break
		}
	}

	if startIndex == -1 {
		return ""
	}

	// Find section end
	endIndex := len(lines)
	for i := startIndex + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "#") {
			level := len(strings.Split(lines[i], " ")[0])
			if level <= currentLevel {
				endIndex = i
				break
			}
		}
	}

	return strings.Join(lines[startIndex:endIndex], "\n")
}

func extractParagraphRange(content, rangeStr string) string {
	paragraphs := strings.Split(content, "\n\n")
	var filtered []string
	for _, p := range paragraphs {
		if strings.TrimSpace(p) != "" {
			filtered = append(filtered, p)
		}
	}

	// Parse range (e.g., "1-5", "3", "10-")
	rangeRegex := regexp.MustCompile(`^(\d+)(?:-(\d*))?$`)
	matches := rangeRegex.FindStringSubmatch(rangeStr)
	if matches == nil {
		return ""
	}

	start, _ := strconv.Atoi(matches[1])
	start-- // Convert to 0-based

	end := start + 1
	if matches[2] != "" {
		if endNum, err := strconv.Atoi(matches[2]); err == nil {
			end = endNum
		} else {
			end = len(filtered)
		}
	} else if len(matches) > 2 && matches[2] == "" {
		end = len(filtered)
	}

	if start >= len(filtered) {
		return ""
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	return strings.Join(filtered[start:end], "\n\n")
}

func applyCharacterPagination(content string, startChar, maxLength int) string {
	if startChar >= len(content) {
		return ""
	}

	if startChar < 0 {
		startChar = 0
	}

	end := len(content)
	if maxLength > 0 {
		end = startChar + maxLength
		if end > len(content) {
			end = len(content)
		}
	}

	return content[startChar:end]
}
