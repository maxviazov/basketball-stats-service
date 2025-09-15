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

// Repository wraps a pgx connection pool and exposes minimal DB primitives I actually need.
// I prefer to hide the pool behind this type to keep call sites decoupled from pgx specifics.
type Repository struct {
	pool *pgxpool.Pool
}

// Pool exposes the underlying pgx pool for repository wiring.
// I keep the type opaque elsewhere, but allow explicit access at composition roots.
func (r *Repository) Pool() *pgxpool.Pool { return r.pool }

// New builds a new Postgres-backed repository using pgxpool.
// I construct the DSN explicitly to avoid subtle quoting issues and then fine-tune the pool.
func New(ctx context.Context, cfg *config.Config, logger zerolog.Logger) (*Repository, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	// 1) Build DSN via url.URL so credentials and options are properly escaped.
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

	// 2) Parse the DSN into a pool config.
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	// 3) Wire up pgx tracelog so I can observe SQL at the right verbosity.
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
		Logger:   newPgxLogger(logger),
		LogLevel: tlLevel,
	}

	// 4) Apply pool tuning from config. These are safe, conservative defaults.
	poolConfig.MaxConns = cfg.Postgres.MaxConns
	poolConfig.MinConns = cfg.Postgres.MinConns
	poolConfig.MaxConnLifetime = time.Duration(cfg.Postgres.MaxConnLifetime) * time.Second
	poolConfig.MaxConnIdleTime = time.Duration(cfg.Postgres.MaxConnIdleTime) * time.Second
	poolConfig.HealthCheckPeriod = time.Duration(cfg.Postgres.HealthCheckPeriod) * time.Second

	// 5) Create the pool.
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// 6) Probe connectivity on startup with a small timeout to fail fast.
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

// Ping is a lightweight readiness check I use to verify DB connectivity on demand.
func (r *Repository) Ping(ctx context.Context) error {
	if r == nil || r.pool == nil {
		return errors.New("postgres pool is not initialized")
	}
	return r.pool.Ping(ctx)
}

// Close releases all resources held by the pool.
func (r *Repository) Close() {
	if r.pool != nil {
		r.pool.Close()
	}
}
