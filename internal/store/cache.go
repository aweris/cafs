package store

import "sync"

// Cache provides in-memory caching for objects.
type Cache interface {
	Get(key string) ([]byte, bool)
	Add(key string, value []byte)
	Has(key string) bool
	Remove(key string)
	Clear()
}

// LRUCache implements a simple LRU cache.
// TODO: Use a proper LRU implementation like hashicorp/golang-lru
type LRUCache struct {
	maxSize int
	items   map[string][]byte
	mu      sync.RWMutex
}

// NewLRUCache creates a new LRU cache.
func NewLRUCache(maxSize int) *LRUCache {
	return &LRUCache{
		maxSize: maxSize,
		items:   make(map[string][]byte),
	}
}

// Get retrieves a value from the cache.
func (c *LRUCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.items[key]
	return val, ok
}

// Add adds a value to the cache.
func (c *LRUCache) Add(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction: if full, remove random item
	// TODO: Implement proper LRU eviction
	if len(c.items) >= c.maxSize {
		// Remove one item (any)
		for k := range c.items {
			delete(c.items, k)
			break
		}
	}

	c.items[key] = value
}

// Has checks if a key exists in the cache.
func (c *LRUCache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.items[key]
	return ok
}

// Remove removes a key from the cache.
func (c *LRUCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Clear clears the cache.
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string][]byte)
}
