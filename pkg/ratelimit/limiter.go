// Package ratelimit provides a generic Redis-based sliding window rate limiter.
// It uses a Lua script executed atomically via EVAL to prevent race conditions.
// Designed as a shared package (pkg/) for reuse across domains (login, messaging, uploads).
package ratelimit

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed rate_limit.lua
var rateLimitScript string

// RateLimiter implements a sliding window rate limiter backed by Redis Sorted Sets.
type RateLimiter struct {
	client *redis.Client
	script *redis.Script
}

// New creates a new RateLimiter using the provided Redis client.
// The Lua script is loaded once and cached via EVALSHA for subsequent calls.
func New(client *redis.Client) *RateLimiter {
	return &RateLimiter{
		client: client,
		script: redis.NewScript(rateLimitScript),
	}
}

// Allow checks whether a request identified by key is permitted within the
// given window and limit. Returns true if the request is allowed, false if
// the rate limit has been exceeded.
//
// The key should be constructed by the caller to represent the resource being
// rate-limited (e.g. "ratelimit:login:user@example.com").
//
// This operation is atomic — the check and increment happen in a single Redis
// EVAL call, preventing TOCTOU race conditions.
func (r *RateLimiter) Allow(ctx context.Context, key string, window time.Duration, limit int) (bool, error) {
	now := time.Now().UnixMilli()
	windowMs := window.Milliseconds()

	result, err := r.script.Run(ctx, r.client, []string{key}, now, windowMs, limit).Int()
	if err != nil {
		return false, fmt.Errorf("rate limit script execution failed: %w", err)
	}

	return result == 1, nil
}
