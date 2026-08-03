package main

import (
	"github.com/iqbaljlldn/nexus/pkg/config"
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

	r := InitializeRouter(log)

	if err := r.Run(); err != nil {
		log.Fatal("Server failed to run", zap.Error(err))
	}
}
