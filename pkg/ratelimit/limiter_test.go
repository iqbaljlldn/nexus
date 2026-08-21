package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestLimiter(t *testing.T) (*RateLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return New(client), mr
}

func TestAllow_UnderLimit(t *testing.T) {
	limiter, _ := setupTestLimiter(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, "test:key", 15*time.Minute, 5)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}
}

func TestAllow_AtLimitDenied(t *testing.T) {
	limiter, _ := setupTestLimiter(t)
	ctx := context.Background()

	// Use up the limit
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, "test:key", 15*time.Minute, 5)
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	// 6th request should be denied
	allowed, err := limiter.Allow(ctx, "test:key", 15*time.Minute, 5)
	require.NoError(t, err)
	assert.False(t, allowed, "request 6 should be denied when limit is 5")
}

func TestAllow_AfterWindowExpiry(t *testing.T) {
	limiter, mr := setupTestLimiter(t)
	ctx := context.Background()

	// Fill up the limit
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, "test:key", 15*time.Minute, 5)
		require.NoError(t, err)
		assert.True(t, allowed)
	}

	// Denied at limit
	allowed, err := limiter.Allow(ctx, "test:key", 15*time.Minute, 5)
	require.NoError(t, err)
	assert.False(t, allowed)

	// Fast-forward past the window
	mr.FastForward(16 * time.Minute)

	// Should be allowed again
	allowed, err = limiter.Allow(ctx, "test:key", 15*time.Minute, 5)
	require.NoError(t, err)
	assert.True(t, allowed, "request should be allowed after window expires")
}

func TestAllow_DifferentKeysIndependent(t *testing.T) {
	limiter, _ := setupTestLimiter(t)
	ctx := context.Background()

	// Fill up key1
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, "test:key1", 15*time.Minute, 5)
		require.NoError(t, err)
		assert.True(t, allowed)
	}

	// key1 should be denied
	allowed, err := limiter.Allow(ctx, "test:key1", 15*time.Minute, 5)
	require.NoError(t, err)
	assert.False(t, allowed)

	// key2 should still be allowed (independent)
	allowed, err = limiter.Allow(ctx, "test:key2", 15*time.Minute, 5)
	require.NoError(t, err)
	assert.True(t, allowed, "different key should have independent rate limit")
}

func TestAllow_LimitOfOne(t *testing.T) {
	limiter, _ := setupTestLimiter(t)
	ctx := context.Background()

	allowed, err := limiter.Allow(ctx, "test:single", 1*time.Minute, 1)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = limiter.Allow(ctx, "test:single", 1*time.Minute, 1)
	require.NoError(t, err)
	assert.False(t, allowed, "second request should be denied with limit=1")
}
