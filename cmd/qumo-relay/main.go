package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/okdaichi/qumo/relay"
	"github.com/okdaichi/qumo/relay/health"
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

	// Create health check handler
	healthHandler := health.NewStatusHandler()

	// Set upstream required if upstream URL is configured
	if config.UpstreamURL != "" {
		healthHandler.SetUpstreamRequired(true)
	}

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
		mux.HandleFunc("/health", healthHandler.ServeHTTP)
		mux.HandleFunc("/health/live", healthHandler.ServeLive)
		mux.HandleFunc("/health/ready", healthHandler.ServeReady)
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
			Upstream:        ymlConfig.Relay.UpstreamURL,
			FrameCapacity:   ymlConfig.Relay.FrameCapacity,
			GroupCacheSize:  ymlConfig.Relay.GroupCacheSize,
			HealthCheckAddr: ymlConfig.Server.HealthCheckAddr,
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

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return fallback
}
