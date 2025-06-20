package extensions

import (
	"sync"
	"time"
)

// MapCache is a simple in-memory cache implementation using a map.
type MapCache struct {
	data        map[string]CacheEntry
	mutex       sync.RWMutex
	ttl         time.Duration
	cleaner     *time.Ticker
	stopCleaner chan struct{}
}

func NewMapCache() *MapCache {
	c := &MapCache{
		data:        make(map[string]CacheEntry),
		ttl:         5 * time.Minute,
		cleaner:     time.NewTicker(1 * time.Minute),
		stopCleaner: make(chan struct{}),
	}

	go c.startCleaner()

	return c
}

func (c *MapCache) startCleaner() {
	go func() {
		for {
			select {
			case <-c.cleaner.C:
				c.mutex.Lock()
				now := time.Now().Unix()
				for key, entry := range c.data {
					if entry.ExpiresAt > 0 && now > entry.ExpiresAt {
						delete(c.data, key)
					}
				}
				c.mutex.Unlock()
			case <-c.stopCleaner:
				return
			}
		}
	}()
}

// SetDefaultTTL sets the default time-to-live for cache entries.
func (c *MapCache) SetDefaultTTL(ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.ttl = ttl
}

// Get retrieves a value from the cache by key.
func (c *MapCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists || (entry.ExpiresAt > 0 && time.Now().Unix() > entry.ExpiresAt) {
		return nil, false
	}

	return entry.Value, true
}

// Set adds a value to the cache with a default expiration time.
func (c *MapCache) Set(key string, value interface{}) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	expiration := time.Now().Add(c.ttl)
	c.data[key] = CacheEntry{
		Value:     value,
		ExpiresAt: expiration.Unix(),
	}

	return nil
}

// SetWithExpiration adds a value to the cache with a specific expiration time.
func (c *MapCache) SetWithExpiration(key string, value interface{}, expiration int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = CacheEntry{
		Value:     value,
		ExpiresAt: expiration,
	}

	return nil
}

// Delete removes a value from the cache by key.
func (c *MapCache) Delete(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)

	return nil
}

// Exists checks if a key exists in the cache.
func (c *MapCache) Exists(key string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	_, exists := c.data[key]

	return exists
}

// Close stops the cache cleaner and releases resources.
func (c *MapCache) Close() error {
	c.cleaner.Stop()

	close(c.stopCleaner)

	return nil
}
