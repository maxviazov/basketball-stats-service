package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxviazov/basketball-stats-service/internal/handler"
	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/service"
)

// stubPingerNoop satisfies handler.Pinger (health endpoints not focus here).
type stubPingerNoop struct{}

func (s stubPingerNoop) Ping(ctx context.Context) error { return nil }

// fakeInvalid replicates aggregated validation error semantics.
type fakeInvalid struct{ fe []service.FieldError }

func (f *fakeInvalid) Error() string                { return service.ErrInvalidInput.Error() }
func (f *fakeInvalid) Unwrap() error                { return service.ErrInvalidInput }
func (f *fakeInvalid) Fields() []service.FieldError { return f.fe }

// stubTeamService lets us control each method outcome.
type stubTeamService struct {
	create struct {
		team model.Team
		err  error
	}
	get struct {
		team model.Team
		err  error
	}
	list struct {
		res repository.PageResult[model.Team]
		err error
	}
	stats struct { // Added for stats endpoint
		res model.TeamAggregatedStats
		err error
	}
}

func (s *stubTeamService) CreateTeam(ctx context.Context, name string) (model.Team, error) {
	return s.create.team, s.create.err
}
func (s *stubTeamService) GetTeam(ctx context.Context, id int64) (model.Team, error) {
	return s.get.team, s.get.err
}
func (s *stubTeamService) ListTeams(ctx context.Context, p repository.Page) (repository.PageResult[model.Team], error) {
	return s.list.res, s.list.err
}
func (s *stubTeamService) GetTeamAggregatedStats(ctx context.Context, teamID int64, season *string) (model.TeamAggregatedStats, error) {
	return s.stats.res, s.stats.err // Dummy implementation
}

func newRouter(ts service.TeamService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.Register(r, stubPingerNoop{}, ts, nil, nil, nil)
	return r
}

func TestTeamHandler_Create_OK(t *testing.T) {
	stub := &stubTeamService{}
	stub.create.team = model.Team{ID: 1, Name: "Lakers"}
	r := newRouter(stub)
	body, _ := json.Marshal(map[string]string{"name": "Lakers"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/teams", bytes.NewReader(body)))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp model.Team
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.ID != 1 || resp.Name != "Lakers" {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestTeamHandler_Create_Invalid(t *testing.T) {
	stub := &stubTeamService{}
	stub.create.err = &fakeInvalid{fe: []service.FieldError{{Field: "name", Message: "must not be empty"}}}
	r := newRouter(stub)
	body, _ := json.Marshal(map[string]string{"name": ""})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/teams", bytes.NewReader(body)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("invalid_input")) || !bytes.Contains(w.Body.Bytes(), []byte("name")) {
		t.Fatalf("expected field error for name, body=%s", w.Body.String())
	}
}

func TestTeamHandler_Get_NotFound(t *testing.T) {
	stub := &stubTeamService{}
	stub.get.err = repository.ErrNotFound
	r := newRouter(stub)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/teams/42", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestTeamHandler_Get_OK(t *testing.T) {
	stub := &stubTeamService{}
	stub.get.team = model.Team{ID: 7, Name: "Heat"}
	r := newRouter(stub)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/teams/7", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("Heat")) {
		t.Fatalf("expected body to contain Heat: %s", w.Body.String())
	}
}
