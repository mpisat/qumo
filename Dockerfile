# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o qumo-relay ./cmd/qumo-relay

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for TLS
RUN apk add --no-cache ca-certificates

# Copy the binary from builder
COPY --from=builder /app/qumo-relay .

# Copy default configuration and certificates
COPY configs/config.docker.yaml ./configs/config.yaml
COPY certs/ ./certs/

# Expose the relay server port (UDP for QUIC)
EXPOSE 5000/udp

# Run the relay server
CMD ["./qumo-relay", "-config", "configs/config.yaml"]
