package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	roleDomain "github.com/iqbaljlldn/nexus/apps/api/internal/role/domain"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type PermissionCacheInvalidator struct {
	redisClient *redis.Client
	log         *zap.Logger
}

func NewPermissionCacheInvalidator(redisClient *redis.Client, log *zap.Logger) *PermissionCacheInvalidator {
	return &PermissionCacheInvalidator{
		redisClient: redisClient,
		log:         log,
	}
}

// InvalidateUserPermissions invalidates all cached permission resolutions for a specific user in a workspace.
// Uses SCAN to avoid blocking Redis, per LLD 2.6.
func (i *PermissionCacheInvalidator) InvalidateUserPermissions(ctx context.Context, workspaceID, userID uuid.UUID) error {
	pattern := fmt.Sprintf("perm:%s:%s:*", workspaceID.String(), userID.String())
	return i.deleteByPattern(ctx, pattern)
}

func (i *PermissionCacheInvalidator) deleteByPattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var deletedCount int

	for {
		var keys []string
		var err error
		keys, cursor, err = i.redisClient.Scan(ctx, cursor, pattern, 100).Result() // batch size 100
		if err != nil {
			return fmt.Errorf("scan redis keys: %w", err)
		}

		if len(keys) > 0 {
			if err := i.redisClient.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("delete redis keys: %w", err)
			}
			deletedCount += len(keys)
		}

		if cursor == 0 { // finished
			break
		}
	}

	if deletedCount > 0 {
		i.log.Debug("invalidated permission cache", zap.String("pattern", pattern), zap.Int("deleted_count", deletedCount))
	}

	return nil
}

// CachedPermissionResolver wraps the base PermissionResolver with a Redis cache-aside pattern.
type CachedPermissionResolver struct {
	base        *PermissionResolver
	redisClient *redis.Client
	log         *zap.Logger
	ttl         time.Duration
}

func NewCachedPermissionResolver(base *PermissionResolver, redisClient *redis.Client, log *zap.Logger) *CachedPermissionResolver {
	return &CachedPermissionResolver{
		base:        base,
		redisClient: redisClient,
		log:         log,
		ttl:         60 * time.Second, // 60 seconds TTL per LLD 3.5.2
	}
}

func (c *CachedPermissionResolver) Resolve(ctx context.Context, userID, workspaceID, channelID uuid.UUID, required roleDomain.PermissionFlag) (bool, error) {
	// Key format: perm:{workspaceID}:{userID}:{channelID}:{requiredFlag}
	key := fmt.Sprintf("perm:%s:%s:%s:%d", workspaceID.String(), userID.String(), channelID.String(), required)

	// Try to get from cache
	val, err := c.redisClient.Get(ctx, key).Result()
	if err == nil {
		return val == "1", nil
	}
	if err != redis.Nil {
		// Log error but continue to DB to maintain availability
		c.log.Warn("redis get failed for permission cache, falling back to db", zap.Error(err), zap.String("key", key))
	}

	// Not in cache (or error), resolve from base (DB)
	allowed, err := c.base.Resolve(ctx, userID, workspaceID, channelID, required)
	if err != nil {
		return false, err
	}

	// Set to cache asynchronously or synchronously (we'll do sync for simplicity and correctness)
	strVal := "0"
	if allowed {
		strVal = "1"
	}
	if err := c.redisClient.Set(ctx, key, strVal, c.ttl).Err(); err != nil {
		// Just log, don't fail the request
		c.log.Warn("redis set failed for permission cache", zap.Error(err), zap.String("key", key))
	}

	return allowed, nil
}
