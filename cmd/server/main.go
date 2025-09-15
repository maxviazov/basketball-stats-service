package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/maxviazov/basketball-stats-service/internal/config"
	"github.com/maxviazov/basketball-stats-service/internal/handler"
	"github.com/maxviazov/basketball-stats-service/internal/logger"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	repoPg "github.com/maxviazov/basketball-stats-service/internal/repository/postgres"
	"github.com/maxviazov/basketball-stats-service/internal/service"
)

func main() {
	// Load config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("❌ Config loading failed: %v", err)
	}

	// Logger
	appLogger, err := logger.New(&cfg.Logger)
	if err != nil {
		log.Fatalf("❌ Logger init failed: %v", err)
	}
	appLogger.Info().Fields(map[string]any{
		"service": cfg.App.Name,
		"version": cfg.App.Version,
		"env":     cfg.App.Env,
	}).Msg("🚀 Service started")

	// Repository (pgx pool)
	repo, err := repository.New(context.Background(), cfg, appLogger)
	if err != nil {
		appLogger.Fatal().Err(err).Msg("failed to init repository")
	}
	defer repo.Close()

	// Wire repositories (postgres implementations) and services
	pool := repo.Pool()
	teamRepo := repoPg.NewTeamRepository(pool)
	playerRepo := repoPg.NewPlayerRepository(pool)
	gameRepo := repoPg.NewGameRepository(pool)
	statsRepo := repoPg.NewStatsRepository(pool)
	txManager := repoPg.NewTxManager(pool)

	teamSvc := service.NewTeamService(teamRepo, appLogger)
	playerSvc := service.NewPlayerService(playerRepo, teamRepo, appLogger)
	gameSvc := service.NewGameService(gameRepo, teamRepo, txManager, appLogger)
	statsSvc := service.NewStatsService(statsRepo, playerRepo, gameRepo, txManager, appLogger)

	// HTTP server (Gin)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	handler.Register(r, repo, teamSvc, playerSvc, gameSvc, statsSvc)

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start server
	go func() {
		appLogger.Info().Str("addr", addr).Msg("HTTP server is starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal().Err(err).Msg("listen")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error().Err(err).Msg("server forced to shutdown")
	}

	appLogger.Info().Msg("Server exited")
}
