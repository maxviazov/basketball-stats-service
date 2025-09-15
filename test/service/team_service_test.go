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

type fakeTeamRepo struct {
	nextID    int64
	items     map[int64]model.Team
	createErr error
	lastPage  repository.Page // capture last page for pagination normalization tests
}

func newFakeTeamRepo() *fakeTeamRepo {
	return &fakeTeamRepo{nextID: 1, items: map[int64]model.Team{}}
}

func (f *fakeTeamRepo) Create(_ context.Context, t model.Team) (model.Team, error) {
	if f.createErr != nil {
		return model.Team{}, f.createErr
	}
	t.ID = f.nextID
	f.nextID++
	f.items[t.ID] = t
	return t, nil
}
func (f *fakeTeamRepo) GetByID(_ context.Context, id int64) (model.Team, error) {
	it, ok := f.items[id]
	if !ok {
		return model.Team{}, repository.ErrNotFound
	}
	return it, nil
}
func (f *fakeTeamRepo) List(_ context.Context, p repository.Page) (repository.PageResult[model.Team], error) {
	f.lastPage = p
	res := repository.PageResult[model.Team]{}
	for _, v := range f.items {
		res.Items = append(res.Items, v)
	}
	res.Total = len(res.Items)
	return res, nil
}

var _ repository.TeamRepository = (*fakeTeamRepo)(nil)

func TestTeamService_CreateTeam_Validation(t *testing.T) {
	logger := zerolog.New(io.Discard)
	svc := service.NewTeamService(newFakeTeamRepo(), logger)

	cases := []struct {
		name      string
		input     string
		wantErr   bool
		wantField string
	}{
		{"empty", "", true, "name"},
		{"spaces", "   ", true, "name"},
		{"too short", "A", true, "name"},
		{"too long", string(make([]byte, 51)), true, "name"},
		{"ok", "Lakers", false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateTeam(context.Background(), tc.input)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr {
				if !serviceErrIsInvalid(err) {
					t.Fatalf("expected ErrInvalidInput, got %v", err)
				}
				fields := service.FieldErrors(err)
				found := false
				for _, f := range fields {
					if f.Field == tc.wantField {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("expected field error for %s, got %+v", tc.wantField, fields)
				}
			}
		})
	}
}

func TestTeamService_CreateTeam_DuplicatePropagates(t *testing.T) {
	logger := zerolog.New(io.Discard)
	repo := newFakeTeamRepo()
	repo.createErr = repository.ErrAlreadyExists
	svc := service.NewTeamService(repo, logger)
	_, err := svc.CreateTeam(context.Background(), "Lakers")
	if err == nil || err != repository.ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestTeamService_GetTeam_InvalidID(t *testing.T) {
	logger := zerolog.New(io.Discard)
	svc := service.NewTeamService(newFakeTeamRepo(), logger)
	_, err := svc.GetTeam(context.Background(), 0)
	if err == nil || !serviceErrIsInvalid(err) {
		t.Fatalf("expected invalid input error, got %v", err)
	}
}

func TestTeamService_ListTeams_PaginationNormalization(t *testing.T) {
	logger := zerolog.New(io.Discard)
	repo := newFakeTeamRepo()
	// seed a couple of items so result isn't empty
	_, _ = repo.Create(context.Background(), model.Team{Name: "A"})
	_, _ = repo.Create(context.Background(), model.Team{Name: "B"})
	svc := service.NewTeamService(repo, logger)
	_, err := svc.ListTeams(context.Background(), repository.Page{Limit: -5, Offset: -10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.lastPage.Limit != 50 { // defaultLimit from service package
		t.Fatalf("expected normalized limit=50 got %d", repo.lastPage.Limit)
	}
	if repo.lastPage.Offset != 0 {
		t.Fatalf("expected normalized offset=0 got %d", repo.lastPage.Offset)
	}
}

func serviceErrIsInvalid(err error) bool {
	return err != nil && (err.Error() == service.ErrInvalidInput.Error())
}
