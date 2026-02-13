# Magefiles (isolated)

This directory contains the project's Mage tasks and a dedicated `go.mod` so Mage's dependencies are isolated from the main module.

## Quick Start

First, create your configuration files:
```bash
# Copy example configs to actual config files
cp config.sdn.example.yaml config.sdn.yaml
cp config.relay.example.yaml config.relay.yaml
```

Then run mage tasks from the repo root:
```bash
# Build the binary
mage build

# Run tests
mage test

# Start SDN controller
mage sdn

# Start relay server (in another terminal)
mage relay

# Start web demo (in another terminal)
mage web
```

## Available Targets

Run `mage help` or `mage -l` to see all available targets.

### 🔨 Build & Install
- `mage build` - Build qumo binary
- `mage install` - Install to $GOPATH/bin
- `mage clean` - Clean build artifacts

### 🧪 Development
- `mage test` - Run all tests
- `mage testVerbose` - Run tests with verbose output
- `mage fmt` - Format code
- `mage vet` - Run static analysis
- `mage lint` - Run golangci-lint
- `mage check` - Run fmt, vet, and test

### 🚀 Runtime
- `mage relay` - Start relay server
- `mage sdn` - Start SDN controller
- `mage dev` - Development mode info

### 🌐 Web Demo
- `mage web` - Start Vite dev server
- `mage webBuild` - Build for production
- `mage webClean` - Clean build artifacts

### 🔧 Utilities
- `mage cert` - Generate TLS certificates
- `mage hash` - Compute cert hash

## Usage

From the repo root:
```bash
# Run mage tasks directly — mage auto-detects ./magefiles
mage <target>
# optional: mage -d ./magefiles <target>
```
