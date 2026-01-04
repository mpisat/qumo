package relay

import (
	"crypto/tls"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper functions for relay package tests

// newTestServer creates a Server instance for testing with sensible defaults
func newTestServer(opts ...func(*Server)) *Server {
	server := &Server{
		Addr:      "localhost:4433",
		TLSConfig: &tls.Config{},
	}
	for _, opt := range opts {
		opt(server)
	}
	return server
}

// withConfig returns a server option that sets a custom config
func withConfig(config *Config) func(*Server) {
	return func(s *Server) {
		s.Config = config
	}
}

// withAddr returns a server option that sets the address
func withAddr(addr string) func(*Server) {
	return func(s *Server) {
		s.Addr = addr
	}
}

// withoutTLS returns a server option that removes TLS config (for panic tests)
func withoutTLS() func(*Server) {
	return func(s *Server) {
		s.TLSConfig = nil
	}
}

// assertServerInitialized checks that all required server fields are initialized
func assertServerInitialized(t *testing.T, server *Server) {
	t.Helper()
	// Note: Config is not automatically initialized by init(), so we only check TrackMux
	require.NotNil(t, server.TrackMux, "TrackMux should be initialized")
}

// newTestDistributor creates a trackDistributor instance for testing
func newTestDistributor(cacheSize ...int) *trackDistributor {
	size := DefaultGroupCacheSize
	if len(cacheSize) > 0 && cacheSize[0] > 0 {
		size = cacheSize[0]
	}
	return &trackDistributor{
		ring:        newGroupRing(size),
		subscribers: make(map[chan struct{}]struct{}),
	}
}

// broadcastToSubscribers sends a broadcast notification to all subscribers
// This is the common broadcast pattern used throughout tests
func broadcastToSubscribers(dist *trackDistributor) {
	dist.mu.RLock()
	defer dist.mu.RUnlock()
	for ch := range dist.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// waitForBroadcasts waits for a specific number of broadcasts with timeout
func waitForBroadcasts(t *testing.T, ch chan struct{}, count int, timeout time.Duration) int {
	t.Helper()
	received := 0
	timer := time.After(timeout)
	for received < count {
		select {
		case <-ch:
			received++
		case <-timer:
			return received
		}
	}
	return received
}

// waitForBroadcastsAtomic waits for broadcasts and updates an atomic counter
func waitForBroadcastsAtomic(t *testing.T, ch chan struct{}, count int, timeout time.Duration, counter *atomic.Int32) {
	t.Helper()
	received := 0
	timer := time.After(timeout)
	for received < count {
		select {
		case <-ch:
			received++
			counter.Add(1)
		case <-timer:
			return
		}
	}
}

// runConcurrentSubscribers starts multiple goroutines that subscribe and wait for broadcasts
func runConcurrentSubscribers(t *testing.T, dist *trackDistributor, numSubs int, broadcasts int, timeout time.Duration) *atomic.Int32 {
	t.Helper()
	var wg sync.WaitGroup
	received := &atomic.Int32{}

	for i := 0; i < numSubs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch := dist.subscribe()
			defer dist.unsubscribe(ch)
			waitForBroadcastsAtomic(t, ch, broadcasts, timeout, received)
		}()
	}

	// Wait for all subscriptions to be registered
	time.Sleep(10 * time.Millisecond)

	return received
}

// assertBroadcastsReceived verifies that all expected broadcasts were received
func assertBroadcastsReceived(t *testing.T, received *atomic.Int32, expected int) {
	t.Helper()
	actual := int(received.Load())
	assert.Equal(t, expected, actual, "Expected %d broadcasts, received %d", expected, actual)
}

// assertNoPanic verifies that a function does not panic
func assertNoPanic(t *testing.T, fn func(), msgAndArgs ...interface{}) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			assert.Fail(t, "Function panicked unexpectedly", msgAndArgs...)
		}
	}()
	fn()
}

// assertPanics verifies that a function panics
func assertPanics(t *testing.T, fn func(), msgAndArgs ...interface{}) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			assert.Fail(t, "Expected panic but got none", msgAndArgs...)
		}
	}()
	fn()
}
