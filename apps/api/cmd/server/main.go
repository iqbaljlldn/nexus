package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

	app := InitializeApp(log, db, redisClient)

	addr := cfg.Port
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           app.Engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		log.Info("server starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received, draining connections")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("forced http server shutdown", zap.Error(err))
	}

	if app.WSRegistry != nil {
		app.WSRegistry.CloseAllGracefully(shutdownCtx)
	}

	log.Info("server shutdown gracefully completed")
}
