package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthChecker_GetStatus(t *testing.T) {
	checker := NewStatusHandler()

	status := checker.GetStatus()

	if status.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", status.Status)
	}
	if status.ActiveConnections != 0 {
		t.Errorf("Expected 0 active connections, got %d", status.ActiveConnections)
	}
	if status.Version != "v0.1.0" {
		t.Errorf("Expected version 'v0.1.0', got '%s'", status.Version)
	}
}

func TestHealthChecker_ConnectionTracking(t *testing.T) {
	checker := NewStatusHandler()

	// Test increment
	checker.IncrementConnections()
	checker.IncrementConnections()
	checker.IncrementConnections()

	status := checker.GetStatus()
	if status.ActiveConnections != 3 {
		t.Errorf("Expected 3 active connections, got %d", status.ActiveConnections)
	}

	// Test decrement
	checker.DecrementConnections()
	status = checker.GetStatus()
	if status.ActiveConnections != 2 {
		t.Errorf("Expected 2 active connections, got %d", status.ActiveConnections)
	}
}

func TestHealthChecker_UpstreamStatus(t *testing.T) {
	checker := NewStatusHandler()

	// Initially not connected
	status := checker.GetStatus()
	if status.UpstreamConnected {
		t.Error("Expected upstream not connected initially")
	}
	if status.Status != "healthy" {
		t.Errorf("Expected status 'healthy' without upstream required, got '%s'", status.Status)
	}

	// Mark as connected
	checker.SetUpstreamConnected(true)
	status = checker.GetStatus()
	if !status.UpstreamConnected {
		t.Error("Expected upstream connected after setting")
	}
	if status.Status != "healthy" {
		t.Errorf("Expected status 'healthy' with upstream connected, got '%s'", status.Status)
	}

	// Test degraded status when upstream required but not connected
	checker.SetUpstreamRequired(true)
	checker.SetUpstreamConnected(false)
	status = checker.GetStatus()
	if status.Status != "degraded" {
		t.Errorf("Expected status 'degraded' when upstream required but not connected, got '%s'", status.Status)
	}
}

func TestHealthChecker_HTTPEndpoint(t *testing.T) {
	checker := NewStatusHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	checker.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var status Status
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if status.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", status.Status)
	}
}

func TestHealthChecker_HEADRequest(t *testing.T) {
	checker := NewStatusHandler()

	req := httptest.NewRequest(http.MethodHead, "/health", nil)
	w := httptest.NewRecorder()

	checker.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// HEAD should have empty body
	if w.Body.Len() != 0 {
		t.Errorf("Expected empty body for HEAD request, got %d bytes", w.Body.Len())
	}
}

func TestHealthChecker_InvalidMethod(t *testing.T) {
	checker := NewStatusHandler()

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	checker.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code 405, got %d", w.Code)
	}
}

func TestHealthChecker_Uptime(t *testing.T) {
	checker := NewStatusHandler()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	status := checker.GetStatus()

	if status.Uptime == "" {
		t.Error("Expected non-empty uptime")
	}

	// Check that timestamp is recent
	if time.Since(status.Timestamp) > time.Second {
		t.Error("Timestamp should be recent")
	}
}

func TestHealthChecker_ConcurrentAccess(t *testing.T) {
	checker := NewStatusHandler()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				checker.IncrementConnections()
				_ = checker.GetStatus()
				checker.DecrementConnections()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// All increments and decrements should cancel out
	status := checker.GetStatus()
	if status.ActiveConnections != 0 {
		t.Errorf("Expected 0 connections after concurrent operations, got %d", status.ActiveConnections)
	}
}

func TestHealthChecker_LivenessProbe(t *testing.T) {
	checker := NewStatusHandler()

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	checker.ServeLive(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "alive" {
		t.Errorf("Expected status 'alive', got '%s'", response["status"])
	}
}

func TestHealthChecker_ReadinessProbe_Ready(t *testing.T) {
	checker := NewStatusHandler()
	checker.SetUpstreamConnected(true)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	checker.ServeReady(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if ready, ok := response["ready"].(bool); !ok || !ready {
		t.Errorf("Expected ready=true, got %v", response["ready"])
	}
}

func TestHealthChecker_ReadinessProbe_NotReady_UpstreamRequired(t *testing.T) {
	checker := NewStatusHandler()
	checker.SetUpstreamRequired(true)
	checker.SetUpstreamConnected(false)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()

	checker.ServeReady(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status code 503, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if ready, ok := response["ready"].(bool); !ok || ready {
		t.Errorf("Expected ready=false, got %v", response["ready"])
	}

	if reason, ok := response["reason"].(string); !ok || reason != "upstream_not_connected" {
		t.Errorf("Expected reason='upstream_not_connected', got %v", response["reason"])
	}
}

func TestHealthChecker_ReadinessProbe_HEADRequest(t *testing.T) {
	checker := NewStatusHandler()

	req := httptest.NewRequest(http.MethodHead, "/health/ready", nil)
	w := httptest.NewRecorder()

	checker.ServeReady(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	if w.Body.Len() != 0 {
		t.Errorf("Expected empty body for HEAD request, got %d bytes", w.Body.Len())
	}
}

func TestHealthChecker_JSONFormat(t *testing.T) {
	checker := NewStatusHandler()
	checker.IncrementConnections()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	checker.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	var status Status
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Verify all fields are present
	if status.Status == "" {
		t.Error("Status field missing")
	}
	if status.Timestamp.IsZero() {
		t.Error("Timestamp field missing")
	}
	if status.Uptime == "" {
		t.Error("Uptime field missing")
	}
	if status.Version == "" {
		t.Error("Version field missing")
	}
}
