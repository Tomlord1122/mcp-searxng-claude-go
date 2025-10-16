package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

func createConfigResource() string {
	config := map[string]interface{}{
		"version":     VERSION,
		"searxng_url": os.Getenv("SEARXNG_URL"),
		"proxy": map[string]string{
			"http":  os.Getenv("HTTP_PROXY"),
			"https": os.Getenv("HTTPS_PROXY"),
		},
		"cache": map[string]interface{}{
			"enabled": true,
			"ttl":     60,
		},
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}

func createHelpResource() string {
	return `# MCP SearXNG Server - Usage Guide

## Overview

This MCP server provides web search capabilities through SearXNG and URL content extraction.

## Available Tools

### 1. searxng_web_search

Performs web searches using the SearXNG metasearch engine.

**Parameters:**
- ` + "`query`" + ` (required): Search query string
- ` + "`pageno`" + ` (optional): Page number (default: 1)
- ` + "`time_range`" + ` (optional): Filter by time ("day", "month", "year")
- ` + "`language`" + ` (optional): Language code (e.g., "en", "fr", "de")
- ` + "`safesearch`" + ` (optional): Safe search level ("0", "1", "2")

**Example:**
` + "```" + `
query: "TypeScript best practices 2024"
pageno: 1
language: "en"
` + "```" + `

### 2. web_url_read

Reads and converts web page content to Markdown format.

**Parameters:**
- ` + "`url`" + ` (required): URL to read
- ` + "`startChar`" + ` (optional): Starting character position
- ` + "`maxLength`" + ` (optional): Maximum characters to return
- ` + "`section`" + ` (optional): Extract specific heading section
- ` + "`paragraphRange`" + ` (optional): Paragraph range (e.g., "1-5", "10-")
- ` + "`readHeadings`" + ` (optional): Return only headings (boolean)

**Example:**
` + "```" + `
url: "https://example.com/article"
maxLength: 5000
` + "```" + `

## Configuration

The server requires the following environment variables:

- ` + "`SEARXNG_URL`" + `: SearXNG instance URL (required)
- ` + "`AUTH_USERNAME`" + `: Basic auth username (optional)
- ` + "`AUTH_PASSWORD`" + `: Basic auth password (optional)
- ` + "`HTTP_PROXY`" + `: HTTP proxy URL (optional)
- ` + "`HTTPS_PROXY`" + `: HTTPS proxy URL (optional)

## Features

- **Caching**: URL content is cached for 60 seconds to reduce load
- **Proxy Support**: Automatic proxy detection from environment
- **Privacy**: All searches go through your own SearXNG instance
- **Markdown Conversion**: HTML content is automatically converted to Markdown

## Getting Help

For more information, visit: https://github.com/AI-Task-Force/mcp-searxng-go
`
}

func createConfigResourceHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	content := createConfigResource()
	return []mcp.ResourceContents{&mcp.TextResourceContents{
		URI:      "config://mcp-searxng",
		MIMEType: "application/json",
		Text:     content,
	}}, nil
}

func createHelpResourceHandler(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	content := createHelpResource()
	return []mcp.ResourceContents{&mcp.TextResourceContents{
		URI:      "help://mcp-searxng",
		MIMEType: "text/markdown",
		Text:     content,
	}}, nil
}
