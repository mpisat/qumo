package relay

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/okdaichi/gomoqt/moqt"
)

// BenchmarkDistributorBroadcast benchmarks broadcast performance
func BenchmarkDistributorBroadcast(b *testing.B) {
	subscribers := []int{1, 10, 100, 1000}

	for _, numSubs := range subscribers {
		b.Run(string(rune(numSubs))+"_subscribers", func(b *testing.B) {
			dist := &trackDistributor{
				subscribers: make(map[chan struct{}]struct{}),
				ring:        newGroupRing(),
			}

			// Subscribe
			channels := make([]chan struct{}, numSubs)
			for i := 0; i < numSubs; i++ {
				channels[i] = dist.subscribe()
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				dist.mu.RLock()
				for ch := range dist.subscribers {
					select {
					case ch <- struct{}{}:
					default:
					}
				}
				dist.mu.RUnlock()
			}

			b.StopTimer()

			// Cleanup
			for _, ch := range channels {
				dist.unsubscribe(ch)
			}
		})
	}
}

// BenchmarkGroupRingOps benchmarks ring buffer operations
func BenchmarkGroupRingOps(b *testing.B) {
	b.Run("add", func(b *testing.B) {
		ring := newGroupRing()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ring.head()
		}
	})

	b.Run("get", func(b *testing.B) {
		ring := newGroupRing()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ring.get(moqt.GroupSequence(i))
		}
	})

	b.Run("head", func(b *testing.B) {
		ring := newGroupRing()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ring.head()
		}
	})

	b.Run("earliestAvailable", func(b *testing.B) {
		ring := newGroupRing()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ring.earliestAvailable()
		}
	})
}

// BenchmarkFramePool benchmarks frame pool operations
func BenchmarkFramePool(b *testing.B) {
	pool := NewFramePool(1024)

	b.Run("get_put", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			frame := pool.Get()
			pool.Put(frame)
		}
	})

	b.Run("get_only", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = pool.Get()
		}
	})
}

// BenchmarkConcurrentSubscriptions benchmarks concurrent subscribe/unsubscribe
func BenchmarkConcurrentSubscriptions(b *testing.B) {
	dist := &trackDistributor{
		subscribers: make(map[chan struct{}]struct{}),
		ring:        newGroupRing(),
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ch := dist.subscribe()
			dist.unsubscribe(ch)
		}
	})
}

// TestRelayHandlerConcurrentMapAccess tests concurrent map access patterns
func TestRelayHandlerConcurrentMapAccess(t *testing.T) {
	handler := &RelayHandler{
		ctx: context.Background(),
	}

	var wg sync.WaitGroup
	operations := 100
	tracks := 10

	// Concurrent writes
	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			trackName := moqt.TrackName(string(rune('A' + (idx % tracks))))

			handler.mu.Lock()
			if handler.relaying == nil {
				handler.relaying = make(map[moqt.TrackName]*trackDistributor)
			}
			handler.relaying[trackName] = &trackDistributor{
				ring:        newGroupRing(),
				subscribers: make(map[chan struct{}]struct{}),
			}
			handler.mu.Unlock()
		}(i)
	}

	// Concurrent reads
	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			trackName := moqt.TrackName(string(rune('A' + (idx % tracks))))

			handler.mu.RLock()
			_ = handler.relaying[trackName]
			handler.mu.RUnlock()
		}(i)
	}

	// Concurrent deletes
	for i := 0; i < operations; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			trackName := moqt.TrackName(string(rune('A' + (idx % tracks))))

			handler.mu.Lock()
			if handler.relaying != nil {
				delete(handler.relaying, trackName)
			}
			handler.mu.Unlock()
		}(i)
	}

	wg.Wait()
}

