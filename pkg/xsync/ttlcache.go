package xsync

import (
	"sync"
	"time"
)

// TTLCache is a thread-safe in-memory cache with time-to-live expiration.
// It uses generics to support any key and value types.
type TTLCache[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]*cacheItem[V]
	ttl   time.Duration
	now   func() time.Time
}

// cacheItem holds the cached value and its expiration time.
type cacheItem[V any] struct {
	value     V
	expiresAt time.Time
}

// NewTTLCache creates a new TTLCache with the specified TTL duration.
func NewTTLCache[K comparable, V any](ttl time.Duration) *TTLCache[K, V] {
	return &TTLCache[K, V]{
		items: make(map[K]*cacheItem[V]),
		ttl:   ttl,
		now:   time.Now,
	}
}

// SetNowFunc sets a custom time function for testing purposes.
func (c *TTLCache[K, V]) SetNowFunc(now func() time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
}

// Get retrieves a value from the cache by key.
// Returns the value and true if found and not expired, otherwise returns zero value and false.
func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		var zero V
		return zero, false
	}

	if c.now().After(item.expiresAt) {
		var zero V
		return zero, false
	}

	return item.value, true
}

// Set stores a value in the cache with the default TTL.
func (c *TTLCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem[V]{
		value:     value,
		expiresAt: c.now().Add(c.ttl),
	}
}

// SetWithTTL stores a value in the cache with a custom TTL.
func (c *TTLCache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem[V]{
		value:     value,
		expiresAt: c.now().Add(ttl),
	}
}

// Delete removes a value from the cache by key.
func (c *TTLCache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache.
func (c *TTLCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[K]*cacheItem[V])
}

// Len returns the number of items in the cache (including expired items).
func (c *TTLCache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// Cleanup removes all expired items from the cache.
// This can be called periodically to prevent memory leaks from expired entries.
func (c *TTLCache[K, V]) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
		}
	}
}

// Shutdown clears all cache entries and releases resources.
// This should be called during graceful shutdown.
func (c *TTLCache[K, V]) Shutdown() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[K]*cacheItem[V])
}

// GetOrSet retrieves a value from the cache or sets it using the provided function if not found.
// This is useful for implementing read-through caching patterns.
// The loader function is called while NOT holding the lock to prevent deadlocks.
func (c *TTLCache[K, V]) GetOrSet(key K, loader func() (V, error)) (V, error) {
	// First, try to get from cache
	if value, ok := c.Get(key); ok {
		return value, nil
	}

	// Load the value (outside of lock to prevent deadlocks)
	value, err := loader()
	if err != nil {
		var zero V
		return zero, err
	}

	// Store in cache
	c.Set(key, value)

	return value, nil
}
