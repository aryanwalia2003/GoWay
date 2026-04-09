package barcode

import (
	"sync"
)

// Cache provides a thread-safe map for storing generated PNG bytes.
type Cache struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewCache initializes a new thread-safe barcode cache.
func NewCache() *Cache {
	return &Cache{
		data: make(map[string][]byte),
	}
}

// Get returns the cached PNG bytes for the given value string if present.
func (c *Cache) Get(value string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	bytes, ok := c.data[value]
	return bytes, ok
}

// Set stores the PNG bytes for the given value string.
func (c *Cache) Set(value string, bytes []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[value] = bytes
}