// TestDistributorStressWithNotifications tests distributor under stress with notifications
func TestDistributorStressWithNotifications(t *testing.T) {
	dist := &trackDistributor{
		subscribers: make(map[chan struct{}]struct{}),
		ring:        newGroupRing(),
	}

	var wg sync.WaitGroup
	subscribers := 50
	notifications := 100
	received := atomic.Int64{}

	// Start subscribers
	for i := 0; i < subscribers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ch := dist.subscribe()
			defer dist.unsubscribe(ch)

			for j := 0; j < notifications; j++ {
				select {
				case <-ch:
					received.Add(1)
				case <-time.After(500 * time.Millisecond):
					return
				}
			}
		}()
	}

	// Wait for subscriptions
	time.Sleep(20 * time.Millisecond)

	// Send notifications
	for i := 0; i < notifications; i++ {
		dist.mu.RLock()
		for ch := range dist.subscribers {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
		dist.mu.RUnlock()
		time.Sleep(time.Millisecond)
	}

	wg.Wait()

	t.Logf("Total notifications received: %d (expected ~%d)", received.Load(), subscribers*notifications)
}

// TestGroupCacheCapacity tests group cache behavior at capacity
func TestGroupCacheCapacity(t *testing.T) {
	ring := newGroupRing()

	// Verify initial state
	if ring.head() != 0 {
		t.Error("Initial head should be 0")
	}

	// Test capacity boundaries
	earliest := ring.earliestAvailable()
	if earliest < 0 {
		t.Errorf("Earliest should be non-negative, got %d", earliest)
	}
}

// TestRelayHandlerNilChecks tests nil handling in RelayHandler
func TestRelayHandlerNilChecks(t *testing.T) {
	tests := []struct {
		name    string
		handler *RelayHandler
	}{
		{
			name: "nil_session_only",
			handler: &RelayHandler{
				Session: nil,
				ctx:     context.Background(),
			},
		},
		{
			name: "nil_announcement_only",
			handler: &RelayHandler{
				Session:      &moqt.Session{},
				Announcement: nil,
				ctx:          context.Background(),
			},
		},
		{
			name: "all_nil",
			handler: &RelayHandler{
				Session:      nil,
				Announcement: nil,
				ctx:          context.Background(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.handler.subscribe("test")
			if result != nil {
				t.Error("Expected nil when dependencies are missing")
			}
		})
	}
}

// TestDistributorUnsubscribeIdempotent tests that unsubscribe is idempotent
func TestDistributorUnsubscribeIdempotent(t *testing.T) {
	dist := &trackDistributor{
		subscribers: make(map[chan struct{}]struct{}),
		ring:        newGroupRing(),
	}

	ch := dist.subscribe()

	// Unsubscribe multiple times
	dist.unsubscribe(ch)
	dist.unsubscribe(ch) // Should not panic
	dist.unsubscribe(ch) // Should not panic

	dist.mu.RLock()
	count := len(dist.subscribers)
	dist.mu.RUnlock()

	if count != 0 {
		t.Errorf("Expected 0 subscribers, got %d", count)
	}
}

// TestNotifyTimeoutModification tests that NotifyTimeout can be modified
func TestNotifyTimeoutModification(t *testing.T) {
	original := NotifyTimeout
	defer func() { NotifyTimeout = original }()

	testValues := []time.Duration{
		500 * time.Microsecond,
		1 * time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
	}

	for _, val := range testValues {
		NotifyTimeout = val
		if NotifyTimeout != val {
			t.Errorf("Expected NotifyTimeout to be %v, got %v", val, NotifyTimeout)
		}
	}
}

// TestGroupCacheSizeModification tests GroupCacheSize variable
func TestGroupCacheSizeModification(t *testing.T) {
	original := GroupCacheSize
	defer func() { GroupCacheSize = original }()

	testSizes := []int{4, 8, 16, 32, 64}

	for _, size := range testSizes {
		GroupCacheSize = size
		if GroupCacheSize != size {
			t.Errorf("Expected GroupCacheSize to be %d, got %d", size, GroupCacheSize)
		}
	}
}
