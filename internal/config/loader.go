package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Load loads configuration from YAML file and overrides with environment variables if present.
func Load(path string) (*Config, error) {
	v := viper.New()

	// Path to the config file
	v.SetConfigFile(path)

	// Map environment variables like APP_POSTGRES_USER -> postgres.user
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// All env vars must be prefixed with APP_
	v.SetEnvPrefix("APP")

	// Enable automatic environment variable binding
	v.AutomaticEnv()

	// Support multiple env var names for convenience across environments
	// postgres.user can be provided by any of: APP_POSTGRES_USER, POSTGRES_USER, DB_USER
	if err := v.BindEnv("postgres.user", "APP_POSTGRES_USER", "POSTGRES_USER", "DB_USER"); err != nil {
		return nil, err
	}
	// postgres.password: APP_POSTGRES_PASSWORD, POSTGRES_PASSWORD, DB_PASSWORD
	if err := v.BindEnv("postgres.password", "APP_POSTGRES_PASSWORD", "POSTGRES_PASSWORD", "DB_PASSWORD"); err != nil {
		return nil, err
	}
	// postgres.dbname: APP_POSTGRES_DB, POSTGRES_DB, DB_NAME
	if err := v.BindEnv("postgres.dbname", "APP_POSTGRES_DB", "POSTGRES_DB", "DB_NAME"); err != nil {
		return nil, err
	}
	// Optional: host/port/sslmode overrides
	_ = v.BindEnv("postgres.host", "APP_POSTGRES_HOST", "POSTGRES_HOST")
	_ = v.BindEnv("postgres.port", "APP_POSTGRES_PORT", "POSTGRES_PORT")
	_ = v.BindEnv("postgres.sslmode", "APP_POSTGRES_SSLMODE", "POSTGRES_SSLMODE")
	// app.port: allow APP_APP_PORT or APP_PORT
	_ = v.BindEnv("app.port", "APP_APP_PORT", "APP_PORT")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config file not found: %w", err)
	}

	// Ensure env-bound keys are materialized so Unmarshal sees them even if absent in YAML
	for _, key := range []string{
		"postgres.user",
		"postgres.password",
		"postgres.dbname",
		"postgres.host",
		"postgres.port",
		"postgres.sslmode",
		"app.port",
	} {
		if val := v.GetString(key); val != "" {
			v.Set(key, val)
		}
	}

	var config Config
	// Unmarshal into our strongly typed Config struct
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required secrets
	if config.Postgres.User == "" || config.Postgres.Password == "" || config.Postgres.DBName == "" {
		return nil, errors.New("missing required env: set one of [APP_POSTGRES_USER|POSTGRES_USER|DB_USER], [APP_POSTGRES_PASSWORD|POSTGRES_PASSWORD|DB_PASSWORD], [APP_POSTGRES_DB|POSTGRES_DB|DB_NAME]")
	}

	return &config, nil
}
