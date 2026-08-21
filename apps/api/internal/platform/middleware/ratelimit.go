// Package middleware provides platform-level middleware for the Nexus API.
// LoginRateLimiter implements progressive lockout for the login endpoint
// as specified in SRS §3.5: 5 attempts per window, with escalating lockout
// durations (5 min → 15 min → 1 hour).
package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/iqbaljlldn/nexus/pkg/ratelimit"
)

const (
	// loginRateLimitPrefix is the Redis key prefix for login attempt tracking.
	loginRateLimitPrefix = "ratelimit:login:"

	// lockoutActivePrefix is the Redis key prefix for active lockout flags.
	lockoutActivePrefix = "ratelimit:lockout:active:"

	// lockoutTierPrefix is the Redis key prefix for lockout tier counters.
	lockoutTierPrefix = "ratelimit:lockout:tier:"

	// loginAttemptLimit is the maximum number of failed login attempts
	// before triggering a lockout (SRS §3.5).
	loginAttemptLimit = 5

	// loginAttemptWindow is the sliding window for counting failed attempts.
	loginAttemptWindow = 15 * time.Minute

	// lockoutTierTTL is how long the lockout tier counter persists.
	// After this period without new lockouts, the tier resets naturally.
	lockoutTierTTL = 24 * time.Hour
)

// lockoutDurations defines progressive lockout tiers (SRS §3.5).
// Index 0 = first lockout, 1 = second, 2+ = third and beyond.
var lockoutDurations = []time.Duration{
	5 * time.Minute,  // Tier 0: first lockout
	15 * time.Minute, // Tier 1: second lockout
	60 * time.Minute, // Tier 2+: third and subsequent lockouts
}

// LoginRateLimitResult contains the outcome of a rate limit check.
type LoginRateLimitResult struct {
	Allowed    bool
	RetryAfter time.Duration // Only meaningful when Allowed is false.
}

// LoginRateLimiter enforces rate limiting on the login endpoint with
// progressive lockout tiers. It combines the generic sliding window
// rate limiter (pkg/ratelimit) with Redis-backed lockout state tracking.
type LoginRateLimiter struct {
	limiter     *ratelimit.RateLimiter
	redisClient *redis.Client
}

// NewLoginRateLimiter creates a new LoginRateLimiter.
func NewLoginRateLimiter(limiter *ratelimit.RateLimiter, redisClient *redis.Client) *LoginRateLimiter {
	return &LoginRateLimiter{
		limiter:     limiter,
		redisClient: redisClient,
	}
}

// CheckLoginAllowed verifies whether a login attempt for the given identifier
// is permitted. It checks for an active lockout first, then the sliding window
// rate limit.
func (l *LoginRateLimiter) CheckLoginAllowed(ctx context.Context, identifier string) (*LoginRateLimitResult, error) {
	// 1. Check if there's an active lockout for this identifier
	lockoutKey := lockoutActivePrefix + identifier
	ttl, err := l.redisClient.TTL(ctx, lockoutKey).Result()
	if err != nil {
		return nil, fmt.Errorf("check lockout TTL: %w", err)
	}

	// TTL > 0 means the key exists and has an expiry — user is locked out
	if ttl > 0 {
		return &LoginRateLimitResult{
			Allowed:    false,
			RetryAfter: ttl,
		}, nil
	}

	return &LoginRateLimitResult{Allowed: true}, nil
}

// RecordFailedAttempt records a failed login attempt for the given identifier.
// If the attempt count exceeds the limit, a progressive lockout is triggered.
// Returns the result indicating whether a lockout was just activated.
func (l *LoginRateLimiter) RecordFailedAttempt(ctx context.Context, identifier string) (*LoginRateLimitResult, error) {
	key := loginRateLimitPrefix + identifier

	// Use the sliding window limiter to track the failed attempt
	allowed, err := l.limiter.Allow(ctx, key, loginAttemptWindow, loginAttemptLimit)
	if err != nil {
		return nil, fmt.Errorf("record failed attempt: %w", err)
	}

	if !allowed {
		// Rate limit exceeded — trigger progressive lockout
		lockoutDuration, err := l.activateLockout(ctx, identifier)
		if err != nil {
			return nil, fmt.Errorf("activate lockout: %w", err)
		}

		return &LoginRateLimitResult{
			Allowed:    false,
			RetryAfter: lockoutDuration,
		}, nil
	}

	return &LoginRateLimitResult{Allowed: true}, nil
}

// ResetOnSuccess clears lockout state for the identifier after a successful login.
// This gives the user a fresh start as confirmed in the implementation plan.
func (l *LoginRateLimiter) ResetOnSuccess(ctx context.Context, identifier string) error {
	pipe := l.redisClient.Pipeline()
	pipe.Del(ctx, loginRateLimitPrefix+identifier)
	pipe.Del(ctx, lockoutActivePrefix+identifier)
	pipe.Del(ctx, lockoutTierPrefix+identifier)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("reset rate limit state: %w", err)
	}
	return nil
}

// activateLockout sets an active lockout for the identifier with a duration
// based on the current lockout tier, then increments the tier.
func (l *LoginRateLimiter) activateLockout(ctx context.Context, identifier string) (time.Duration, error) {
	tierKey := lockoutTierPrefix + identifier
	lockoutKey := lockoutActivePrefix + identifier

	// Get current tier (defaults to 0 if not set)
	tierStr, err := l.redisClient.Get(ctx, tierKey).Result()
	tier := 0
	if err == nil {
		tier, _ = strconv.Atoi(tierStr)
	}

	// Determine lockout duration from tier
	duration := lockoutDuration(tier)

	// Set the active lockout with TTL
	if err := l.redisClient.Set(ctx, lockoutKey, "1", duration).Err(); err != nil {
		return 0, fmt.Errorf("set lockout key: %w", err)
	}

	// Increment the lockout tier for next time
	pipe := l.redisClient.Pipeline()
	pipe.Incr(ctx, tierKey)
	pipe.Expire(ctx, tierKey, lockoutTierTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("increment lockout tier: %w", err)
	}

	// Clear the sliding window counter so it's fresh after lockout expires
	if err := l.redisClient.Del(ctx, loginRateLimitPrefix+identifier).Err(); err != nil {
		return 0, fmt.Errorf("clear rate limit counter: %w", err)
	}

	return duration, nil
}

// lockoutDuration returns the lockout duration for the given tier.
func lockoutDuration(tier int) time.Duration {
	if tier >= len(lockoutDurations) {
		return lockoutDurations[len(lockoutDurations)-1]
	}
	return lockoutDurations[tier]
}
