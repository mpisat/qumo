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
	"syscall"

	"github.com/okdaichi/qumo/relay"
	"github.com/okdaichi/qumo/relay/health"
	"gopkg.in/yaml.v3"
)

type config struct {
	Address         string
	CertFile        string
	KeyFile         string
	UpstreamURL     string
	HealthCheckAddr string
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

	// Apply relay configuration
	relay.NewFrameCapacity = config.RelayConfig.FrameCapacity
	relay.GroupCacheSize = config.RelayConfig.GroupCacheSize

	log.Printf("Starting qumo-relay server on %s", config.Address)

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
		Health:    healthHandler,
	}

	// Start health check HTTP server if configured
	var httpServer *http.Server
	if config.HealthCheckAddr != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/health", healthHandler.ServeHTTP)
		mux.HandleFunc("/health/live", healthHandler.ServeLive)
		mux.HandleFunc("/health/ready", healthHandler.ServeReady)

		httpServer = &http.Server{
			Addr:    config.HealthCheckAddr,
			Handler: mux,
		}

		go func() {
			log.Printf("Starting health check server on %s", config.HealthCheckAddr)
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Health check server error: %v", err)
			}
		}()
	}

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Println("Server started successfully")

	// Wait for shutdown signal
	<-ctx.Done()

	slog.Info("Shutting down server...")

	// Graceful shutdown
	shutdownCtx := context.Background()

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
		NextProtos:   []string{"h3"}, // HTTP/3 for QUIC
	}, nil
}
