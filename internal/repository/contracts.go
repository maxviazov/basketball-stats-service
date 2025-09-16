package repository

import (
	"context"

	"github.com/maxviazov/basketball-stats-service/internal/model"
)

// Pinger represents a minimal readiness probe capability.
// I use it to decouple health checks from storage implementation details.
type Pinger interface {
	Ping(ctx context.Context) error
}

// TxFunc is the unit of work executed within a transaction boundary.
// I pass context through so nested calls can honor cancellations and deadlines.
type TxFunc func(ctx context.Context) error

// TxManager abstracts transactional execution for repositories that support it.
// I prefer a single entry point to keep transaction boundaries explicit and testable.
type TxManager interface {
	WithinTx(ctx context.Context, fn TxFunc) error
}

// TeamRepository declares persistence operations for teams.
// I return domain models and surface domain errors from errors.go rather than PG codes.
type TeamRepository interface {
	Create(ctx context.Context, t model.Team) (model.Team, error)
	GetByID(ctx context.Context, id int64) (model.Team, error)
	List(ctx context.Context, p Page) (PageResult[model.Team], error)
	Exists(ctx context.Context, id int64) (bool, error)
	// GetTeamAggregatedStats calculates a team's performance stats, optionally filtered by season.
	// A nil season returns career stats across all seasons.
	GetTeamAggregatedStats(ctx context.Context, teamID int64, season *string) (model.TeamAggregatedStats, error)
}

// PlayerRepository declares persistence operations for players.
type PlayerRepository interface {
	Create(ctx context.Context, p model.Player) (model.Player, error)
	GetByID(ctx context.Context, id int64) (model.Player, error)
	ListByTeam(ctx context.Context, teamID int64, p Page) (PageResult[model.Player], error)
	Exists(ctx context.Context, id int64) (bool, error)
	// GetPlayerAggregatedStats calculates a player's stats, optionally filtered by season.
	// A nil season returns career stats.
	GetPlayerAggregatedStats(ctx context.Context, playerID int64, season *string) (model.PlayerAggregatedStats, error)
}

// GameRepository declares persistence operations for games.
type GameRepository interface {
	Create(ctx context.Context, g model.Game) (model.Game, error)
	GetByID(ctx context.Context, id int64) (model.Game, error)
	List(ctx context.Context, p Page) (PageResult[model.Game], error)
}

// StatsRepository declares operations for player stat lines per game.
type StatsRepository interface {
	UpsertStatLine(ctx context.Context, s model.PlayerStatLine) (model.PlayerStatLine, error)
	ListByGame(ctx context.Context, gameID int64) ([]model.PlayerStatLine, error)
}
