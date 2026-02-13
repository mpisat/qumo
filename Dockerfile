# Build stage
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o qumo .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/qumo .

# Copy config files
COPY config.relay.yaml config.sdn.yaml ./

# Create certs directory (will be mounted as volume)
RUN mkdir -p certs

# Expose ports
# 4433: MoQT (QUIC)
# 8080: HTTP health check
# 8090: SDN HTTP API
EXPOSE 4433/udp 8080 8090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command (relay)
# Override with: docker run qumo sdn -config config.sdn.yaml
ENTRYPOINT ["./qumo"]
CMD ["relay", "-config", "config.relay.yaml"]
