package application

import (
	"context"

	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
	"github.com/iqbaljlldn/nexus/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Service interface {
	Healthz(ctx context.Context) (*HealthResponse, error)
	Readyz(ctx context.Context) (*HealthResponse, error)
}

type service struct {
	logger *zap.Logger
	db     *pgxpool.Pool
	redis  *redis.Client
}

func NewService(logger *zap.Logger, db *pgxpool.Pool, redisClient *redis.Client) Service {
	return &service{
		logger: logger,
		db:     db,
		redis:  redisClient,
	}
}

func (s *service) Healthz(ctx context.Context) (*HealthResponse, error) {
	log := logger.FromContext(ctx, s.logger)
	log.Debug("healthz check")

	return &HealthResponse{
		Status: "ok",
	}, nil
}

func (s *service) Readyz(ctx context.Context) (*HealthResponse, error) {
	log := logger.FromContext(ctx, s.logger)
	log.Debug("readyz check")

	if err := s.db.Ping(ctx); err != nil {
		log.Error("database connection failed", zap.Error(err))
		return nil, &pkgerrors.InfrastructureError{
			Message: "database unavailable",
			Err:     err,
		}
	}

	if err := s.redis.Ping(ctx).Err(); err != nil {
		log.Error("redis connection failed", zap.Error(err))
		return nil, &pkgerrors.InfrastructureError{
			Message: "redis unavailable",
			Err:     err,
		}
	}

	return &HealthResponse{
		Status: "ok",
	}, nil
}
