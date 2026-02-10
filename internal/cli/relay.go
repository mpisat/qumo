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
	"syscall"
	"time"

	"github.com/okdaichi/qumo/internal/relay"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"
)

type config struct {
	Address         string
	CertFile        string
	KeyFile         string
	UpstreamURL     string
	HealthCheckAddr string
	MetricsAddr     string
	AdminAddr       string
	RelayConfig     relay.Config
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

	slog.Info("Starting qumo-relay server", "address", config.Address)

	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create MOQT server
	server := &relay.Server{
		Addr:      config.Address,
		TLSConfig: tlsConfig,
		Config:    &config.RelayConfig,
		CheckHTTPOrigin: func(r *http.Request) bool {
			return true //TODO:
		},
	}

	// Start health check HTTP server if configured
	var httpServer *http.Server
	if config.HealthCheckAddr != "" {
		mux := http.NewServeMux()
		registerHealthHandler(server, mux)
		mux.Handle("/metrics", promhttp.Handler())

		httpServer = &http.Server{
			Addr:    config.HealthCheckAddr,
			Handler: mux,
		}

		go func() {
			log.Printf("HTTP server starting on %s", config.HealthCheckAddr)
			log.Println("  /health       - Health check")
			log.Println("  /health/live  - Liveness probe")
			log.Println("  /health/ready - Readiness probe")
			log.Println("  /metrics      - Prometheus metrics")
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP server error: %v", err)
			}
		}()
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting MoQ relay server on %s", config.Address)
		if err := server.ListenAndServe(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Println("Server started successfully")

	// Wait for shutdown signal
	<-ctx.Done()
	cancel() // Stop listening for signals, so next Ctrl+C kills the program

	slog.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown health check server
	if httpServer != nil {
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down health check server: %v", err)
		}
	}

	// Shutdown MOQT server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	slog.Info("Server stopped")
}

func loadConfig(filename string) (*config, error) {
	type yamlConfig struct {
		Server struct {
			Address         string `yaml:"address"`
			CertFile        string `yaml:"cert_file"`
			KeyFile         string `yaml:"key_file"`
			HealthCheckAddr string `yaml:"health_check_addr"`
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
		Address:         ymlConfig.Server.Address,
		CertFile:        ymlConfig.Server.CertFile,
		KeyFile:         ymlConfig.Server.KeyFile,
		UpstreamURL:     ymlConfig.Relay.UpstreamURL,
		HealthCheckAddr: ymlConfig.Server.HealthCheckAddr,
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

func registerHealthHandler(server *relay.Server, mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		status := server.Status()

		// Set status code based on health
		statusCode := http.StatusOK
		switch status.Status {
		case "unhealthy":
			statusCode = http.StatusServiceUnavailable
		case "degraded":
			statusCode = http.StatusOK // Still operational, just degraded
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if r.Method == http.MethodHead {
			return
		}

		json.NewEncoder(w).Encode(status)
	})

	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Liveness: just check if we can respond
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodHead {
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
		})
	})

	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		status := server.Status()

		activeConns := status.ActiveConnections
		upstreamConn := status.UpstreamConnected

		// Readiness checks
		ready := true
		reason := "ready"

		// Check if connections are in valid state
		if activeConns < 0 {
			ready = false
			reason = "invalid_connection_state"
		}

		// Check upstream if required
		if server.Config.Upstream != "" && !upstreamConn {
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

		response := map[string]any{
			"ready": ready,
		}
		if !ready {
			response["reason"] = reason
		}

		json.NewEncoder(w).Encode(response)
	})
}
