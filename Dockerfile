# ============ Build Stage ============
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mcp-searxng-go .

# ============ Release Stage ============
FROM alpine:latest AS release

# Update system packages for security
RUN apk update && apk upgrade && apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/mcp-searxng-go .

# Run as non-root user
RUN adduser -D -u 1000 mcpuser
USER mcpuser

# Set the entrypoint
ENTRYPOINT ["./mcp-searxng-go"]
