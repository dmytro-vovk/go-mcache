package mcache_test

import (
	"testing"
	"time"

	mcache "github.com/dmytro-vovk/go-mcache"
)

// Baseline analog of BenchmarkCacheSet but with a TTL longer than any run,
// so the expiry timer never fires. Map still grows to b.N entries.
// If this is fast + stable vs BenchmarkCacheSet -> timer contention (H1).
func BenchmarkSetLongTTL(b *testing.B) {
	c := mcache.New[int, int]()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Set(i, i, time.Hour)
	}
}

// Long TTL + periodic reset so the map stays small (<= window entries).
// If this is faster than BenchmarkSetLongTTL -> map growth/GC (H2/H3).
func BenchmarkSetLongTTLBoundedMap(b *testing.B) {
	const window = 4096

	c := mcache.New[int, int]()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if i%window == 0 {
			b.StopTimer()
			c = mcache.New[int, int]()
			b.StartTimer()
		}

		c.Set(i, i, time.Hour)
	}
}

// Steady-state: fixed working set, keys replaced (Set on existing key path).
func BenchmarkSetReplaceSteadyState(b *testing.B) {
	const window = 4096

	c := mcache.New[int, int]()
	for i := 0; i < window; i++ {
		c.Set(i, i, time.Hour)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Set(i%window, i, time.Hour)
	}
}
