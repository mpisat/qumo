package relay

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// Status represents the health status of the relay server
type Status struct {
	Status            string    `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp         time.Time `json:"timestamp"`
	Uptime            string    `json:"uptime"`
	ActiveConnections int32     `json:"active_connections"`
	UpstreamConnected bool      `json:"upstream_connected"`
}

// statusHandler manages health check state
type statusHandler struct {
	startTime         time.Time
	activeConnections atomic.Int32
	upstreamConnected atomic.Bool
	upstreamRequired  bool // Whether upstream connection is required for readiness
}

// newStatusHandler creates a new health checker
func newStatusHandler() *statusHandler {
	return &statusHandler{
		startTime:        time.Now(),
		upstreamRequired: false, // Default: upstream not required
	}
}

// SetUpstreamRequired sets whether upstream connection is required for readiness
func (h *statusHandler) SetUpstreamRequired(required bool) {
	h.upstreamRequired = required
}

// IncrementConnections increments the active connection count
func (h *statusHandler) incrementConnections() {
	if h == nil {
		return
	}
	h.activeConnections.Add(1)
}

// DecrementConnections decrements the active connection count
func (h *statusHandler) decrementConnections() {
	if h == nil {
		return
	}
	h.activeConnections.Add(-1)
}

// GetStatus returns the current health status
func (h *statusHandler) getStatus() Status {
	if h == nil {
		return Status{}
	}

	uptime := time.Since(h.startTime)
	activeConns := h.activeConnections.Load()
	upstreamConn := h.upstreamConnected.Load()

	// Determine overall status
	status := "healthy"

	// Check for unhealthy conditions
	if activeConns < 0 {
		status = "unhealthy" // Should never happen, but defensive
	} else if h.upstreamRequired && !upstreamConn {
		// If upstream is required but not connected, mark as degraded
		status = "degraded"
	}

	return Status{
		Status:            status,
		Timestamp:         time.Now(),
		Uptime:            uptime.String(),
		ActiveConnections: activeConns,
		UpstreamConnected: upstreamConn,
	}
}

// ServeHTTP implements http.Handler for health check endpoint
func (h *statusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	status := h.getStatus()

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
}
