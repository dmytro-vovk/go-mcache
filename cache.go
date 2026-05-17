package mcache

import (
	"sync"
	"time"
)

type Cache[K comparable, V any] struct {
	cache map[K]valuePtr[K, V] // Cached items
	head  *item[K]             // The earliest item to evict, head of the queue
	tail  *item[K]             // The latest item to evict
	timer *time.Timer          // Single expiry timer for the head item
	m     sync.RWMutex
}

type valuePtr[K comparable, V any] struct {
	Value V        // The value that we cache
	Ptr   *item[K] // Pointer to the node in the ordered queue for fast access
}

// ordered queue item
type item[K comparable] struct {
	Prev    *item[K]
	Next    *item[K]
	Key     K
	Expires time.Time
}

// New creates a news cache instance, using any comparable type for keys, and any type for values.
func New[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		cache: make(map[K]valuePtr[K, V]),
	}
}

// Set adds or replaces a value with key and given TTL.
func (c *Cache[K, V]) Set(key K, value V, ttl time.Duration) {
	c.m.Lock()

	if _, ok := c.cache[key]; ok {
		// We are replacing the item
		c.delete(key)
	}

	i := &item[K]{
		Key:     key,
		Expires: time.Now().Add(ttl),
	}

	c.cache[key] = valuePtr[K, V]{
		Value: value,
		Ptr:   i,
	}

	if c.head == nil {
		c.head = i
		c.tail = i

		c.setTimer()

		c.m.Unlock()

		return
	}

	// Start from the tail, it is the most likely new item will have TTL past the last existing item
	for n := c.tail; ; n = n.Prev {
		if n.Expires.Before(i.Expires) {
			c.insertAfter(i, n)

			break
		}
		// The new item is the earliest to evict
		if n.Prev == nil {
			c.insertBefore(i, n)
			c.setTimer()

			break
		}
	}

	c.m.Unlock()
}

// Get returns value and true, if key exists, of zero value and false if not found.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.m.RLock()

	value, ok := c.cache[key]

	c.m.RUnlock()

	return value.Value, ok
}

// GetMany returns key/value pairs as a map. Will not return non-existing keys/expired values.
func (c *Cache[K, V]) GetMany(keys ...K) map[K]V {
	values := make(map[K]V)

	c.m.RLock()

	for k := range keys {
		if v, ok := c.cache[keys[k]]; ok {
			values[keys[k]] = v.Value
		}
	}

	c.m.RUnlock()

	return values
}

// Swap sets the new value returning the old one. Will return false if key is not found.
func (c *Cache[K, V]) Swap(key K, value V) (V, bool) {
	c.m.Lock()

	v, ok := c.cache[key]
	if !ok {
		c.m.Unlock()

		return value, false
	}

	oldValue := v.Value

	c.cache[key] = valuePtr[K, V]{
		Value: value,
		Ptr:   v.Ptr,
	}

	c.m.Unlock()

	return oldValue, true
}

// Delete removes value from thr cache.
func (c *Cache[K, V]) Delete(key K) (ok bool) {
	c.m.Lock()

	timerResetNeeded := c.head != nil && c.head.Key == key

	ok = c.delete(key)

	if c.head != nil {
		if timerResetNeeded {
			c.setTimer()
		}
	} else if ok && c.timer != nil {
		c.timer.Stop()
	}

	c.m.Unlock()

	return
}

// GetAndDelete returns value and true, and deletes the key if it was found, of zero value and false if the key not found.
func (c *Cache[K, V]) GetAndDelete(key K) (V, bool) {
	c.m.Lock()

	value, ok := c.cache[key]
	if !ok {
		c.m.Unlock()

		return value.Value, false
	}

	c.delete(key)

	c.m.Unlock()

	return value.Value, true
}

// Update sets new value for key without changing TTL, returning false if key not found.
func (c *Cache[K, V]) Update(key K, value V) bool {
	c.m.Lock()

	v, ok := c.cache[key]
	if !ok {
		c.m.Unlock()

		return false
	}

	c.cache[key] = valuePtr[K, V]{
		Value: value,
		Ptr:   v.Ptr,
	}

	c.m.Unlock()

	return true
}

