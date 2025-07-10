package extensions

import (
	"encoding/json"
	"net/http"
)

// Cache is an interface that defines methods for caching data.
type Cache interface {
	// Get retrieves a value from the cache by key.
	Get(key string) (interface{}, bool)
	// Set stores a value in the cache storage with the specified key.
	Set(key string, value interface{}) error
	// SetWithExpiration adds a value to the cache with a specific expiration time.
	SetWithExpiration(key string, value interface{}, expiration int64) error
	// Delete removes a value from the cache by key.
	Delete(key string) error
	// Exists checks if a key exists in the cache.
	Exists(key string) bool
	// Close closes the cache storage, releasing any resources it holds.
	Close() error
}

type CacheEntry struct {
	Value     interface{}
	ExpiresAt int64 // Unix timestamp in seconds
}

// CacheMiddleware is a middleware that provides caching functionality.
// It used to caching response data for HTTP requests by matching the request path and method.
// Key pattern that used by this middleware is `"{method}:{path}"`. For example, if the request path is `/api/users/1`, the key will be `GET:/api/users/1`.
func CacheMiddleware(cache Cache, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + ":" + r.URL.Path

		if c, ok := cache.Get(key); ok {
			if resp, ok := c.([]byte); ok {
				if isJSON := json.Valid(resp); isJSON {
					w.Header().Set("Content-Type", "application/json")
				} else {
					w.Header().Set("Content-Type", "text/plain")
				}

				w.WriteHeader(http.StatusOK)
				w.Write(resp)

				return
			}
		}

		wrapped := NewCustomResponseWriter(w)
		next(wrapped, r)

		capturedBody := wrapped.Body.Bytes()
		if wrapped.StatusCode == http.StatusOK && len(capturedBody) > 0 {
			cache.Set(key, capturedBody)
		}

		wrapped.Flush()
	}
}
