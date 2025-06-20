package extensions

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// Cache is an interface that defines methods for caching data.
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}) error
	SetWithExpiration(key string, value interface{}, expiration int64) error
	Delete(key string) error
	Exists(key string) bool
	Close() error
}

type CacheEntry struct {
	Value     interface{}
	ExpiresAt int64 // Unix timestamp in seconds
}

type cacheResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newCacheResponseWriter(w http.ResponseWriter) *cacheResponseWriter {
	return &cacheResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           new(bytes.Buffer),
	}
}

func (rw *cacheResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
}

func (rw *cacheResponseWriter) Write(b []byte) (int, error) {
	return rw.body.Write(b)
}

func (rw *cacheResponseWriter) flush() {
	rw.ResponseWriter.WriteHeader(rw.statusCode)

	if rw.body.Len() > 0 {
		rw.ResponseWriter.Write(rw.body.Bytes())
	}
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

		wrapped := newCacheResponseWriter(w)
		next(wrapped, r)

		capturedBody := wrapped.body.Bytes()
		if wrapped.statusCode == http.StatusOK && len(capturedBody) > 0 {
			cache.Set(key, capturedBody)
		}

		wrapped.flush()
	}
}
