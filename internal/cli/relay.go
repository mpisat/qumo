package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/okdaichi/qumo/internal/relay"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"
)

type config struct {
	Address     string
	CertFile    string
	KeyFile     string
	UpstreamURL string
	MetricsAddr string
	AdminAddr   string
	RelayConfig relay.Config
}

func main() {
	var configFile = flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup TLS
	tlsConfig, err := setupTLS(config.CertFile, config.KeyFile)
	if err != nil {
		log.Fatalf("Failed to setup TLS: %v", err)
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create relay relayServer
	relayServer := &relay.Server{
		Addr:      config.Address,
		TLSConfig: tlsConfig,
		Config:    &config.RelayConfig,
		CheckHTTPOrigin: func(r *http.Request) bool {
			return true //TODO:
		},
	}

	// Start health check HTTP server if configured
	httpServer := &http.Server{
		Addr: config.Address,
	}
	http.Handle("/health", &healthHandler{
		upstreamRequired: config.UpstreamURL != "",
		statusFunc:       relayServer.Status,
	})
	http.Handle("/metrics", promhttp.Handler())

	var wg sync.WaitGroup

	// Start relay server in a goroutine
	wg.Go(func() {
		if err := relayServer.ListenAndServe(); err != nil {
			log.Printf("Server error: %v", err)
		}
	})

	wg.Go(func() {
		if err := httpServer.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				return // Normal shutdown
			}
			log.Printf("HTTP server error: %v", err)
		}
	})

	log.Println("Server started successfully")
	log.Println("  /             - WebTransport & MoQ endpoint")
	log.Println("  /health       - Health check (?probe=live|ready)")
	log.Println("  /metrics      - Prometheus metrics")

	// Wait for shutdown signal
	<-ctx.Done()
	cancel() // Stop listening for signals, so next Ctrl+C kills the program

	slog.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown MOQT server
	if err := relayServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	// Shutdown http server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down http server: %v", err)
	}

	slog.Info("Server stopped")
}

func loadConfig(filename string) (*config, error) {
	type yamlConfig struct {
		Server struct {
			Address  string `yaml:"address"`
			CertFile string `yaml:"cert_file"`
			KeyFile  string `yaml:"key_file"`
		} `yaml:"server"`
		Relay struct {
			UpstreamURL    string `yaml:"upstream_url"`
			GroupCacheSize int    `yaml:"group_cache_size"`
			FrameCapacity  int    `yaml:"frame_capacity"`
		} `yaml:"relay"`
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var ymlConfig yamlConfig
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&ymlConfig); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	// Set defaults
	if ymlConfig.Relay.FrameCapacity == 0 {
		ymlConfig.Relay.FrameCapacity = 1500
	}
	if ymlConfig.Relay.GroupCacheSize == 0 {
		ymlConfig.Relay.GroupCacheSize = 100
	}

	config := &config{
		Address:     ymlConfig.Server.Address,
		CertFile:    ymlConfig.Server.CertFile,
		KeyFile:     ymlConfig.Server.KeyFile,
		UpstreamURL: ymlConfig.Relay.UpstreamURL,
		RelayConfig: relay.Config{
			Upstream:       ymlConfig.Relay.UpstreamURL,
			FrameCapacity:  ymlConfig.Relay.FrameCapacity,
			GroupCacheSize: ymlConfig.Relay.GroupCacheSize,
		},
	}

	return config, nil
}

func setupTLS(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h3", "moq-00"}, // HTTP/3 for WebTransport, MOQ native QUIC
	}, nil
}

type healthHandler struct {
	upstreamRequired bool
	statusFunc       func() relay.Status
}

func (h *healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// single handler that supports probes via query param: ?probe=live|ready
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	probe := r.URL.Query().Get("probe")

	switch probe {
	case "live":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
		return

	case "ready":
		status := h.statusFunc()
		activeConns := status.ActiveConnections
		upstreamConn := status.UpstreamConnected

		ready := true
		reason := "ready"

		if activeConns < 0 {
			ready = false
			reason = "invalid_connection_state"
		}
		if h.upstreamRequired && !upstreamConn {
			ready = false
			reason = "upstream_not_connected"
		}

		statusCode := http.StatusOK
		if !ready {
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if r.Method == http.MethodHead {
			return
		}

		response := map[string]any{"ready": ready}
		if !ready {
			response["reason"] = reason
		}
		json.NewEncoder(w).Encode(response)
		return

	default:
		// full status
		status := h.statusFunc()

		ready := true
		reason := "ready"
		if status.ActiveConnections < 0 {
			ready = false
			reason = "invalid_connection_state"
		}
		if h.upstreamRequired && !status.UpstreamConnected {
			ready = false
			reason = "upstream_not_connected"
		}

		response := map[string]any{
			"status":             status.Status,
			"timestamp":          status.Timestamp,
			"uptime":             status.Uptime,
			"active_connections": status.ActiveConnections,
			"upstream_connected": status.UpstreamConnected,
			"live":               true,
			"ready":              ready,
		}
		if !ready {
			response["ready_reason"] = reason
		}

		statusCode := http.StatusOK
		switch status.Status {
		case "unhealthy":
			statusCode = http.StatusServiceUnavailable
		case "degraded":
			statusCode = http.StatusOK
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if r.Method == http.MethodHead {
			return
		}
		json.NewEncoder(w).Encode(response)
		return
	}
}
