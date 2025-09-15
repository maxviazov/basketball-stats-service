package repository_test

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
	pg "github.com/maxviazov/basketball-stats-service/internal/repository/postgres"
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
		skippy = true
		os.Exit(m.Run())
	}
	// Build DSN from env first; no DSN -> skip to avoid false negatives in CI where DB is optional.
	dsn = buildDSNFromEnv()
	if dsn == "" {
		fmt.Println("[contract] missing DB env; skipping")
		skippy = true
		os.Exit(m.Run())
	}
	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		fmt.Println("sql open:", err)
		os.Exit(1)
	}
	if err := db.Ping(); err != nil { // early fail gives clearer feedback than later migration noise
		fmt.Println("db ping:", err)
		os.Exit(1)
	}
	// Correct relative path: test/repository -> project root is ../.. .
	// Previously we used one extra ".." which pointed outside the repo, causing goose to fail.
	migrationsDir := filepath.Clean(filepath.Join("..", "..", "migrations", "goose_sql"))
	if st, statErr := os.Stat(migrationsDir); statErr != nil || !st.IsDir() {
		fmt.Printf("[contract] migrations dir not found at %s (err=%v); skipping\n", migrationsDir, statErr)
		skippy = true
		os.Exit(m.Run())
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		fmt.Println("goose up:", err)
		os.Exit(1)
	}
	pool, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		fmt.Println("pool new:", err)
		os.Exit(1)
	}
	code := m.Run()
	pool.Close()
	_ = db.Close()
	os.Exit(code)
}

func skipIfNeeded(t *testing.T) {
	if skippy {
		t.Skip("contract tests skipped")
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
	stmts := []string{
		"TRUNCATE TABLE player_stats RESTART IDENTITY CASCADE",
		"TRUNCATE TABLE players RESTART IDENTITY CASCADE",
		"TRUNCATE TABLE games RESTART IDENTITY CASCADE",
		"TRUNCATE TABLE teams RESTART IDENTITY CASCADE",
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("truncate: %v", err)
		}
	}
}

func makeTeamRepo(t *testing.T) (repository.TeamRepository, func()) {
	skipIfNeeded(t)
	truncateAll(t)
	return pg.NewTeamRepository(pool), func() { truncateAll(t) }
}

func makePlayerRepo(t *testing.T) (repository.PlayerRepository, func(ctx context.Context, name string) (int64, error), func()) {
	skipIfNeeded(t)
	truncateAll(t)
	teamRepo := pg.NewTeamRepository(pool)
	mkTeam := func(ctx context.Context, name string) (int64, error) {
		tm, err := teamRepo.Create(ctx, model.Team{Name: name})
		if err != nil {
			return 0, err
		}
		return tm.ID, nil
	}
	return pg.NewPlayerRepository(pool), mkTeam, func() { truncateAll(t) }
}

func makeGameRepo(t *testing.T) (repository.GameRepository, func(ctx context.Context, name string) (int64, error), func()) {
	skipIfNeeded(t)
	truncateAll(t)
	teamRepo := pg.NewTeamRepository(pool)
	mkTeam := func(ctx context.Context, name string) (int64, error) {
		tm, err := teamRepo.Create(ctx, model.Team{Name: name})
		if err != nil {
			return 0, err
		}
		return tm.ID, nil
	}
	return pg.NewGameRepository(pool), mkTeam, func() { truncateAll(t) }
}

func makeStatsRepo(t *testing.T) (repository.StatsRepository, func(ctx context.Context) (int64, error), func(ctx context.Context) (int64, error), func()) {
	skipIfNeeded(t)
	truncateAll(t)
	teamRepo := pg.NewTeamRepository(pool)
	playerRepo := pg.NewPlayerRepository(pool)
	gameRepo := pg.NewGameRepository(pool)
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
		g, err := gameRepo.Create(ctx, model.Game{Season: "2025-26", Date: time.Now().UTC(), HomeTeamID: h.ID, AwayTeamID: a.ID, Status: "scheduled"})
		if err != nil {
			return 0, err
		}
		return g.ID, nil
	}
	return pg.NewStatsRepository(pool), mkPlayer, mkGame, func() { truncateAll(t) }
}

func makeTx(t *testing.T) (repository.TxManager, repository.TeamRepository, func()) {
	skipIfNeeded(t)
	truncateAll(t)
	return pg.NewTxManager(pool), pg.NewTeamRepository(pool), func() { truncateAll(t) }
}

func makePinger(t *testing.T) (repository.Pinger, func()) {
	skipIfNeeded(t)
	return pg.NewPinger(pool), func() {}
}

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
func TestTxManager_PostgresContract(t *testing.T) { contract.RunTxManagerContract(t, makeTx) }
func TestPinger_PostgresContract(t *testing.T)    { contract.RunPingerContract(t, makePinger) }
