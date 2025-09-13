package config

import (
	"github.com/maxviazov/basketball-stats-service/internal/logger"
)

type Config struct {
	Logger logger.LoggerConfig `mapstructure:"logger"`
}
