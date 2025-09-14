package main

import (
	"fmt"
	"log"

	"github.com/maxviazov/basketball-stats-service/internal/config"
	"github.com/maxviazov/basketball-stats-service/internal/logger"
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

	// Start service
	appLogger.Info().Msg("🚀 Service started")
	appLogger.Info().Msg("✅ Logger initialized successfully")
	appLogger.Info().Msg(fmt.Sprintf("%+v\n", cfg.Postgres))
	appLogger.Info().Msg("Config loaded successfully")
}
