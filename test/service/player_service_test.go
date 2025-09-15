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

type fakePlayerRepo struct {
	nextID  int64
	players map[int64]model.Player
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

var _ repository.PlayerRepository = (*fakePlayerRepo)(nil)

type fakeLookupTeamRepo struct{ exists map[int64]bool }

func (f *fakeLookupTeamRepo) Create(context.Context, model.Team) (model.Team, error) {
	return model.Team{}, nil
}
func (f *fakeLookupTeamRepo) GetByID(_ context.Context, id int64) (model.Team, error) {
	if f.exists[id] {
		return model.Team{ID: id, Name: "X"}, nil
	}
	return model.Team{}, repository.ErrNotFound
}
func (f *fakeLookupTeamRepo) List(context.Context, repository.Page) (repository.PageResult[model.Team], error) {
	return repository.PageResult[model.Team]{}, nil
}

var _ repository.TeamRepository = (*fakeLookupTeamRepo)(nil)

func TestPlayerService_CreatePlayer_Validation(t *testing.T) {
	logger := zerolog.New(io.Discard)
	teamRepo := &fakeLookupTeamRepo{exists: map[int64]bool{10: true}}
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
