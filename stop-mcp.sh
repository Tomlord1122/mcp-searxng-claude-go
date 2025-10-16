#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if containers are running
if docker ps --format "{{.Names}}" | grep -q "mcp-searxng-go\|searxng-go"; then
    print_info "Stopping MCP SearXNG Go containers..."
    docker compose down

    print_info "Containers stopped successfully."
else
    print_warning "No running containers found."
fi

# Optional: Clean up volumes (commented out by default for safety)
# Uncomment the following lines to remove data volumes
# print_info "Cleaning up volumes..."
# docker compose down -v
# rm -rf searxng/

print_info "To restart, run: ./start-mcp.sh"
