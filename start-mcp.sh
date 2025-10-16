#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

# Check if Docker is installed
if ! command -v docker &>/dev/null; then
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker daemon is running
if ! docker info &>/dev/null; then
    print_error "Docker daemon is not running. Please start Docker."
    exit 1
fi

# Check if .env file exists
if [ ! -f .env ]; then
    print_warning ".env file not found. Creating from template..."
    cat > .env << EOF
# SearXNG Configuration
SEARXNG_SECRET=$(python3 -c "import secrets; print(secrets.token_hex(16))" 2>/dev/null || openssl rand -hex 16)

# Optional: HTTP Proxy
# HTTP_PROXY=http://proxy.example.com:8080
# HTTPS_PROXY=http://proxy.example.com:8080

# Optional: Basic Auth for SearXNG
# AUTH_USERNAME=admin
# AUTH_PASSWORD=changeme
EOF
    print_info ".env file created. Please review and edit if needed."
fi

# Check container status
SEARXNG_RUNNING=$(docker ps --filter "name=searxng-go" --filter "status=running" --format "{{.Names}}" 2>/dev/null || true)
MCP_RUNNING=$(docker ps --filter "name=mcp-searxng-go" --filter "status=running" --format "{{.Names}}" 2>/dev/null || true)

# Start containers if not running
if [ -z "$SEARXNG_RUNNING" ] || [ -z "$MCP_RUNNING" ]; then
    print_info "Starting containers..."

    # Build and start services
    docker compose build --no-cache 2>&1 | grep -v "WARNING" || true
    docker compose up -d

    print_info "Waiting for containers to be ready..."
    sleep 3
else
    print_info "Containers are already running."
fi

# Wait for MCP container to be ready (max 30 seconds)
TIMEOUT=30
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
    if docker ps --filter "name=mcp-searxng-go" --filter "status=running" --format "{{.Names}}" | grep -q "^mcp-searxng-go$"; then
        print_info "MCP SearXNG Go server is ready."
        break
    fi
    sleep 1
    ELAPSED=$((ELAPSED + 1))
done

# Check if timeout occurred
if [ $ELAPSED -ge $TIMEOUT ]; then
    print_error "Timeout waiting for mcp-searxng-go container to start."
    print_info "Container logs:"
    docker logs mcp-searxng-go 2>&1 || echo "Could not retrieve container logs"
    exit 1
fi

# Wait for SearXNG to be ready
print_info "Waiting for SearXNG to be ready..."
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
    if docker exec searxng-go wget -q --spider http://localhost:8080 2>/dev/null; then
        print_info "SearXNG is ready."
        break
    fi
    sleep 1
    ELAPSED=$((ELAPSED + 1))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    print_warning "SearXNG may not be fully ready, but continuing..."
fi

print_info "Starting MCP server..."
print_info "Press Ctrl+C to stop the server."

# Execute the MCP server (keeps stdin open for MCP protocol)
exec docker exec -i mcp-searxng-go ./mcp-searxng-go
