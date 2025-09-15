package contract

import (
	"context"
	"testing"
	"time"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
)

// Team contracts

type TeamFactory func(t *testing.T) (repository.TeamRepository, func())

type PlayerFactory func(t *testing.T) (repo repository.PlayerRepository, createTeam func(ctx context.Context, name string) (int64, error), cleanup func())

type GameFactory func(t *testing.T) (repo repository.GameRepository, createTeam func(ctx context.Context, name string) (int64, error), cleanup func())

type StatsFactory func(t *testing.T) (repo repository.StatsRepository, mkPlayer func(ctx context.Context) (int64, error), mkGame func(ctx context.Context) (int64, error), cleanup func())

type TxFactory func(t *testing.T) (tx repository.TxManager, teams repository.TeamRepository, cleanup func())

type PingerFactory func(t *testing.T) (repository.Pinger, func())

func RunTeamRepositoryContract(t *testing.T, makeRepo TeamFactory) {
	t.Helper()

	t.Run("create_and_get", func(t *testing.T) {
		repo, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		ctx := context.Background()
		created, err := repo.Create(ctx, model.Team{Name: "Warriors"})
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if got.ID != created.ID || got.Name != created.Name {
			t.Fatalf("mismatch: %+v", got)
		}
	})

	t.Run("get_not_found", func(t *testing.T) {
		repo, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		_, err := repo.GetByID(context.Background(), 999999)
		if err == nil || err != repository.ErrNotFound {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("list_pagination_total", func(t *testing.T) {
		repo, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		ctx := context.Background()
		for i := 0; i < 7; i++ {
			name := "T-" + string(rune('A'+i))
			if _, err := repo.Create(ctx, model.Team{Name: name}); err != nil {
				t.Fatalf("seed: %v", err)
			}
		}
		res, err := repo.List(ctx, repository.Page{Limit: 3, Offset: 0})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(res.Items) != 3 || res.Total != 7 {
			t.Fatalf("unexpected page: len=%d total=%d", len(res.Items), res.Total)
		}
		res2, err := repo.List(ctx, repository.Page{Limit: 3, Offset: 3})
		if err != nil {
			t.Fatalf("list2: %v", err)
		}
		if len(res2.Items) != 3 || res2.Total != 7 {
			t.Fatalf("unexpected page2: len=%d total=%d", len(res2.Items), res2.Total)
		}
	})

	t.Run("create_duplicate_name_conflict", func(t *testing.T) {
		repo, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		ctx := context.Background()
		_, err := repo.Create(ctx, model.Team{Name: "Dup"})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		_, err = repo.Create(ctx, model.Team{Name: "Dup"})
		if err == nil || err != repository.ErrAlreadyExists {
			t.Fatalf("expected ErrAlreadyExists, got %v", err)
		}
	})
}

func RunPlayerRepositoryContract(t *testing.T, makeRepo PlayerFactory) {
	t.Helper()

	t.Run("create_and_get", func(t *testing.T) {
		repo, mkTeam, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		ctx := context.Background()
		teamID, err := mkTeam(ctx, "Bulls")
		if err != nil {
			t.Fatalf("seed team: %v", err)
		}
		created, err := repo.Create(ctx, model.Player{TeamID: teamID, FirstName: "Michael", LastName: "Jordan", Position: "SG"})
		if err != nil {
			t.Fatalf("create player: %v", err)
		}
		got, err := repo.GetByID(ctx, created.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.ID != created.ID || got.TeamID != teamID {
			t.Fatalf("mismatch: %+v", got)
		}
	})

	t.Run("get_not_found", func(t *testing.T) {
		repo, _, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		_, err := repo.GetByID(context.Background(), 42424242)
		if err == nil || err != repository.ErrNotFound {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("list_by_team_pagination", func(t *testing.T) {
		repo, mkTeam, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		ctx := context.Background()
		teamID, err := mkTeam(ctx, "Lakers")
		if err != nil {
			t.Fatalf("seed team: %v", err)
		}
		for i := 0; i < 5; i++ {
			p := model.Player{TeamID: teamID, FirstName: "P", LastName: string(rune('A' + i)), Position: "SF"}
			if _, err := repo.Create(ctx, p); err != nil {
				t.Fatalf("seed player %d: %v", i, err)
			}
		}
		res, err := repo.ListByTeam(ctx, teamID, repository.Page{Limit: 2, Offset: 0})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(res.Items) != 2 || res.Total != 5 {
			t.Fatalf("unexpected page: len=%d total=%d", len(res.Items), res.Total)
		}
	})

	t.Run("create_fk_violation_conflict", func(t *testing.T) {
		repo, _, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		_, err := repo.Create(context.Background(), model.Player{TeamID: 9999999, FirstName: "X", LastName: "Y", Position: "PG"})
		if err == nil || err != repository.ErrConflict {
			t.Fatalf("expected ErrConflict on FK violation, got %v", err)
		}
	})
}

func RunGameRepositoryContract(t *testing.T, makeRepo GameFactory) {
	t.Helper()

	t.Run("create_get_list", func(t *testing.T) {
		repo, mkTeam, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		ctx := context.Background()
		homeID, _ := mkTeam(ctx, "Home")
		awayID, _ := mkTeam(ctx, "Away")
		g, err := repo.Create(ctx, model.Game{Season: "2025", Date: time.Now().UTC(), HomeTeamID: homeID, AwayTeamID: awayID, Status: "scheduled"})
		if err != nil {
			t.Fatalf("create game: %v", err)
		}
		got, err := repo.GetByID(ctx, g.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.ID != g.ID || got.HomeTeamID != homeID || got.AwayTeamID != awayID {
			t.Fatalf("mismatch: %+v", got)
		}
		page, err := repo.List(ctx, repository.Page{Limit: 10, Offset: 0})
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(page.Items) < 1 || page.Total < 1 {
			t.Fatalf("unexpected list: %#v", page)
		}
	})

	t.Run("get_not_found", func(t *testing.T) {
		repo, _, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		_, err := repo.GetByID(context.Background(), 7777777)
		if err == nil || err != repository.ErrNotFound {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func RunStatsRepositoryContract(t *testing.T, makeRepo StatsFactory) {
	t.Helper()

	t.Run("upsert_and_list", func(t *testing.T) {
		repo, mkPlayer, mkGame, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		ctx := context.Background()
		pid, err := mkPlayer(ctx)
		if err != nil {
			t.Fatalf("mkPlayer: %v", err)
		}
		gid, err := mkGame(ctx)
		if err != nil {
			t.Fatalf("mkGame: %v", err)
		}
		line := model.PlayerStatLine{PlayerID: pid, GameID: gid, Points: 10}
		l1, err := repo.UpsertStatLine(ctx, line)
		if err != nil {
			t.Fatalf("upsert1: %v", err)
		}
		if l1.Points != 10 {
			t.Fatalf("unexpected points: %d", l1.Points)
		}
		line.Points = 22
		l2, err := repo.UpsertStatLine(ctx, line)
		if err != nil {
			t.Fatalf("upsert2: %v", err)
		}
		if l2.Points != 22 {
			t.Fatalf("upsert didn't update points: %d", l2.Points)
		}
		list, err := repo.ListByGame(ctx, gid)
		if err != nil {
			t.Fatalf("list by game: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 line, got %d", len(list))
		}
	})

	t.Run("list_empty_ok", func(t *testing.T) {
		repo, _, mkGame, cleanup := makeRepo(t)
		t.Cleanup(cleanup)
		ctx := context.Background()
		gid, err := mkGame(ctx)
		if err != nil {
			t.Fatalf("mkGame: %v", err)
		}
		list, err := repo.ListByGame(ctx, gid)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(list) != 0 {
			t.Fatalf("expected empty list, got %d", len(list))
		}
	})
}

func RunTxManagerContract(t *testing.T, makeTx TxFactory) {
	t.Helper()

	t.Run("commit_on_nil_error", func(t *testing.T) {
		tx, teams, cleanup := makeTx(t)
		t.Cleanup(cleanup)
		ctx := context.Background()
		var createdID int64
		err := tx.WithinTx(ctx, func(ctx context.Context) error {
			out, err := teams.Create(ctx, model.Team{Name: "TxCommit"})
			if err != nil {
				return err
			}
			createdID = out.ID
			return nil
		})
		if err != nil {
			t.Fatalf("WithinTx: %v", err)
		}
		if _, err := teams.GetByID(ctx, createdID); err != nil {
			t.Fatalf("expected committed row visible, got err=%v", err)
		}
	})

	t.Run("rollback_on_error", func(t *testing.T) {
		tx, teams, cleanup := makeTx(t)
		t.Cleanup(cleanup)
		ctx := context.Background()
		var createdID int64
		errMarker := assertErr("boom")
		err := tx.WithinTx(ctx, func(ctx context.Context) error {
			out, err := teams.Create(ctx, model.Team{Name: "TxRollback"})
			if err != nil {
				return err
			}
			createdID = out.ID
			return errMarker
		})
		if err == nil || err.Error() != errMarker.Error() {
			t.Fatalf("expected marker error, got %v", err)
		}
		if _, err := teams.GetByID(ctx, createdID); err == nil || err != repository.ErrNotFound {
			t.Fatalf("expected ErrNotFound after rollback, got %v", err)
		}
	})
}

func RunPingerContract(t *testing.T, makePinger PingerFactory) {
	t.Helper()
	t.Run("ping_ok", func(t *testing.T) {
		p, cleanup := makePinger(t)
		t.Cleanup(cleanup)
		if err := p.Ping(context.Background()); err != nil {
			t.Fatalf("expected ping ok, got %v", err)
		}
	})
}

// assertErr builds a sentinel error without importing errors to keep helpers local.
func assertErr(msg string) error { return &sentinel{msg} }

type sentinel struct{ s string }

func (e *sentinel) Error() string { return e.s }
