package cache

import (
	"context"
	"time"

	"github.com/iqbaljlldn/nexus/pkg/config"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/redis/go-redis/v9"
)

// NewRedis initializes and returns a new redis.Client connected to Redis.
func NewRedis(cfg config.Redis) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, &pkgerrors.InfrastructureError{
			Message: "unable to parse redis config",
			Err:     err,
		}
	}

	client := redis.NewClient(opt)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, &pkgerrors.InfrastructureError{
			Message: "redis ping failed",
			Err:     err,
		}
	}

	return client, nil
}
