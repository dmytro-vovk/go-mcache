package mcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestCache(t *testing.T) {
	c := New[int, int]()

	assert.Zero(t, c.Len())

	c.Delete(0)

	c.Set(3, 3, 300*time.Millisecond)

	c.Delete(3)

	c.Set(3, 3, 300*time.Millisecond)
	c.Set(2, 2, 200*time.Millisecond)
	c.Set(10, 10, 1000*time.Millisecond)
	c.Set(5, 5, 500*time.Millisecond)
	c.Set(1, 1, 100*time.Millisecond)
	c.Delete(3)
	c.Set(7, 7, 700*time.Millisecond)

	c.Delete(10)
	if v, ok := c.Get(1); assert.True(t, ok) {
		assert.Equal(t, 1, v)
	}

	assert.Eventually(t, func() bool {
		_, ok := c.Get(1)
		return !ok
	}, 130*time.Millisecond, 20*time.Millisecond)

	assert.Eventually(t, func() bool {
		return 0 == c.Len()
	}, 1100*time.Millisecond, 20*time.Millisecond)
}

func TestReplace(t *testing.T) {
	c := New[int, string]()

	c.Set(1, "foo", 50*time.Millisecond)
	if v, ok := c.Get(1); assert.True(t, ok) {
		assert.Equal(t, "foo", v)
	}

	c.Set(1, "bar", 50*time.Millisecond)
	if v, ok := c.Get(1); assert.True(t, ok) {
		assert.Equal(t, "bar", v)
	}

	assert.Eventually(t, func() bool {
		return 0 == c.Len()
	}, 100*time.Millisecond, 20*time.Millisecond)
}

func TestDelete(t *testing.T) {
	c := New[int, int]()

	c.Set(1, 1, 50*time.Millisecond)
	c.Set(2, 2, 50*time.Millisecond)
	c.Set(3, 3, 50*time.Millisecond)
	c.Set(4, 4, 50*time.Millisecond)

	_, ok := c.Get(3)
	assert.True(t, ok)

	assert.True(t, c.Delete(3))

	_, ok = c.Get(3)
	assert.False(t, ok)

	assert.True(t, c.Delete(1))

	_, ok = c.Get(1)
	assert.False(t, ok)

	assert.False(t, c.Delete(49))

	if v, ok := c.GetAndDelete(2); assert.True(t, ok) {
		assert.Equal(t, 2, v)
		_, ok = c.Get(2)
		assert.False(t, ok)
	}

	_, ok = c.GetAndDelete(999)
	assert.False(t, ok)

	assert.Eventually(t, func() bool {
		return 0 == c.Len()
	}, 100*time.Millisecond, 20*time.Millisecond)
}

func TestOrder(t *testing.T) {
	c := New[int, int]()

	c.Set(1, 1, 100*time.Millisecond)
	c.Set(2, 2, 200*time.Millisecond)

	assert.Eventually(t, func() bool {
		return 1 == c.Len()
	}, 300*time.Millisecond, 20*time.Millisecond, "remaining %d", c.Len())

	_, ok := c.Get(1)
	assert.False(t, ok)

	if v, ok := c.Get(2); assert.True(t, ok) {
		assert.Equal(t, 2, v)
	}

	assert.Eventually(t, func() bool {
		return 0 == c.Len()
	}, 300*time.Millisecond, 20*time.Millisecond, "remaining %d", c.Len())
}

func TestOrder2(t *testing.T) {
	c := New[int, int]()

	c.Set(2, 2, 200*time.Millisecond)
	c.Set(1, 1, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		return 1 == c.Len()
	}, 250*time.Millisecond, 20*time.Millisecond, "remaining %d", c.Len())

	_, ok := c.Get(1)
	assert.False(t, ok)

	if v, ok := c.Get(2); assert.True(t, ok) {
		assert.Equal(t, 2, v)
	}

	assert.Eventually(t, func() bool {
		return 0 == c.Len()
	}, 250*time.Millisecond, 20*time.Millisecond, "remaining %d", c.Len())
}

func TestOrder3(t *testing.T) {
	c := New[int, int]()

	c.Set(2, 2, 200*time.Millisecond)
	c.Set(3, 3, 300*time.Millisecond)
	c.Set(1, 1, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		return 2 == c.Len()
	}, 450*time.Millisecond, 20*time.Millisecond, "remaining %d", c.Len())

	_, ok := c.Get(1)
	assert.False(t, ok)

	if v, ok := c.Get(2); assert.True(t, ok) {
		assert.Equal(t, 2, v)
	}

	assert.Eventually(t, func() bool {
		return 0 == c.Len()
	}, 350*time.Millisecond, 20*time.Millisecond, "remaining %d", c.Len())
}

