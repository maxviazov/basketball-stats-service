// Package service holds business logic orchestration across repositories and handlers.
// I keep this minimal until concrete use-cases arrive to avoid speculative abstractions.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
)

// ErrInvalidInput signals domain validation errors suitable for HTTP 400 mapping.
var ErrInvalidInput = errors.New("invalid input")

// TeamService defines team-oriented use cases.
type TeamService interface {
	CreateTeam(ctx context.Context, name string) (model.Team, error)
	GetTeam(ctx context.Context, id int64) (model.Team, error)
	ListTeams(ctx context.Context, page repository.Page) (repository.PageResult[model.Team], error)
}

// PlayerService defines player-oriented use cases.
type PlayerService interface {
	CreatePlayer(ctx context.Context, teamID int64, firstName, lastName, position string) (model.Player, error)
	GetPlayer(ctx context.Context, id int64) (model.Player, error)
	ListPlayersByTeam(ctx context.Context, teamID int64, page repository.Page) (repository.PageResult[model.Player], error)
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
