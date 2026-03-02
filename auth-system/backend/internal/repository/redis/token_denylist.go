// internal/repository/redis/token_denylist.go
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenDenylist stores revoked refresh token hashes in Redis.
// Entries expire after the maximum session TTL to bound memory usage.
type TokenDenylist struct {
	client *redis.Client
}

// NewTokenDenylist creates a new Redis-backed token denylist.
func NewTokenDenylist(client *redis.Client) *TokenDenylist {
	return &TokenDenylist{client: client}
}

// Add adds a token hash to the denylist with the given TTL.
func (d *TokenDenylist) Add(ctx context.Context, tokenHash string, ttl time.Duration) error {
	key := fmt.Sprintf("denylist:%s", tokenHash)
	return d.client.Set(ctx, key, "1", ttl).Err()
}

// IsRevoked returns true if the token hash is in the denylist.
func (d *TokenDenylist) IsRevoked(ctx context.Context, tokenHash string) (bool, error) {
	key := fmt.Sprintf("denylist:%s", tokenHash)
	result, err := d.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check denylist: %w", err)
	}
	return result > 0, nil
}
