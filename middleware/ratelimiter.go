package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	HEADER_RATELIMIT_LIMIT       = "X-RateLimit-Limit"
	HEADER_RATELIMIT_REMAINING   = "X-RateLimit-Remaining"
	HEADER_RATELIMIT_RESET       = "X-RateLimit-Reset"
	HEADER_RATELIMIT_RETRY_AFTER = "Retry-After"
)

// RateLimiterType defines the type of rate limiter algorithm to use.
type RateLimiterAlgorithm string

// RateLimiterDatabaseType defines the type of database used for rate limiting.
type RateLimiterDatabaseType string

const (
	TokenBucket   RateLimiterAlgorithm = "token_bucket"
	SlidingWindow RateLimiterAlgorithm = "sliding_window"
)

const (
	Redis    RateLimiterDatabaseType = "redis"
	InMemory RateLimiterDatabaseType = "in_memory"
)

// RateLimiterError is a custom error type for rate limiter operations.
type RateLimiterError struct {
	Msg string
	Err error
}

func NewRateLimiterError(msg string, err error) *RateLimiterError {
	return &RateLimiterError{
		Msg: msg,
		Err: err,
	}
}

func (e *RateLimiterError) Error() string {
	return fmt.Sprintf("RateLimiterError: %s: %v", e.Msg, e.Err)
}

// RateLimiter is an interface that defines methods for rate limiting functionality.
type RateLimiter interface {
	// Allow checks if a request is allowed based on the rate limiting algorithm.
	Allow(key string) (bool, error)
	// Set sets the rate limit for a given key.
	Set(key string, limit int64) error
	// SetWithExpiration sets the rate limit for a given key with an expiration time.
	SetWithExpiration(key string, limit int64, expiration int64) error
	// Get retrieves the current rate limit for a given key.
	Get(key string) (int64, error)
	// GetWithExpiration retrieves the current rate limit and expiration time for a given key.
	GetWithExpiration(key string) (int64, int64, error)
	// Delete removes the rate limit for a given key.
	Delete(key string) error
	// Exists checks if a rate limit exists for a given key.
	Exists(key string) bool
	// Close closes the rate limiter, releasing any resources it holds.
	Close() error
	// Cleanup performs any necessary cleanup operations for the rate limiter.
	Cleanup() error
	// Middleware returns a middleware function that applies the rate limiting logic.
	Middleware(next http.HandlerFunc) http.HandlerFunc
}

type RateLimiterConfig struct {
	Algorithm  RateLimiterAlgorithm    // The algorithm used for rate limiting
	Database   RateLimiterDatabaseType // The database type used for storing rate limits
	Limit      int64                   // The maximum number of requests allowed
	Expiration int64                   // Reset expiration time in seconds
}

// TokenBucketEntry represents a token bucket for rate limiting
type TokenBucketEntry struct {
	Tokens     int64     // Current number of tokens
	Capacity   int64     // Maximum capacity of the bucket
	RefillRate int64     // Tokens per second refill rate
	LastRefill time.Time // Last time tokens were refilled
	ExpiresAt  int64     // Unix timestamp when this entry expires (0 for no expiration)
}

// InMemoryRateLimiter implements the RateLimiter interface using in-memory storage and token bucket algorithm
type InMemoryRateLimiter struct {
	buckets       map[string]*TokenBucketEntry
	mutex         sync.RWMutex
	defaultConfig RateLimiterConfig
	cleaner       *time.Ticker
	stopCleaner   chan struct{}
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter with token bucket algorithm
// If no configuration is provided, it uses default values.
func NewInMemoryRateLimiter(config *RateLimiterConfig) *InMemoryRateLimiter {
	if config == nil {
		config = &RateLimiterConfig{
			Algorithm:  TokenBucket,
			Database:   InMemory,
			Limit:      100,
			Expiration: int64(60 * time.Minute),
		}
	}

	rl := &InMemoryRateLimiter{
		buckets:       make(map[string]*TokenBucketEntry),
		defaultConfig: *config,
		cleaner:       time.NewTicker(1 * time.Minute),
		stopCleaner:   make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.startCleaner()

	return rl
}

func (rl *InMemoryRateLimiter) startCleaner() {
	for {
		select {
		case <-rl.cleaner.C:
			rl.mutex.Lock()
			now := time.Now().Unix()
			for key, bucket := range rl.buckets {
				if bucket.ExpiresAt > 0 && now > bucket.ExpiresAt {
					delete(rl.buckets, key)
				}
			}
			rl.mutex.Unlock()
		case <-rl.stopCleaner:
			return
		}
	}
}

// Allow checks if a request is allowed based on the token bucket algorithm
func (rl *InMemoryRateLimiter) Allow(key string) (bool, error) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[key]

	if !exists {
		// Create new bucket with default config
		bucket = &TokenBucketEntry{
			Tokens:     rl.defaultConfig.Limit - 1, // Subtract 1 for current request
			Capacity:   rl.defaultConfig.Limit,
			RefillRate: rl.defaultConfig.Limit, // Refill rate equals capacity for simplicity
			LastRefill: now,
			ExpiresAt:  0, // No expiration by default
		}
		rl.buckets[key] = bucket

		return true, nil
	}

	// Check if bucket has expired
	if bucket.ExpiresAt > 0 && now.Unix() > bucket.ExpiresAt {
		delete(rl.buckets, key)

		return rl.Allow(key) // Recursive call to create new bucket
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(bucket.LastRefill)
	tokensToAdd := int64(elapsed.Seconds()) * bucket.RefillRate
	if tokensToAdd > 0 {
		bucket.Tokens = min(bucket.Capacity, bucket.Tokens+tokensToAdd)
		bucket.LastRefill = now
	}

	// Check if request is allowed
	if bucket.Tokens > 0 {
		bucket.Tokens--

		return true, nil
	}

	return false, nil
}

// Set sets the rate limit for a given key
func (rl *InMemoryRateLimiter) Set(key string, limit int64) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket := &TokenBucketEntry{
		Tokens:     limit,
		Capacity:   limit,
		RefillRate: limit,
		LastRefill: time.Now(),
		ExpiresAt:  0,
	}
	rl.buckets[key] = bucket

	return nil
}

// SetWithExpiration sets the rate limit for a given key with an expiration time
func (rl *InMemoryRateLimiter) SetWithExpiration(key string, limit int64, expiration int64) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket := &TokenBucketEntry{
		Tokens:     limit,
		Capacity:   limit,
		RefillRate: limit,
		LastRefill: time.Now(),
		ExpiresAt:  expiration,
	}
	rl.buckets[key] = bucket

	return nil
}

