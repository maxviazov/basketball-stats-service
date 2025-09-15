package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maxviazov/basketball-stats-service/internal/config"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

func TestConfigLoad_FromYAMLAndEnv(t *testing.T) {
	// Minimal YAML; secrets will come from ENV
	yaml := `
app:
  name: basketball-stats-service
  version: 0.1.0
  env: test
  port: 18080

logger:
  level: info
  format: json
  output_target: stdout
  time_format: rfc3339
  with_caller: false
  stacktrace: false

postgres:
  host: 127.0.0.1
  port: 5432
  sslmode: disable
  max_conns: 5
  min_conns: 1
  max_conn_lifetime: 60
  max_conn_idle_time: 30
  health_check_period: 15
`
	path := writeTempConfig(t, yaml)

	// Provide required secrets via ENV using the canonical APP_* names
	t.Setenv("APP_POSTGRES_USER", "testuser")
	t.Setenv("APP_POSTGRES_PASSWORD", "testpass")
	t.Setenv("APP_POSTGRES_DB", "testdb")

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.Port != 18080 {
		t.Fatalf("expected app.port 18080, got %d", cfg.App.Port)
	}
	if cfg.Postgres.User != "testuser" || cfg.Postgres.Password != "testpass" || cfg.Postgres.DBName != "testdb" {
		t.Fatalf("env overrides not applied: got user=%q pass=%q db=%q", cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.DBName)
	}
	if cfg.Postgres.Host != "127.0.0.1" || cfg.Postgres.Port != 5432 || cfg.Postgres.SSLMode != "disable" {
		t.Fatalf("yaml values not loaded as expected: host=%q port=%d sslmode=%q", cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.SSLMode)
	}
}

func TestConfigLoad_MissingRequiredEnvFails(t *testing.T) {
	yaml := `
app:
  name: abc
  version: 0.0.0
  env: test
  port: 18080

logger:
  level: info

postgres:
  host: localhost
  port: 5432
  sslmode: disable
`
	path := writeTempConfig(t, yaml)

	// Ensure secrets are not set
	t.Setenv("APP_POSTGRES_USER", "")
	t.Setenv("APP_POSTGRES_PASSWORD", "")
	t.Setenv("APP_POSTGRES_DB", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_DB", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")

	_, err := config.Load(path)
	if err == nil {
		t.Fatalf("expected error when required env are missing, got nil")
	}
}
