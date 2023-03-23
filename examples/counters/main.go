/*
Example of using mcache to build ever incrementing counter with a default TTL
which gets updated every time a counter is incremented. It is thread-safe.

It will return 0 for non-existing keys.
*/
package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/dmytro-vovk/go-mcache"
)

func main() {
	c := NewCounter(time.Hour)

	c.Inc("a")
	c.Inc("a")
	c.Inc("b")
	c.Inc("a")

	fmt.Printf("a: %d\n", c.Get("a"))
	fmt.Printf("b: %d\n", c.Get("b"))
	fmt.Printf("c: %d\n", c.Get("c"))
	/*
		Output:
		a: 3
		b: 1
		c: 0
	*/
}

type Counter struct {
	c   *mcache.Cache[string, int]
	ttl time.Duration
	m   sync.Mutex
}

func NewCounter(ttl time.Duration) *Counter {
	return &Counter{
		c:   mcache.New[string, int](),
		ttl: ttl,
	}
}

func (c *Counter) Inc(key string) {
	c.m.Lock()

	value, _ := c.c.Get(key)
	c.c.Set(key, value+1, c.ttl)

	c.m.Unlock()
}

func (c *Counter) Get(key string) int {
	c.m.Lock()

	value, _ := c.c.Get(key)

	c.m.Unlock()

	return value
}
