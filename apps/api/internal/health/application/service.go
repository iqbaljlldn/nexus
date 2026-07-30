package application

import (
	"context"

	"nexus-be/pkg/logger"

	"go.uber.org/zap"
)

type Service interface {
	Check(ctx context.Context) (*HealthResponse, error)
}

type service struct {
	logger *zap.Logger
}

func NewService(logger *zap.Logger) Service {
	return &service{
		logger: logger,
	}
}

func (s *service) Check(ctx context.Context) (*HealthResponse, error) {
	log := logger.FromContext(ctx, s.logger)

	log.Info("health check")

	return &HealthResponse{
		Status: "UP",
	}, nil
}