// Refresh sets new TTL for the given key, returning true if the key (still) exists.
func (c *Cache[K, V]) Refresh(key K, ttl time.Duration) bool {
	c.m.Lock()

	v, ok := c.cache[key]
	if !ok {
		c.m.Unlock()

		return false
	}

	expires := time.Now().Add(ttl)
	wasFirst := c.head.Key == key

	start := c.remove(v.Ptr) // Remove the item from the queue to put into a new place

	if start == nil { // It was the only item in the queue
		v.Ptr.Expires = expires
		c.head = v.Ptr
		c.tail = v.Ptr
		c.setTimer()
		c.m.Unlock()

		return true
	}

	if expires.After(v.Ptr.Expires) { // Move towards the tail
		for n := start; ; n = n.Next {
			if expires.Before(n.Expires) {
				c.insertBefore(v.Ptr, n)

				break
			}

			if n.Next == nil {
				c.insertAfter(v.Ptr, n)

				break
			}
		}
	} else { // Move it towards the head
		for n := start; ; n = n.Prev {
			if expires.After(n.Expires) {
				c.insertAfter(v.Ptr, n)

				break
			}

			if n.Prev == nil {
				c.insertBefore(v.Ptr, n)

				break
			}
		}
	}

	v.Ptr.Expires = expires

	if wasFirst || c.head.Key == key {
		c.setTimer()
	}

	c.m.Unlock()

	return true
}

// Evict removes (at most) n items that expire earliest, returning the number of actually evicted items.
func (c *Cache[K, V]) Evict(n int) (evicted int) {
	c.m.Lock()

	for evicted = 0; evicted < n && c.head != nil && c.delete(c.head.Key); evicted++ {
	}

	if evicted > 0 {
		if c.head != nil {
			c.setTimer()
		} else if c.timer != nil {
			c.timer.Stop()
		}
	}

	c.m.Unlock()

	return
}

// Range iterates over key/value pairs using supplied function until it returns false.
// Values are provided in the order of eviction. It is safe to manipulate the cache within the function.
func (c *Cache[K, V]) Range(fn func(K, V) bool) {
	c.m.RLock()
	keys := make([]K, 0, len(c.cache))
	for n := c.head; n != nil; n = n.Next {
		keys = append(keys, n.Key)
	}
	c.m.RUnlock()

	for k := range keys {
		c.m.RLock()
		value := c.cache[keys[k]].Value
		c.m.RUnlock()

		if !fn(keys[k], value) {
			break
		}
	}
}

// Rekey replaces value's key. Returns false if the old key is not present.
func (c *Cache[K, V]) Rekey(oldKey, newKey K) bool {
	c.m.Lock()

	v, ok := c.cache[oldKey]
	if !ok {
		c.m.Unlock()
		return false
	}

	if oldKey == newKey {
		c.m.Unlock()
		return true
	}

	if existing, ok := c.cache[newKey]; ok {
		timerResetNeeded := c.head == existing.Ptr

		c.remove(existing.Ptr)

		if c.head != nil && timerResetNeeded {
			c.setTimer()
		}
	}

	v.Ptr.Key = newKey
	c.cache[newKey] = v
	delete(c.cache, oldKey)

	c.m.Unlock()

	return true
}

// Len returns number of items currently stored in the cache.
func (c *Cache[K, V]) Len() int {
	c.m.RLock()
	defer c.m.RUnlock()

	return len(c.cache)
}

// setTimer (re)schedules the single expiry timer for the current head.
// It must be called with c.m held and c.head != nil.
func (c *Cache[K, V]) setTimer() {
	d := time.Until(c.head.Expires)

	if c.timer == nil {
		c.timer = time.AfterFunc(d, c.onTimer)

		return
	}

	c.timer.Reset(d)
}

// onTimer evicts every item that has actually expired and reschedules for
// the next one. It re-derives all work from live state, so a spurious or
// stale wake-up (e.g. from a Reset/Stop race) only ever evicts truly
// expired items and is otherwise a no-op.
func (c *Cache[K, V]) onTimer() {
	c.m.Lock()
	defer c.m.Unlock()

	now := time.Now()

	for c.head != nil && !c.head.Expires.After(now) {
		delete(c.cache, c.head.Key)

		c.remove(c.head)
	}

	if c.head != nil {
		c.timer.Reset(time.Until(c.head.Expires))
	}
}

func (c *Cache[K, V]) delete(key K) bool {
	if c.head == nil {
		return false
	}

	value, ok := c.cache[key]
	if !ok {
		return false
	}

	c.remove(value.Ptr)

	delete(c.cache, key)

	return true
}

func (c *Cache[K, V]) remove(n *item[K]) (r *item[K]) {
	if n.Prev == nil {
		c.head = n.Next
	} else {
		n.Prev.Next = n.Next
	}

	if n.Next == nil {
		c.tail = n.Prev
		r = n.Prev
	} else {
		n.Next.Prev = n.Prev
		r = n.Next
	}

	n.Prev, n.Next = nil, nil

	return
}

func (c *Cache[K, V]) insertBefore(n, p *item[K]) {
	if p.Prev == nil {
		c.head = n
	} else {
		p.Prev.Next = n
	}

	n.Prev = p.Prev
	n.Next = p
	p.Prev = n

}

func (c *Cache[K, V]) insertAfter(n, p *item[K]) {
	if p.Next == nil {
		c.tail = n
	} else {
		p.Next.Prev = n
	}

	n.Next = p.Next
	n.Prev = p
	p.Next = n
}