// Get retrieves the current rate limit for a given key
func (rl *InMemoryRateLimiter) Get(key string) (int64, error) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		return 0, NewRateLimiterError("key not found", nil)
	}

	// Check if bucket has expired
	if bucket.ExpiresAt > 0 && time.Now().Unix() > bucket.ExpiresAt {
		return 0, NewRateLimiterError("key expired", nil)
	}

	return bucket.Tokens, nil
}

// GetWithExpiration retrieves the current rate limit and expiration time for a given key
func (rl *InMemoryRateLimiter) GetWithExpiration(key string) (int64, int64, error) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		return 0, 0, NewRateLimiterError("key not found", nil)
	}

	// Check if bucket has expired
	if bucket.ExpiresAt > 0 && time.Now().Unix() > bucket.ExpiresAt {
		return 0, 0, NewRateLimiterError("key expired", nil)
	}

	return bucket.Tokens, bucket.ExpiresAt, nil
}

// Delete removes the rate limit for a given key
func (rl *InMemoryRateLimiter) Delete(key string) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	delete(rl.buckets, key)

	return nil
}

// Exists checks if a rate limit exists for a given key
func (rl *InMemoryRateLimiter) Exists(key string) bool {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		return false
	}

	// Check if bucket has expired
	if bucket.ExpiresAt > 0 && time.Now().Unix() > bucket.ExpiresAt {
		return false
	}

	return true
}

// Close closes the rate limiter, releasing any resources it holds
func (rl *InMemoryRateLimiter) Close() error {
	rl.cleaner.Stop()
	close(rl.stopCleaner)

	return nil
}

// Cleanup performs any necessary cleanup operations for the rate limiter
func (rl *InMemoryRateLimiter) Cleanup() error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now().Unix()
	for key, bucket := range rl.buckets {
		if bucket.ExpiresAt > 0 && now > bucket.ExpiresAt {
			delete(rl.buckets, key)
		}
	}

	return nil
}

// Middleware returns a middleware function that applies the rate limiting logic
func (rl *InMemoryRateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.RemoteAddr

		allowed, err := rl.Allow(key)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Get bucket info for headers
		rl.mutex.RLock()
		bucket, exists := rl.buckets[key]
		rl.mutex.RUnlock()

		var resetTime int64
		if exists && bucket != nil {
			now := time.Now()
			tokensNeeded := bucket.Capacity - bucket.Tokens
			if tokensNeeded > 0 && bucket.RefillRate > 0 {
				secondsToRefill := tokensNeeded / bucket.RefillRate
				resetTime = now.Add(time.Duration(secondsToRefill) * time.Second).Unix()
			} else {
				resetTime = now.Unix()
			}
		} else {
			resetTime = time.Now().Unix()
		}

		if !allowed {
			w.Header().Set(HEADER_RATELIMIT_LIMIT, fmt.Sprintf("%d", rl.defaultConfig.Limit))
			w.Header().Set(HEADER_RATELIMIT_REMAINING, "0")
			w.Header().Set(HEADER_RATELIMIT_RESET, fmt.Sprintf("%d", resetTime))
			w.Header().Set(HEADER_RATELIMIT_RETRY_AFTER, "1")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)

			return
		}

		tokens, _ := rl.Get(key)
		w.Header().Set(HEADER_RATELIMIT_LIMIT, fmt.Sprintf("%d", rl.defaultConfig.Limit))
		w.Header().Set(HEADER_RATELIMIT_REMAINING, fmt.Sprintf("%d", tokens))
		w.Header().Set(HEADER_RATELIMIT_RESET, fmt.Sprintf("%d", resetTime))

		next(w, r)
	}
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}
