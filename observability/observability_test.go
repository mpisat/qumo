package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestConfig_ZeroValue(t *testing.T) {
	// Zero value should disable all features
	var cfg Config
	if cfg.Service != "" {
		t.Error("expected empty service")
	}
	if cfg.TraceAddr != "" {
		t.Error("expected empty trace addr")
	}
	if cfg.LogAddr != "" {
		t.Error("expected empty log addr")
	}
	if cfg.Metrics {
		t.Error("expected metrics disabled by default")
	}
}

func TestSetup_NoConfig(t *testing.T) {
	ctx := context.Background()

	// Setup with zero config should succeed (noop mode)
	err := Setup(ctx, Config{})
	if err != nil {
		t.Fatalf("Setup with zero config failed: %v", err)
	}
	defer Shutdown(ctx)

	// Should report disabled
	if Enabled() {
		t.Error("expected tracing disabled")
	}
	if MetricsEnabled() {
		t.Error("expected metrics disabled")
	}
}

func TestSetup_MetricsOnly(t *testing.T) {
	ctx := context.Background()

	err := Setup(ctx, Config{
		Service: "test-service",
		Metrics: true,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer Shutdown(ctx)

	if Enabled() {
		t.Error("expected tracing disabled")
	}
	if !MetricsEnabled() {
		t.Error("expected metrics enabled")
	}
}

func TestStart_NoTracer(t *testing.T) {
	ctx := context.Background()

	// Setup without tracing
	err := Setup(ctx, Config{Service: "test"})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer Shutdown(ctx)

	// Start should still work (noop span)
	ctx2, span := Start(ctx, "test-operation")
	if ctx2 == nil {
		t.Error("expected non-nil context")
	}
	if span == nil {
		t.Error("expected non-nil span")
	}

	// End should not panic
	span.End()
}

func TestSpan_Error(t *testing.T) {
	ctx := context.Background()

	err := Setup(ctx, Config{Service: "test"})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer Shutdown(ctx)

	_, span := Start(ctx, "test-operation")

	// Error should not panic even without tracer
	span.Error(nil, "test error")
}

func TestSpan_Event(t *testing.T) {
	ctx := context.Background()

	err := Setup(ctx, Config{Service: "test"})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer Shutdown(ctx)

	_, span := Start(ctx, "test-operation")

	// Event should not panic
	span.Event("test-event", Track("video"))
	span.End()
}

func TestSpan_Set(t *testing.T) {
	ctx := context.Background()

	err := Setup(ctx, Config{Service: "test"})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer Shutdown(ctx)

	_, span := Start(ctx, "test-operation")

	// Set should not panic
	span.Set(Track("video"), Group(42))
	span.End()
}

func TestStartWith_Options(t *testing.T) {
	ctx := context.Background()

	err := Setup(ctx, Config{Service: "test"})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer Shutdown(ctx)

	started := false
	ended := false

	ctx2, span := StartWith(ctx, "test-operation",
		Attrs(Track("video")),
		OnStart(func() { started = true }),
		OnEnd(func() { ended = true }),
	)

	if ctx2 == nil {
		t.Error("expected non-nil context")
	}
	if !started {
		t.Error("expected OnStart to be called")
	}
	if ended {
		t.Error("expected OnEnd not called yet")
	}

	span.End()

	if !ended {
		t.Error("expected OnEnd to be called")
	}
}

func TestAttributes(t *testing.T) {
	tests := []struct {
		name     string
		attr     attribute.KeyValue
		wantKey  string
		wantType string
	}{
		{"Track", Track("video"), "moq.track", "STRING"},
		{"Group", Group(42), "moq.group", "INT64"},
		{"GroupSequence", GroupSequence(100), "moq.group", "INT64"},
		{"Frames", Frames(10), "moq.frames", "INT64"},
		{"Broadcast", Broadcast("/live"), "moq.broadcast", "STRING"},
		{"Subscribers", Subscribers(5), "moq.subscribers", "INT64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.attr.Key) != tt.wantKey {
				t.Errorf("key = %s, want %s", tt.attr.Key, tt.wantKey)
			}
			if tt.attr.Value.Type().String() != tt.wantType {
				t.Errorf("type = %s, want %s", tt.attr.Value.Type().String(), tt.wantType)
			}
		})
	}
}

func TestStr_Num(t *testing.T) {
	s := Str("custom.key", "value")
	if string(s.Key) != "custom.key" {
		t.Errorf("Str key = %s, want custom.key", s.Key)
	}
	if s.Value.AsString() != "value" {
		t.Errorf("Str value = %s, want value", s.Value.AsString())
	}

	n := Num("custom.num", 123)
	if string(n.Key) != "custom.num" {
		t.Errorf("Num key = %s, want custom.num", n.Key)
	}
	if n.Value.AsInt64() != 123 {
		t.Errorf("Num value = %d, want 123", n.Value.AsInt64())
	}
}
