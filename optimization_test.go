package mcache_test

import (
	"testing"
	"time"

	mcache "github.com/dmytro-vovk/go-mcache"
)

// NewWithSize must return a fully functional cache that behaves like New().
func TestNewWithSize(t *testing.T) {
	c := mcache.NewWithSize[int, int](128)

	c.Set(1, 11, time.Hour)
	c.Set(2, 22, time.Hour)

	if v, ok := c.Get(1); !ok || v != 11 {
		t.Fatalf("Get(1) = %v,%v; want 11,true", v, ok)
	}
	if v, ok := c.Get(2); !ok || v != 22 {
		t.Fatalf("Get(2) = %v,%v; want 22,true", v, ok)
	}
	if !c.Delete(1) {
		t.Fatalf("Delete(1) = false; want true")
	}
	if _, ok := c.Get(1); ok {
		t.Fatalf("Get(1) after delete returned ok=true")
	}
	if c.Len() != 1 {
		t.Fatalf("Len() = %d; want 1", c.Len())
	}
}

// After a node has been freed (via Delete), the next Set must reuse it
// instead of allocating a fresh node. Steady-state Set+Delete churn must
// therefore allocate zero objects.
func TestSetReusesFreedNodes(t *testing.T) {
	c := mcache.New[int, int]()

	// Prime the free list.
	c.Set(1, 1, time.Hour)
	c.Delete(1)

	allocs := testing.AllocsPerRun(200, func() {
		c.Set(1, 1, time.Hour)
		if v, ok := c.Get(1); !ok || v != 1 {
			t.Fatalf("round-trip failed: %v,%v", v, ok)
		}
		c.Delete(1)
	})

	if allocs != 0 {
		t.Fatalf("Set+Delete churn allocated %v objects/op; want 0 (free-list reuse)", allocs)
	}
}

// The Set-replace path (overwriting an existing key) must also reuse the
// freed node, since Set internally deletes then re-inserts.
func TestSetReplaceReusesFreedNodes(t *testing.T) {
	c := mcache.New[int, int]()
	c.Set(1, 1, time.Hour) // warm-up alloc happens here

	allocs := testing.AllocsPerRun(200, func() {
		c.Set(1, 1, time.Hour) // replace existing key
	})

	if allocs != 0 {
		t.Fatalf("Set-replace allocated %v objects/op; want 0 (free-list reuse)", allocs)
	}
}

// Analog of BenchmarkCacheSet but using a map pre-sized to the run length,
// isolating the NewWithSize win (no incremental map growth/rehashing).
func BenchmarkSetPresized(b *testing.B) {
	c := mcache.NewWithSize[int, int](b.N)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Set(i, i, time.Hour)
	}
}
