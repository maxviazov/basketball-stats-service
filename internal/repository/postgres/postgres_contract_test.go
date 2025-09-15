package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/repository/contract"
	"github.com/pressly/goose/v3"
)

var (
	db     *sql.DB
	pool   *pgxpool.Pool
	dsn    string
	skippy bool
)

func TestMain(m *testing.M) {
	if os.Getenv("CONTRACT_TESTS") != "1" {
		// allow skipping contract tests unless explicitly enabled
		skippy = true
		os.Exit(m.Run())
	}

	dsn = buildDSNFromEnv()
	if dsn == "" {
		fmt.Println("[contract] DATABASE_URL or APP_POSTGRES_* env not set; skipping")
		skippy = true
		os.Exit(m.Run())
	}

	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		fmt.Println("[contract] sql open error:", err)
		os.Exit(1)
	}
	if err := db.Ping(); err != nil {
		fmt.Println("[contract] db ping error:", err)
		os.Exit(1)
	}

	// Run migrations up
	migrationsDir := filepath.Clean(filepath.Join("..", "..", "..", "migrations", "goose_sql"))
	if err := goose.Up(db, migrationsDir); err != nil {
		fmt.Println("[contract] goose up error:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Println("[contract] pgxpool new error:", err)
		os.Exit(1)
	}

	code := m.Run()
	pool.Close()
	db.Close()
	os.Exit(code)
}

func skipIfNeeded(t *testing.T) {
	if skippy {
		t.Skip("contract tests skipped; set CONTRACT_TESTS=1 and provide DB env")
	}
}

func buildDSNFromEnv() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	user := firstNonEmpty(os.Getenv("APP_POSTGRES_USER"), os.Getenv("POSTGRES_USER"), os.Getenv("DB_USER"))
	pass := firstNonEmpty(os.Getenv("APP_POSTGRES_PASSWORD"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("DB_PASSWORD"))
	host := firstNonEmpty(os.Getenv("APP_POSTGRES_HOST"), os.Getenv("POSTGRES_HOST"), "localhost")
	port := firstNonEmpty(os.Getenv("APP_POSTGRES_PORT"), os.Getenv("POSTGRES_PORT"), "5432")
	db := firstNonEmpty(os.Getenv("APP_POSTGRES_DB"), os.Getenv("POSTGRES_DB"), os.Getenv("DB_NAME"))
	ssl := firstNonEmpty(os.Getenv("APP_POSTGRES_SSLMODE"), os.Getenv("POSTGRES_SSLMODE"), "disable")
	if user == "" || pass == "" || db == "" {
		return ""
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, pass, host, port, db, ssl)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func truncateAll(t *testing.T) {
	t.Helper()
	stmts := []string{
		"TRUNCATE TABLE player_stats RESTART IDENTITY CASCADE",
		"TRUNCATE TABLE players RESTART IDENTITY CASCADE",
		"TRUNCATE TABLE games RESTART IDENTITY CASCADE",
		"TRUNCATE TABLE teams RESTART IDENTITY CASCADE",
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("truncate failed: %v", err)
		}
	}
}

// Factories used by contract suites

func makeTeamRepo(t *testing.T) (repository.TeamRepository, func()) {
	skipIfNeeded(t)
	truncateAll(t)
	return NewTeamRepository(pool), func() { truncateAll(t) }
}

func makePlayerRepo(t *testing.T) (repository.PlayerRepository, func(ctx context.Context, name string) (int64, error), func()) {
	skipIfNeeded(t)
	truncateAll(t)
	teamRepo := NewTeamRepository(pool)
	makeTeam := func(ctx context.Context, name string) (int64, error) {
		team, err := teamRepo.Create(ctx, model.Team{Name: name})
		if err != nil {
			return 0, err
		}
		return team.ID, nil
	}
	return NewPlayerRepository(pool), makeTeam, func() { truncateAll(t) }
}

func makeGameRepo(t *testing.T) (repository.GameRepository, func(ctx context.Context, name string) (int64, error), func()) {
	skipIfNeeded(t)
	truncateAll(t)
	teamRepo := NewTeamRepository(pool)
	makeTeam := func(ctx context.Context, name string) (int64, error) {
		team, err := teamRepo.Create(ctx, model.Team{Name: name})
		if err != nil {
			return 0, err
		}
		return team.ID, nil
	}
	return NewGameRepository(pool), makeTeam, func() { truncateAll(t) }
}

func makeStatsRepo(t *testing.T) (repository.StatsRepository, func(ctx context.Context) (int64, error), func(ctx context.Context) (int64, error), func()) {
	skipIfNeeded(t)
	truncateAll(t)
	teamRepo := NewTeamRepository(pool)
	playerRepo := NewPlayerRepository(pool)
	gameRepo := NewGameRepository(pool)
	mkPlayer := func(ctx context.Context) (int64, error) {
		team, err := teamRepo.Create(ctx, model.Team{Name: "SeedTeam"})
		if err != nil {
			return 0, err
		}
		p, err := playerRepo.Create(ctx, model.Player{TeamID: team.ID, FirstName: "John", LastName: "Doe", Position: "SG"})
		if err != nil {
			return 0, err
		}
		return p.ID, nil
	}
	mkGame := func(ctx context.Context) (int64, error) {
		h, _ := teamRepo.Create(ctx, model.Team{Name: "Home"})
		a, _ := teamRepo.Create(ctx, model.Team{Name: "Away"})
		g, err := gameRepo.Create(ctx, model.Game{Season: "2025", Date: time.Now().UTC(), HomeTeamID: h.ID, AwayTeamID: a.ID, Status: "scheduled"})
		if err != nil {
			return 0, err
		}
		return g.ID, nil
	}
	return NewStatsRepository(pool), mkPlayer, mkGame, func() { truncateAll(t) }
}

func makeTx(t *testing.T) (repository.TxManager, repository.TeamRepository, func()) {
	skipIfNeeded(t)
	truncateAll(t)
	return NewTxManager(pool), NewTeamRepository(pool), func() { truncateAll(t) }
}

func makePinger(t *testing.T) (repository.Pinger, func()) {
	skipIfNeeded(t)
	return NewPinger(pool), func() {}
}

// Wire the contract suites to Postgres factories

func TestTeamRepository_PostgresContract(t *testing.T) {
	contract.RunTeamRepositoryContract(t, makeTeamRepo)
}

func TestPlayerRepository_PostgresContract(t *testing.T) {
	contract.RunPlayerRepositoryContract(t, makePlayerRepo)
}

func TestGameRepository_PostgresContract(t *testing.T) {
	contract.RunGameRepositoryContract(t, makeGameRepo)
}

func TestStatsRepository_PostgresContract(t *testing.T) {
	contract.RunStatsRepositoryContract(t, makeStatsRepo)
}

func TestTxManager_PostgresContract(t *testing.T) {
	contract.RunTxManagerContract(t, makeTx)
}

func TestPinger_PostgresContract(t *testing.T) {
	contract.RunPingerContract(t, makePinger)
}
