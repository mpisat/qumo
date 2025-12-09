# qumo

[![CI](https://github.com/okdaichi/qumo/actions/workflows/ci.yml/badge.svg)](https://github.com/okdaichi/qumo/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/okdaichi/qumo)](https://goreportcard.com/report/github.com/okdaichi/qumo)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**qumo** is a Media over QUIC (MoQ) relay server and CDN implementation, providing high-performance media streaming over the QUIC transport protocol.

## Features

- 🚀 High-performance media relay using QUIC
- 📡 Support for Media over QUIC protocol
- 🔒 Built-in TLS/security support
- 📊 Prometheus metrics for monitoring
- ⚙️ Flexible YAML-based configuration
- 🐳 Docker support (coming soon)

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Basic understanding of QUIC protocol

### Installation

```bash
go install github.com/okdaichi/qumo/cmd/qumo-relay@latest
```

### Building from Source

```bash
git clone https://github.com/okdaichi/qumo.git
cd qumo
go build -o bin/qumo-relay ./cmd/qumo-relay
```

### Running the Relay

```bash
# Copy the example configuration
cp configs/config.example.yaml config.yaml

# Edit the configuration as needed
# vim config.yaml

# Run the relay server
./bin/qumo-relay --config config.yaml
```

## Configuration

See [`configs/config.example.yaml`](configs/config.example.yaml) for a complete configuration example with detailed comments.

Basic configuration structure:

```yaml
server:
  address: "0.0.0.0:4433"
  tls:
    cert_file: "certs/server.crt"
    key_file: "certs/server.key"

relay:
  max_connections: 1000
  read_buffer_size: 65536
  write_buffer_size: 65536

logging:
  level: "info"
  format: "json"
  output: "stdout"

monitoring:
  enabled: true
  address: "0.0.0.0:9090"
  path: "/metrics"
```

## Project Structure

```
qumo/
├── cmd/
│   └── qumo-relay/        # Main relay server application
├── internal/              # Private application code
├── pkg/                   # Public library code
├── docs/                  # Documentation
├── configs/               # Configuration examples
├── monitoring/            # Monitoring and observability configs
├── .github/               # GitHub templates and workflows
└── README.md
```

## Documentation

- [Contributing Guidelines](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Issue Templates](.github/ISSUE_TEMPLATE/)

## Monitoring

qumo exposes Prometheus metrics at the `/metrics` endpoint (default port 9090).

Available metrics (to be documented as implemented):
- Connection statistics
- Throughput metrics
- Error rates
- Latency measurements

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# Run tests with race detector
go test -race ./...
```

### Linting

```bash
golangci-lint run
```

### Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Community

- [Discussions](https://github.com/okdaichi/qumo/discussions) - Ask questions and discuss ideas
- [Issues](https://github.com/okdaichi/qumo/issues) - Report bugs and request features

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Media over QUIC (MoQ) Working Group
- IETF QUIC Working Group

## Status

⚠️ **This project is under active development.** APIs and features may change.
