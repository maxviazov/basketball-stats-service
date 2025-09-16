package service_test

import (
	"context"
	"io"
	"testing"

	"github.com/rs/zerolog"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/service"
)

type fakeStatsRepo struct{}

func (f *fakeStatsRepo) UpsertStatLine(_ context.Context, s model.PlayerStatLine) (model.PlayerStatLine, error) {
	s.ID = 1
	return s, nil
}
func (f *fakeStatsRepo) ListByGame(_ context.Context, gameID int64) ([]model.PlayerStatLine, error) {
	return []model.PlayerStatLine{}, nil
}

var _ repository.StatsRepository = (*fakeStatsRepo)(nil)

type fakePlayerLookup struct{ ok map[int64]bool }

func (f *fakePlayerLookup) Create(context.Context, model.Player) (model.Player, error) {
	return model.Player{}, nil
}
func (f *fakePlayerLookup) GetByID(_ context.Context, id int64) (model.Player, error) {
	if f.ok[id] {
		return model.Player{ID: id}, nil
	}
	return model.Player{}, repository.ErrNotFound
}
func (f *fakePlayerLookup) ListByTeam(context.Context, int64, repository.Page) (repository.PageResult[model.Player], error) {
	return repository.PageResult[model.Player]{}, nil
}
func (f *fakePlayerLookup) GetPlayerAggregatedStats(context.Context, int64, *string) (model.PlayerAggregatedStats, error) {
	return model.PlayerAggregatedStats{}, nil // Dummy implementation
}

var _ repository.PlayerRepository = (*fakePlayerLookup)(nil)

type fakeGameLookup struct{ ok map[int64]bool }

func (f *fakeGameLookup) Create(context.Context, model.Game) (model.Game, error) {
	return model.Game{}, nil
}
func (f *fakeGameLookup) GetByID(_ context.Context, id int64) (model.Game, error) {
	if f.ok[id] {
		return model.Game{ID: id}, nil
	}
	return model.Game{}, repository.ErrNotFound
}
func (f *fakeGameLookup) List(context.Context, repository.Page) (repository.PageResult[model.Game], error) {
	return repository.PageResult[model.Game]{}, nil
}

var _ repository.GameRepository = (*fakeGameLookup)(nil)

type fakeTxStats struct{}

func (f *fakeTxStats) WithinTx(ctx context.Context, fn repository.TxFunc) error { return fn(ctx) }

var _ repository.TxManager = (*fakeTxStats)(nil)

func TestStatsService_UpsertStatLine_Validation(t *testing.T) {
	logger := zerolog.New(io.Discard)
	statsRepo := &fakeStatsRepo{}
	players := &fakePlayerLookup{ok: map[int64]bool{2: true}}
	games := &fakeGameLookup{ok: map[int64]bool{3: true}}
	tx := &fakeTxStats{}
	svc := service.NewStatsService(statsRepo, players, games, tx, logger)

	cases := []struct {
		name    string
		line    model.PlayerStatLine
		wantErr bool
		field   string
	}{
		{"bad ids", model.PlayerStatLine{PlayerID: 0, GameID: 0}, true, "player_id"},
		{"negative stat", model.PlayerStatLine{PlayerID: 2, GameID: 3, Points: -1}, true, "points"},
		{"player missing", model.PlayerStatLine{PlayerID: 9, GameID: 3}, true, "player_id"},
		{"game missing", model.PlayerStatLine{PlayerID: 2, GameID: 99}, true, "game_id"},
		{"ok", model.PlayerStatLine{PlayerID: 2, GameID: 3, Points: 10}, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.UpsertStatLine(context.Background(), tc.line)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr {
				if !serviceErrIsInvalid(err) {
					t.Fatalf("want invalid input err")
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
						t.Fatalf("missing field error %s", tc.field)
					}
				}
			}
		})
	}
}
