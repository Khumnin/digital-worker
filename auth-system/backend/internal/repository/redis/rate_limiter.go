// internal/repository/redis/rate_limiter.go
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements middleware.RateLimiter using a Redis sliding window Lua script.
type RedisRateLimiter struct {
	client *redis.Client
}

// NewRedisRateLimiter creates a new rate limiter backed by Redis.
func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

// slidingWindowScript implements a Redis sliding window rate limiter using sorted sets.
// Arguments: KEYS[1]=key, ARGV[1]=window_ms, ARGV[2]=limit, ARGV[3]=now_ms
// Returns: [allowed(0/1), current_count, ttl_ms]
var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local window_ms = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local window_start = now_ms - window_ms

-- Remove expired entries.
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

-- Count current entries in window.
local count = redis.call('ZCARD', key)

if count < limit then
  -- Add the new request.
  redis.call('ZADD', key, now_ms, now_ms .. '-' .. math.random(1000000))
  redis.call('PEXPIRE', key, window_ms)
  return {1, count + 1, window_ms}
else
  -- Rate limit exceeded.
  local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
  local retry_after_ms = window_ms
  if #oldest > 0 then
    retry_after_ms = window_ms - (now_ms - tonumber(oldest[2]))
  end
  return {0, count, retry_after_ms}
end
`)

// Allow checks whether the request is within the rate limit.
// Returns (allowed, currentCount, retryAfterSeconds, error).
func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, int, error) {
	now := time.Now().UnixMilli()
	windowMS := window.Milliseconds()

	result, err := slidingWindowScript.Run(ctx, r.client,
		[]string{key},
		windowMS, limit, now,
	).Slice()

	if err != nil {
		return false, 0, 0, fmt.Errorf("rate limiter lua script: %w", err)
	}

	if len(result) != 3 {
		return false, 0, 0, fmt.Errorf("unexpected rate limiter result length: %d", len(result))
	}

	allowed := result[0].(int64) == 1
	count := int(result[1].(int64))
	retryAfterMS := result[2].(int64)
	retryAfterSec := int(retryAfterMS / 1000)
	if retryAfterSec < 1 {
		retryAfterSec = 1
	}

	return allowed, count, retryAfterSec, nil
}
