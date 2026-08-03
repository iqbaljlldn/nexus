package database

import (
	"context"
	"time"

	"github.com/iqbaljlldn/nexus/pkg/config"
	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgres initializes and returns a new pgxpool.Pool connected to PostgreSQL.
func NewPostgres(cfg config.Database) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, &pkgerrors.InfrastructureError{
			Message: "unable to parse database config",
			Err:     err,
		}
	}

	// Pool configuration (defaults for high concurrency)
	poolConfig.MaxConns = 50
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Create a context with timeout for the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, &pkgerrors.InfrastructureError{
			Message: "unable to connect to database",
			Err:     err,
		}
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, &pkgerrors.InfrastructureError{
			Message: "database ping failed",
			Err:     err,
		}
	}

	return pool, nil
}