func TestRefresh(t *testing.T) {
	c := New[int, int]()

	c.Set(1, 1, 100*time.Millisecond)
	c.Set(2, 2, 200*time.Millisecond)
	c.Set(3, 3, 300*time.Millisecond)

	assert.True(t, c.Refresh(1, 350*time.Millisecond))

	assert.True(t, c.Refresh(2, 300*time.Millisecond))

	assert.True(t, c.Refresh(2, 100*time.Millisecond))

	assert.True(t, c.Refresh(1, 150*time.Millisecond))

	assert.False(t, c.Refresh(100, time.Millisecond))

	assert.Eventually(t, func() bool {
		return 0 == c.Len()
	}, 3500*time.Millisecond, 20*time.Millisecond)
}

func TestUpdate(t *testing.T) {
	c := New[string, string]()

	c.Set("a", "foo", 20*time.Millisecond)
	assert.True(t, c.Update("a", "bar"))
	assert.False(t, c.Update("x", "bar"))

}

func TestLargeCache(t *testing.T) {
	t.SkipNow()

	c, n, ttl := New[int, int](), 1_000_000, 20*time.Second

	for i := 0; i < n; i++ {
		c.Set(i, i, ttl)
	}

	assert.Equal(t, n, c.Len())

	if v, ok := c.Get(0); assert.True(t, ok) {
		assert.Equal(t, 0, v)
	}

	if v, ok := c.Get(n / 2); assert.True(t, ok) {
		assert.Equal(t, n/2, v)
	}

	if v, ok := c.Get(n - 1); assert.True(t, ok) {
		assert.Equal(t, n-1, v)
	}

	assert.Eventually(t, func() bool {
		return 0 == c.Len()
	}, 21*time.Second, 20*time.Millisecond)
}

func TestSwap(t *testing.T) {
	c := New[int, int]()

	c.Set(1, 100, 50*time.Millisecond)

	if v, ok := c.Swap(1, 222); assert.True(t, ok) {
		assert.Equal(t, 100, v)
	}

	_, ok := c.Swap(19, 19)
	assert.False(t, ok)

	assert.Eventually(t, func() bool {
		_, ok := c.Get(1)

		return !ok
	}, 150*time.Millisecond, 5*time.Millisecond)
}

func TestEvict(t *testing.T) {
	c := New[int, int]()

	c.Set(1, 1, 50*time.Millisecond)
	c.Set(2, 2, 50*time.Millisecond)
	c.Set(3, 3, 50*time.Millisecond)

	require.Equal(t, 3, c.Len())

	require.Equal(t, 2, c.Evict(2))

	require.Equal(t, 1, c.Len())

	require.Equal(t, 1, c.Evict(2))

	require.Equal(t, 0, c.Len())

	c.Set(10, 10, 100*time.Millisecond)
	c.Evict(1)
}

func TestRange(t *testing.T) {
	c := New[int, int]()

	c.Set(1, 1, 50*time.Millisecond)
	c.Set(2, 2, 50*time.Millisecond)
	c.Set(3, 3, 50*time.Millisecond)

	var seen []int
	c.Range(func(k int, v int) bool {
		assert.Equal(t, k, v)
		seen = append(seen, k)
		return true
	})

	require.Equal(t, []int{1, 2, 3}, seen)

	seen = []int{}
	c.Range(func(k int, v int) bool {
		seen = append(seen, k)
		return v != 2
	})

	require.Equal(t, []int{1, 2}, seen)
}

func TestRekey(t *testing.T) {
	c := New[string, bool]()

	c.Set("foo", true, 10*time.Millisecond)
	require.True(t, c.Rekey("foo", "bar"))
	if v, ok := c.Get("bar"); assert.True(t, ok) {
		require.True(t, v)
	}

	_, ok := c.Get("foo")
	require.False(t, ok)

	require.False(t, c.Rekey("non-existing", "new key"))
}

func TestGetMany(t *testing.T) {
	c := New[int, string]()

	c.Set(1, "1", 50*time.Millisecond)
	c.Set(2, "2", 50*time.Millisecond)
	c.Set(3, "3", 50*time.Millisecond)
	c.Set(4, "4", 50*time.Millisecond)
	c.Set(5, "5", 50*time.Millisecond)

	require.Equal(t, map[int]string{1: "1", 3: "3", 5: "5"}, c.GetMany(5, 3, 1))
}

func BenchmarkCacheSet(b *testing.B) {
	c := New[int, int]()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Set(i, i, 50*time.Millisecond)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	c := New[int, int]()

	c.Set(1, 1, 5*time.Second)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if n, ok := c.Get(1); !(ok && n == 1) {
			b.Fail()
		}
	}

	b.StopTimer()
	c.Evict(1)
}

// BenchmarkCacheAddGet-12    	  761462	      1642 ns/op	     281 B/op	       5 allocs/op
// BenchmarkCacheAddGet-12    	  681804	      1619 ns/op	     280 B/op	       5 allocs/op
// BenchmarkCacheAddGet-12    	  794340	      1627 ns/op	     281 B/op	       5 allocs/op
func BenchmarkCacheAddGet(b *testing.B) {
	c := New[int, int]()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Set(i, i, 50*time.Millisecond)

		if n, ok := c.Get(i); !(ok && n == i) {
			b.Fail()
		}

		c.Delete(i)
	}
}
