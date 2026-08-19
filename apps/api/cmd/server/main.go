package main

import (
	"github.com/iqbaljlldn/nexus/pkg/cache"
	"github.com/iqbaljlldn/nexus/pkg/config"
	"github.com/iqbaljlldn/nexus/pkg/database"
	"github.com/iqbaljlldn/nexus/pkg/logger"
	"go.uber.org/zap"
)

// @title           Nexus API
// @version         1.0
// @description     This is the Nexus API server.
// @host            localhost:8080
// @BasePath        /api/v1
func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	log, err := logger.New(*cfg)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()
	// Initialize Database
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := cache.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatal("failed to connect to redis", zap.Error(err))
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("failed to close redis client", zap.Error(err))
		}
	}()

	r := InitializeRouter(log, db, redisClient)

	if err := r.Run(); err != nil {
		log.Fatal("Server failed to run", zap.Error(err))
	}
}
