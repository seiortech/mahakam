package extensions

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrorRedisClientNil       = errors.New("redis client cannot be nil")
	ErrorRedisUnsupportedType = errors.New("unsupported value type")
	ErrorRedisValueNil        = errors.New("value cannot be nil")
	ErrorExpirationInFuture   = errors.New("expiration time must be in the future")
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	ctx    context.Context
}

// NewRedisCache creates a new RedisCache instance with the provided Redis client.
func NewRedisCache(client *redis.Client) (*RedisCache, error) {
	if client == nil {
		return nil, ErrorRedisClientNil
	}

	if cmd := client.Ping(context.Background()); cmd.Err() != nil && cmd.Err() != redis.Nil {
		return nil, cmd.Err()
	}

	return &RedisCache{
		client: client,
		ttl:    5 * time.Minute,
		ctx:    context.Background(),
	}, nil
}

// SetDefaultTTL sets the default time-to-live for cache entries.
func (c *RedisCache) SetDefaultTTL(ttl time.Duration) {
	c.ttl = ttl
}

// Get retrieves a value from the cache by key.
func (c *RedisCache) Get(key string) (interface{}, bool) {
	val, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		return nil, false
	}

	return []byte(val), true
}

// Set adds a value to the cache with a default expiration time.
// It supports string and []byte types, returning an error for unsupported types.
func (c *RedisCache) Set(key string, value interface{}) error {
	if value == nil {
		return ErrorRedisValueNil
	}

	var val string
	switch v := value.(type) {
	case string:
		val = v
	case []byte:
		val = string(v)
	default:
		return ErrorRedisUnsupportedType
	}

	return c.client.Set(c.ctx, key, val, c.ttl).Err()
}

// SetWithExpiration adds a value to the cache with a specific expiration time.
// It supports string and []byte types, returning an error for unsupported types.
func (c *RedisCache) SetWithExpiration(key string, value interface{}, expiration int64) error {
	if value == nil {
		return ErrorRedisValueNil
	}

	var val string
	switch v := value.(type) {
	case string:
		val = v
	case []byte:
		val = string(v)
	default:
		return ErrorRedisUnsupportedType
	}

	// Calculate TTL from expiration timestamp
	now := time.Now().Unix()
	if expiration <= now {
		return ErrorExpirationInFuture
	}

	ttl := time.Duration(expiration-now) * time.Second
	return c.client.Set(c.ctx, key, val, ttl).Err()
}

// Delete removes a value from the cache by key.
func (c *RedisCache) Delete(key string) error {
	return c.client.Del(c.ctx, key).Err()
}

// Exists checks if a key exists in the cache.
func (c *RedisCache) Exists(key string) bool {
	result, err := c.client.Exists(c.ctx, key).Result()
	if err != nil {
		return false
	}
	return result > 0
}

// Close closes the cache storage, releasing any resources it holds.
func (c *RedisCache) Close() error {
	return c.client.Close()
}
