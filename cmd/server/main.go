package main

import (
	"context"
	"log"

	"github.com/maxviazov/basketball-stats-service/internal/config"
	"github.com/maxviazov/basketball-stats-service/internal/logger"
	postgres "github.com/maxviazov/basketball-stats-service/internal/repository"
)

func main() {
	// Load application config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("❌ Config loading failed: %v", err)
	}

	// Initialize logger
	appLogger, err := logger.New(&cfg.Logger)
	if err != nil {
		log.Fatalf("❌ Logger initialization failed: %v", err)
	}

	connectPgx, err := postgres.New(context.Background(), cfg, &appLogger)
	if err != nil {
		log.Fatalf("❌ Postgres connection failed: %v", err)
	}
	defer connectPgx.Close()
	// Start service
	appLogger.Info().Msg("🚀 Service started")
	appLogger.Info().Msg("✅ Logger initialized successfully")
	appLogger.Info().Msg("Config loaded successfully")
}
