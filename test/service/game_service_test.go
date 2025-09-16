package service_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/service"
)

type fakeGameRepo struct {
	nextID int64
	games  map[int64]model.Game
}

func newFakeGameRepo() *fakeGameRepo { return &fakeGameRepo{nextID: 1, games: map[int64]model.Game{}} }
func (f *fakeGameRepo) Create(_ context.Context, g model.Game) (model.Game, error) {
	g.ID = f.nextID
	f.nextID++
	f.games[g.ID] = g
	return g, nil
}
func (f *fakeGameRepo) GetByID(_ context.Context, id int64) (model.Game, error) {
	g, ok := f.games[id]
	if !ok {
		return model.Game{}, repository.ErrNotFound
	}
	return g, nil
}
func (f *fakeGameRepo) List(_ context.Context, _ repository.Page) (repository.PageResult[model.Game], error) {
	var res repository.PageResult[model.Game]
	for _, g := range f.games {
		res.Items = append(res.Items, g)
	}
	res.Total = len(res.Items)
	return res, nil
}

var _ repository.GameRepository = (*fakeGameRepo)(nil)

type fakeExistTeamRepo struct{ exist map[int64]bool }

func (f *fakeExistTeamRepo) Create(context.Context, model.Team) (model.Team, error) {
	return model.Team{}, nil
}
func (f *fakeExistTeamRepo) GetByID(_ context.Context, id int64) (model.Team, error) {
	if f.exist[id] {
		return model.Team{ID: id, Name: "T"}, nil
	}
	return model.Team{}, repository.ErrNotFound
}
func (f *fakeExistTeamRepo) List(context.Context, repository.Page) (repository.PageResult[model.Team], error) {
	return repository.PageResult[model.Team]{}, nil
}
func (f *fakeExistTeamRepo) GetTeamAggregatedStats(context.Context, int64, *string) (model.TeamAggregatedStats, error) {
	return model.TeamAggregatedStats{}, nil // Dummy implementation
}

var _ repository.TeamRepository = (*fakeExistTeamRepo)(nil)

type fakeTx struct{}

func (f *fakeTx) WithinTx(ctx context.Context, fn repository.TxFunc) error { return fn(ctx) }

var _ repository.TxManager = (*fakeTx)(nil)

func TestGameService_CreateGame_Validation(t *testing.T) {
	logger := zerolog.New(io.Discard)
	teamRepo := &fakeExistTeamRepo{exist: map[int64]bool{1: true, 2: true}}
	gameRepo := newFakeGameRepo()
	tx := &fakeTx{}
	svc := service.NewGameService(gameRepo, teamRepo, tx, logger)

	cases := []struct {
		name       string
		season     string
		date       time.Time
		home, away int64
		status     string
		wantErr    bool
		field      string
	}{
		{"same teams", "2025-26", time.Now(), 1, 1, "scheduled", true, "teams"},
		{"bad season", "2025", time.Now(), 1, 2, "scheduled", true, "season"},
		{"bad status", "2025-26", time.Now(), 1, 2, "bad", true, "status"},
		{"missing team", "2025-26", time.Now(), 1, 3, "scheduled", true, "away_team_id"},
		{"ok", "2025-26", time.Now(), 1, 2, "scheduled", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateGame(context.Background(), tc.season, tc.date, tc.home, tc.away, tc.status)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr {
				if !serviceErrIsInvalid(err) {
					t.Fatalf("expected invalid input")
				}
				if tc.field != "" {
					found := false
					for _, fe := range service.FieldErrors(err) {
						if fe.Field == tc.field {
							found = true
							break
						}
					}
					if !found {
						t.Fatalf("expected field %s", tc.field)
					}
				}
			}
		})
	}
}
