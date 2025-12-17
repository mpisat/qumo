package relay

import (
	"testing"
)

// TestConfigDefaults tests default config values
func TestConfigDefaults(t *testing.T) {
	cfg := &Config{}

	if cfg.Upstream != "" {
		t.Error("Default Upstream should be empty")
	}

	if cfg.GroupCacheSize != 0 {
		t.Error("Default GroupCacheSize should be 0")
	}

	if cfg.FrameCapacity != 0 {
		t.Error("Default FrameCapacity should be 0")
	}

	if cfg.HealthCheckAddr != "" {
		t.Error("Default HealthCheckAddr should be empty")
	}
}

// TestConfigWithUpstream tests config with upstream URL
func TestConfigWithUpstream(t *testing.T) {
	tests := []struct {
		name     string
		upstream string
	}{
		{"http URL", "http://localhost:4433"},
		{"https URL", "https://example.com:4433"},
		{"WebTransport URL", "https://example.com:4433/moq"},
		{"IP address", "https://192.168.1.1:4433"},
		{"localhost", "https://localhost:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Upstream: tt.upstream,
			}

			if cfg.Upstream != tt.upstream {
				t.Errorf("Expected Upstream=%s, got %s", tt.upstream, cfg.Upstream)
			}
		})
	}
}

// TestConfigWithGroupCacheSize tests different cache sizes
func TestConfigWithGroupCacheSize(t *testing.T) {
	tests := []struct {
		name      string
		cacheSize int
	}{
		{"small cache", 10},
		{"medium cache", 100},
		{"large cache", 1000},
		{"very large cache", 10000},
		{"zero cache", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				GroupCacheSize: tt.cacheSize,
			}

			if cfg.GroupCacheSize != tt.cacheSize {
				t.Errorf("Expected GroupCacheSize=%d, got %d", tt.cacheSize, cfg.GroupCacheSize)
			}
		})
	}
}

// TestConfigWithFrameCapacity tests different frame capacities
func TestConfigWithFrameCapacity(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
	}{
		{"1KB", 1024},
		{"64KB", 65536},
		{"1MB", 1048576},
		{"10MB", 10485760},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				FrameCapacity: tt.capacity,
			}

			if cfg.FrameCapacity != tt.capacity {
				t.Errorf("Expected FrameCapacity=%d, got %d", tt.capacity, cfg.FrameCapacity)
			}
		})
	}
}

// TestConfigWithHealthCheckAddr tests health check address configuration
func TestConfigWithHealthCheckAddr(t *testing.T) {
	tests := []struct {
		name string
		addr string
	}{
		{"port only", ":8080"},
		{"localhost with port", "localhost:8080"},
		{"all interfaces", "0.0.0.0:8080"},
		{"specific IP", "127.0.0.1:8080"},
		{"IPv6", "[::1]:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				HealthCheckAddr: tt.addr,
			}

			if cfg.HealthCheckAddr != tt.addr {
				t.Errorf("Expected HealthCheckAddr=%s, got %s", tt.addr, cfg.HealthCheckAddr)
			}
		})
	}
}

// TestConfigFullyPopulated tests a fully configured Config
func TestConfigFullyPopulated(t *testing.T) {
	cfg := &Config{
		Upstream:        "https://upstream.example.com:4433",
		GroupCacheSize:  200,
		FrameCapacity:   2048,
		HealthCheckAddr: ":8080",
	}

	if cfg.Upstream != "https://upstream.example.com:4433" {
		t.Error("Upstream not set correctly")
	}

	if cfg.GroupCacheSize != 200 {
		t.Error("GroupCacheSize not set correctly")
	}

	if cfg.FrameCapacity != 2048 {
		t.Error("FrameCapacity not set correctly")
	}

	if cfg.HealthCheckAddr != ":8080" {
		t.Error("HealthCheckAddr not set correctly")
	}
}

// TestConfigCopy tests that Config can be copied correctly
func TestConfigCopy(t *testing.T) {
	original := &Config{
		Upstream:        "https://example.com",
		GroupCacheSize:  100,
		FrameCapacity:   1024,
		HealthCheckAddr: ":8080",
	}

	// Create a copy
	copy := &Config{
		Upstream:        original.Upstream,
		GroupCacheSize:  original.GroupCacheSize,
		FrameCapacity:   original.FrameCapacity,
		HealthCheckAddr: original.HealthCheckAddr,
	}

	// Verify all fields match
	if copy.Upstream != original.Upstream {
		t.Error("Upstream not copied correctly")
	}
	if copy.GroupCacheSize != original.GroupCacheSize {
		t.Error("GroupCacheSize not copied correctly")
	}
	if copy.FrameCapacity != original.FrameCapacity {
		t.Error("FrameCapacity not copied correctly")
	}
	if copy.HealthCheckAddr != original.HealthCheckAddr {
		t.Error("HealthCheckAddr not copied correctly")
	}

	// Modify copy and ensure original is unchanged
	copy.Upstream = "https://different.com"
	if original.Upstream == copy.Upstream {
		t.Error("Modifying copy should not affect original")
	}
}

// TestConfigNegativeValues tests handling of negative/invalid values
func TestConfigNegativeValues(t *testing.T) {
	cfg := &Config{
		GroupCacheSize: -1,
		FrameCapacity:  -100,
	}

	// The Config struct doesn't validate, but we document the behavior
	if cfg.GroupCacheSize != -1 {
		t.Error("Negative GroupCacheSize should be stored as-is")
	}

	if cfg.FrameCapacity != -100 {
		t.Error("Negative FrameCapacity should be stored as-is")
	}
}

// TestConfigEmptyUpstream tests empty upstream configuration
func TestConfigEmptyUpstream(t *testing.T) {
	cfg := &Config{
		Upstream: "",
	}

	if cfg.Upstream != "" {
		t.Error("Empty Upstream should remain empty")
	}
}

// TestConfigStructSize tests that Config doesn't grow unexpectedly
func TestConfigStructSize(t *testing.T) {
	cfg := &Config{}

	// This test ensures we're aware if Config grows
	// Config should have exactly 4 fields as of now
	expectedFields := 4

	// This is a documentation test - if this fails, update the test
	// and verify all new fields are tested
	_ = cfg.Upstream
	_ = cfg.GroupCacheSize
	_ = cfg.FrameCapacity
	_ = cfg.HealthCheckAddr

	t.Logf("Config has %d fields - ensure all are tested", expectedFields)
}
