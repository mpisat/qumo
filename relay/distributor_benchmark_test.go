package relay

import (
	"testing"
)

// ============================================================================
// trackDistributor Benchmark Tests
// ============================================================================

// BenchmarkBroadcast_10 benchmarks with 10 subscribers
func BenchmarkBroadcast_10(b *testing.B) {
	benchmarkBroadcast(b, 10)
}

// BenchmarkBroadcast_100 benchmarks with 100 subscribers
func BenchmarkBroadcast_100(b *testing.B) {
	benchmarkBroadcast(b, 100)
}

// BenchmarkBroadcast_500 benchmarks with 500 subscribers
func BenchmarkBroadcast_500(b *testing.B) {
	benchmarkBroadcast(b, 500)
}

func benchmarkBroadcast(b *testing.B, numSubscribers int) {
	dist := newTestDistributor()

	for i := 0; i < numSubscribers; i++ {
		dist.subscribe()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broadcastToSubscribers(dist)
	}
}

// BenchmarkSubscribe benchmarks subscription operations
func BenchmarkSubscribe(b *testing.B) {
	dist := newTestDistributor()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch := dist.subscribe()
		dist.unsubscribe(ch)
	}
}

// BenchmarkDistributorVariableLoad benchmarks with varying subscriber activity
func BenchmarkDistributorVariableLoad(b *testing.B) {
	dist := newTestDistributor()

	// 50% active subscribers
	const totalSubs = 100
	for i := 0; i < totalSubs; i++ {
		ch := dist.subscribe()
		if i%2 == 0 {
			// Active subscriber - drain channel
			go func() {
				for range ch {
				}
			}()
		}
		// Passive subscribers don't drain
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broadcastToSubscribers(dist)
	}
}
