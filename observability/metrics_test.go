package observability

import (
	"testing"
	"time"
)

func TestRecorder_New(t *testing.T) {
	rec := NewRecorder("video")
	if rec == nil {
		t.Fatal("expected non-nil recorder")
	}
	if rec.track != "video" {
		t.Errorf("track = %s, want video", rec.track)
	}
}

func TestRecorder_Methods(t *testing.T) {
	// Setup with metrics enabled
	err := Setup(t.Context(), Config{
		Service: "test",
		Metrics: true,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer Shutdown(t.Context())

	rec := NewRecorder("test-track")

	// These should not panic
	rec.GroupReceived()
	rec.CacheHit()
	rec.CacheMiss()
	rec.Catchup(5)
	rec.IncSubscribers()
	rec.DecSubscribers()
	rec.SetSubscribers(10)
	rec.Broadcast(time.Millisecond, 10, 8)
}

func TestRecorder_LatencyObs(t *testing.T) {
	err := Setup(t.Context(), Config{
		Service: "test",
		Metrics: true,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer Shutdown(t.Context())

	rec := NewRecorder("test-track")

	obs := rec.LatencyObs("receive")
	if obs == nil {
		t.Error("expected non-nil observer when metrics enabled")
	}

	// Should not panic
	obs.Observe(0.001)
}

func TestRecorder_MetricsDisabled(t *testing.T) {
	err := Setup(t.Context(), Config{
		Service: "test",
		Metrics: false,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer Shutdown(t.Context())

	rec := NewRecorder("test-track")

	// All methods should be safe to call when metrics disabled
	rec.GroupReceived()
	rec.CacheHit()
	rec.CacheMiss()
	rec.Catchup(5)
	rec.IncSubscribers()
	rec.DecSubscribers()
	rec.SetSubscribers(10)
	rec.Broadcast(time.Millisecond, 10, 8)

	// LatencyObs returns nil when disabled
	obs := rec.LatencyObs("receive")
	if obs != nil {
		t.Error("expected nil observer when metrics disabled")
	}
}

func TestGlobalMetrics(t *testing.T) {
	err := Setup(t.Context(), Config{
		Service: "test",
		Metrics: true,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer Shutdown(t.Context())

	// These should not panic
	IncTracks()
	DecTracks()
}
