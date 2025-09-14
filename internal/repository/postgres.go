package repository

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/maxviazov/basketball-stats-service/internal/config"
	"github.com/rs/zerolog"
)

// Repository инкапсулирует пул соединений pgx.
type Repository struct {
	pool *pgxpool.Pool
}

// New создает новый репозиторий Postgres с пулом соединений.
func New(ctx context.Context, cfg *config.Config, logger *zerolog.Logger) (*Repository, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	// 1. Собираем DSN через url.URL для корректного экранирования.
	u := url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", cfg.Postgres.Host, cfg.Postgres.Port),
		Path:   cfg.Postgres.DBName,
	}
	if cfg.Postgres.User != "" || cfg.Postgres.Password != "" {
		u.User = url.UserPassword(cfg.Postgres.User, cfg.Postgres.Password)
	}
	q := u.Query()
	if cfg.Postgres.SSLMode != "" {
		q.Set("sslmode", cfg.Postgres.SSLMode)
	}
	u.RawQuery = q.Encode()
	dsn := u.String()

	// 2. Парсим DSN в конфигурацию пула соединений.
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	// 3. Настраиваем трассировщик pgx через tracelog.
	var tlLevel tracelog.LogLevel
	switch {
	case logger.GetLevel() <= zerolog.TraceLevel:
		tlLevel = tracelog.LogLevelTrace
	case logger.GetLevel() <= zerolog.DebugLevel:
		tlLevel = tracelog.LogLevelDebug
	case logger.GetLevel() <= zerolog.InfoLevel:
		tlLevel = tracelog.LogLevelInfo
	case logger.GetLevel() <= zerolog.WarnLevel:
		tlLevel = tracelog.LogLevelWarn
	default:
		tlLevel = tracelog.LogLevelError
	}
	poolConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   newPgxLogger(*logger),
		LogLevel: tlLevel,
	}

	// 4. Применяем параметры тюнинга пула.
	poolConfig.MaxConns = cfg.Postgres.MaxConns
	poolConfig.MinConns = cfg.Postgres.MinConns
	poolConfig.MaxConnLifetime = time.Duration(cfg.Postgres.MaxConnLifetime) * time.Second
	poolConfig.MaxConnIdleTime = time.Duration(cfg.Postgres.MaxConnIdleTime) * time.Second
	poolConfig.HealthCheckPeriod = time.Duration(cfg.Postgres.HealthCheckPeriod) * time.Second

	// 5. Создаем пул.
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// 6. Проверяем соединение с таймаутом, чтобы не зависнуть на старте.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	logger.Info().
		Str("host", cfg.Postgres.Host).
		Int("port", cfg.Postgres.Port).
		Str("user", cfg.Postgres.User).
		Str("db", cfg.Postgres.DBName).
		Msg("Successfully connected to PostgreSQL")

	return &Repository{pool: pool}, nil
}

// Close освобождает все ресурсы пула соединений.
func (r *Repository) Close() {
	if r.pool != nil {
		r.pool.Close()
	}
}
