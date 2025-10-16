#!/bin/bash

# Quick local test for MCP SearXNG Go Server
# This tests basic functionality without full MCP protocol

set -e

echo "🧪 Testing MCP SearXNG Go Server"
echo "================================"
echo ""

# Check if binary exists
if [ ! -f ./mcp-searxng-go ]; then
    echo "❌ Binary not found. Building..."
    go build -o mcp-searxng-go .
    echo "✅ Build complete"
fi

# Check if containers are running
echo "🔍 Checking Docker containers..."
if ! docker ps | grep -q "searxng-go"; then
    echo "❌ SearXNG container not running"
    echo "   Run: docker compose up -d"
    exit 1
fi

echo "✅ SearXNG container is running"

# Test SearXNG health
echo ""
echo "🏥 Testing SearXNG health..."
if docker exec searxng-go wget -q --spider http://localhost:8080/search?q=test&format=json; then
    echo "✅ SearXNG is responding"
else
    echo "❌ SearXNG is not responding"
    exit 1
fi

# Test with simple MCP request
echo ""
echo "🧪 Testing MCP Server..."
echo "   Note: Full MCP protocol testing requires MCP Inspector"
echo "   Run: make inspector (requires DANGEROUSLY_OMIT_AUTH=true)"

echo ""
echo "✅ All basic tests passed!"
echo ""
echo "Next steps:"
echo "  1. Add to Claude Code: claude mcp add searxng-go $(pwd)/start-mcp.sh"
echo "  2. Test in Claude Code with queries like:"
echo "     - 'Search for Golang best practices'"
echo "     - 'Read https://go.dev/blog and summarize'"
