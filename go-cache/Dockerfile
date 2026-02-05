# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files from go-cache
COPY go-cache/go.mod go-cache/go.sum ./

# Copy proxy-common submodule (required by go.mod replace directive)
COPY go-cache/proxy-common ./proxy-common

# Download dependencies
RUN go mod download

# Copy source code from go-cache
COPY go-cache/ .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o nft-proxy ./cmd/nft-proxy

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/nft-proxy .

# Copy configuration file from go-cache
COPY go-cache/cache_config.yaml /app/cache_config.yaml

# Copy secrets directory (API keys)
COPY secrets /app/secrets

# Expose metrics port (main service uses Unix socket)
EXPOSE 8099

# Run the application
CMD ["./nft-proxy"]
