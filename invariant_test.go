package mcache

import (
	"math/rand"
	"testing"
	"time"
)

// checkInvariants verifies the internal ordered queue is consistent with the
// map: same size, doubly-linked both ways, sorted ascending by Expires, and
// every map entry's Ptr is the queue node carrying the matching key.
func checkInvariants(t *testing.T, c *Cache[int, int]) {
	t.Helper()

	c.m.Lock()
	defer c.m.Unlock()

	if (c.head == nil) != (c.tail == nil) {
		t.Fatalf("head/tail nil mismatch: head=%v tail=%v", c.head, c.tail)
	}
	if c.head != nil && c.head.Prev != nil {
		t.Fatalf("head.Prev must be nil")
	}
	if c.tail != nil && c.tail.Next != nil {
		t.Fatalf("tail.Next must be nil")
	}

	seen := map[*item[int]]bool{}
	var prev *item[int]
	n := 0
	for it := c.head; it != nil; it = it.Next {
		if seen[it] {
			t.Fatalf("cycle / duplicate node for key %v", it.Key)
		}
		seen[it] = true

		if it.Prev != prev {
			t.Fatalf("broken Prev link at key %v", it.Key)
		}
		if prev != nil && it.Expires.Before(prev.Expires) {
			t.Fatalf("queue not sorted: %v(%v) after %v(%v)",
				it.Key, it.Expires, prev.Key, prev.Expires)
		}

		vp, ok := c.cache[it.Key]
		if !ok {
			t.Fatalf("queue node key %v missing from map", it.Key)
		}
		if vp.Ptr != it {
			t.Fatalf("map[%v].Ptr does not point to its queue node", it.Key)
		}

		prev = it
		n++
	}

	if n != len(c.cache) {
		t.Fatalf("queue length %d != map length %d", n, len(c.cache))
	}
	if c.tail != prev {
		t.Fatalf("tail is not the last walked node")
	}
}

func TestQueueInvariantsFuzz(t *testing.T) {
	for _, seed := range []int64{1, 2, 3, 42, 1337} {
		r := rand.New(rand.NewSource(seed))
		c := New[int, int]()

		// Long TTLs so the background timer never fires mid-fuzz; this
		// isolates the Set/Refresh/Rekey/Delete queue logic.
		ttl := func() time.Duration {
			return time.Duration(1+r.Intn(1000)) * time.Second
		}
		key := func() int { return r.Intn(40) }

		for i := 0; i < 20000; i++ {
			switch r.Intn(7) {
			case 0, 1:
				c.Set(key(), i, ttl())
			case 2:
				c.Refresh(key(), ttl())
			case 3:
				c.Delete(key())
			case 4:
				c.Rekey(key(), key())
			case 5:
				c.Swap(key(), i)
			case 6:
				c.GetAndDelete(key())
			}
			checkInvariants(t, c)
		}

		// Drain everything via the real expiry timer and verify it empties.
		c.m.Lock()
		for it := c.head; it != nil; it = it.Next {
			it.Expires = time.Now().Add(20 * time.Millisecond)
		}
		if c.head != nil {
			c.setTimer()
		}
		c.m.Unlock()

		deadline := time.Now().Add(time.Second)
		for c.Len() != 0 && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
		if l := c.Len(); l != 0 {
			t.Fatalf("seed %d: cache did not drain, %d left", seed, l)
		}
		checkInvariants(t, c)
	}
}
