// Package service holds business logic orchestration across repositories and handlers.
// Kept intentionally lean: only use-case coordination, validation and domain error shaping.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
)

// ErrInvalidInput is the marker error for aggregated validation failures (maps to HTTP 400).
// Field-level details are retrieved via FieldErrors(err).
var ErrInvalidInput = errors.New("invalid input")

// FieldError describes a single invalid field in a client request.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// invalidInputError aggregates multiple FieldError instances and unwraps to ErrInvalidInput.
type invalidInputError struct {
	fields []FieldError
}

func (e *invalidInputError) Error() string        { return ErrInvalidInput.Error() }
func (e *invalidInputError) Unwrap() error        { return ErrInvalidInput }
func (e *invalidInputError) Fields() []FieldError { return e.fields }

// newInvalidInput builds an aggregated validation error if any field errors are present.
func newInvalidInput(fe []FieldError) error {
	if len(fe) == 0 { // protective case
		return nil
	}
	return &invalidInputError{fields: fe}
}

// FieldErrors extracts field errors from an aggregated validation error.
func FieldErrors(err error) []FieldError {
	if err == nil {
		return nil
	}
	type feIface interface{ Fields() []FieldError }
	if v, ok := err.(feIface); ok && errors.Is(err, ErrInvalidInput) {
		return v.Fields()
	}
	return nil
}

// TeamService defines team-oriented use cases.
type TeamService interface {
	CreateTeam(ctx context.Context, name string) (model.Team, error)
	GetTeam(ctx context.Context, id int64) (model.Team, error)
	ListTeams(ctx context.Context, page repository.Page) (repository.PageResult[model.Team], error)
	GetTeamAggregatedStats(ctx context.Context, teamID int64, season *string) (model.TeamAggregatedStats, error)
}

// PlayerService defines player-oriented use cases.
type PlayerService interface {
	CreatePlayer(ctx context.Context, teamID int64, firstName, lastName, position string) (model.Player, error)
	GetPlayer(ctx context.Context, id int64) (model.Player, error)
	ListPlayersByTeam(ctx context.Context, teamID int64, page repository.Page) (repository.PageResult[model.Player], error)
	GetPlayerAggregatedStats(ctx context.Context, playerID int64, season *string) (model.PlayerAggregatedStats, error)
}

// GameService defines game-oriented use cases.
type GameService interface {
	CreateGame(ctx context.Context, season string, date time.Time, homeID, awayID int64, status string) (model.Game, error)
	GetGame(ctx context.Context, id int64) (model.Game, error)
	ListGames(ctx context.Context, page repository.Page) (repository.PageResult[model.Game], error)
}

// StatsService defines stat line use cases.
type StatsService interface {
	UpsertStatLine(ctx context.Context, line model.PlayerStatLine) (model.PlayerStatLine, error)
	ListStatsByGame(ctx context.Context, gameID int64) ([]model.PlayerStatLine, error)
}
