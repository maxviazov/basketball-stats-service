package service_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/service"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

type fakePlayerRepo struct {
	nextID      int64
	players     map[int64]model.Player
	statsResult model.PlayerAggregatedStats
	statsErr    error
}

func newFakePlayerRepo() *fakePlayerRepo {
	return &fakePlayerRepo{nextID: 1, players: map[int64]model.Player{}}
}
func (f *fakePlayerRepo) Create(_ context.Context, p model.Player) (model.Player, error) {
	p.ID = f.nextID
	f.nextID++
	f.players[p.ID] = p
	return p, nil
}
func (f *fakePlayerRepo) GetByID(_ context.Context, id int64) (model.Player, error) {
	p, ok := f.players[id]
	if !ok {
		return model.Player{}, repository.ErrNotFound
	}
	return p, nil
}
func (f *fakePlayerRepo) ListByTeam(_ context.Context, teamID int64, _ repository.Page) (repository.PageResult[model.Player], error) {
	var res repository.PageResult[model.Player]
	for _, p := range f.players {
		if p.TeamID == teamID {
			res.Items = append(res.Items, p)
		}
	}
	res.Total = len(res.Items)
	return res, nil
}

func (f *fakePlayerRepo) GetPlayerAggregatedStats(_ context.Context, playerID int64, _ *string) (model.PlayerAggregatedStats, error) {
	if f.statsErr != nil {
		return model.PlayerAggregatedStats{}, f.statsErr
	}
	if _, ok := f.players[playerID]; !ok {
		return model.PlayerAggregatedStats{}, repository.ErrNotFound
	}
	return f.statsResult, nil
}

var _ repository.PlayerRepository = (*fakePlayerRepo)(nil)

type fakeLookupTeamRepo struct{
	exists map[int64]bool
	// Embed the full fake repo to satisfy the interface without having to implement all methods.
	*fakeTeamRepo
}

func newFakeLookupTeamRepo(ids ...int64) *fakeLookupTeamRepo {
	m := make(map[int64]bool)
	for _, id := range ids {
		m[id] = true
	}
	return &fakeLookupTeamRepo{exists: m, fakeTeamRepo: newFakeTeamRepo()}
}

func (f *fakeLookupTeamRepo) GetByID(_ context.Context, id int64) (model.Team, error) {
	if f.exists[id] {
		return model.Team{ID: id, Name: "X"}, nil
	}
	return model.Team{}, repository.ErrNotFound
}

func TestPlayerService_CreatePlayer_Validation(t *testing.T) {
	logger := zerolog.New(io.Discard)
	teamRepo := newFakeLookupTeamRepo(10)
	playerRepo := newFakePlayerRepo()
	svc := service.NewPlayerService(playerRepo, teamRepo, logger)
	cases := []struct {
		name        string
		teamID      int64
		pos, fn, ln string
		wantErr     bool
		field       string
	}{
		{"bad team", -1, "PG", "J", "D", true, "team_id"},
		{"missing name", 10, "PG", "", "Doe", true, "first_name"},
		{"bad position", 10, "XX", "John", "Doe", true, "position"},
		{"team not exists", 99, "PG", "John", "Doe", true, "team_id"},
		{"ok", 10, "pg", "John", "Doe", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreatePlayer(context.Background(), tc.teamID, tc.fn, tc.ln, tc.pos)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr {
				if !serviceErrIsInvalid(err) {
					t.Fatalf("want invalid input")
				}
				found := false
				for _, fe := range service.FieldErrors(err) {
					if fe.Field == tc.field {
						found = true
						break
					}
				}
				if tc.field != "" && !found {
					t.Fatalf("field %s not reported", tc.field)
				}
			}
		})
	}
}

func TestPlayerService_GetPlayerAggregatedStats(t *testing.T) {
	logger := zerolog.New(io.Discard)
	playerRepo := newFakePlayerRepo()
	teamRepo := newFakeLookupTeamRepo()
	svc := service.NewPlayerService(playerRepo, teamRepo, logger)

	// Seed a player for valid ID checks
	_, err := playerRepo.Create(context.Background(), model.Player{ID: 1, TeamID: 1, FirstName: "Test"})
	require.NoError(t, err)

	t.Run("Valid Request", func(t *testing.T) {
		expected := model.PlayerAggregatedStats{GamesPlayed: 82, TotalPoints: 2500}
		playerRepo.statsResult = expected
		playerRepo.statsErr = nil

		stats, err := svc.GetPlayerAggregatedStats(context.Background(), 1, nil)
		require.NoError(t, err)
		require.Equal(t, expected, stats)
	})

	t.Run("Invalid Player ID", func(t *testing.T) {
		_, err := svc.GetPlayerAggregatedStats(context.Background(), 0, nil)
		require.Error(t, err)
		require.True(t, serviceErrIsInvalid(err), "expected invalid input error")
		fields := service.FieldErrors(err)
		require.Len(t, fields, 1)
		require.Equal(t, "id", fields[0].Field)
	})

	t.Run("Invalid Season Format", func(t *testing.T) {
		invalidSeason := "2023/24"
		_, err := svc.GetPlayerAggregatedStats(context.Background(), 1, &invalidSeason)
		require.Error(t, err)
		require.True(t, serviceErrIsInvalid(err), "expected invalid input error")
		fields := service.FieldErrors(err)
		require.Len(t, fields, 1)
		require.Equal(t, "season", fields[0].Field)
	})

	t.Run("Repository Error", func(t *testing.T) {
		playerRepo.statsErr = errors.New("db is down")
		_, err := svc.GetPlayerAggregatedStats(context.Background(), 1, nil)
		require.Error(t, err)
		require.False(t, serviceErrIsInvalid(err))
		require.Equal(t, "db is down", err.Error())
	})
}
