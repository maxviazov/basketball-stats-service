package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
)

type playerRepository struct{ pool *pgxpool.Pool }

func NewPlayerRepository(pool *pgxpool.Pool) repository.PlayerRepository {
	return &playerRepository{pool: pool}
}

func (r *playerRepository) Create(ctx context.Context, p model.Player) (model.Player, error) {
	if err := ensurePool(r.pool); err != nil {
		return model.Player{}, err
	}
	exec := getQ(ctx, r.pool)
	row := exec.QueryRow(ctx,
		`INSERT INTO players (team_id, first_name, last_name, position)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, team_id, first_name, last_name, position, created_at, updated_at`,
		p.TeamID, p.FirstName, p.LastName, p.Position,
	)
	var out model.Player
	if err := row.Scan(&out.ID, &out.TeamID, &out.FirstName, &out.LastName, &out.Position, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return model.Player{}, repository.MapPgError(err)
	}
	return out, nil
}

func (r *playerRepository) GetByID(ctx context.Context, id int64) (model.Player, error) {
	if err := ensurePool(r.pool); err != nil {
		return model.Player{}, err
	}
	exec := getQ(ctx, r.pool)
	row := exec.QueryRow(ctx,
		`SELECT id, team_id, first_name, last_name, position, created_at, updated_at
		 FROM players WHERE id = $1`, id,
	)
	var out model.Player
	if err := row.Scan(&out.ID, &out.TeamID, &out.FirstName, &out.LastName, &out.Position, &out.CreatedAt, &out.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Player{}, repository.ErrNotFound
		}
		return model.Player{}, repository.MapPgError(err)
	}
	return out, nil
}

func (r *playerRepository) ListByTeam(ctx context.Context, teamID int64, p repository.Page) (repository.PageResult[model.Player], error) {
	if err := ensurePool(r.pool); err != nil {
		return repository.PageResult[model.Player]{}, err
	}
	limit, offset := sanitizeLimitOffset(p.Limit, p.Offset)
	exec := getQ(ctx, r.pool)
	rows, err := exec.Query(ctx,
		`SELECT id, team_id, first_name, last_name, position, created_at, updated_at, COUNT(*) OVER() AS total
		 FROM players WHERE team_id = $1
		 ORDER BY id
		 LIMIT $2 OFFSET $3`,
		teamID, limit, offset,
	)
	if err != nil {
		return repository.PageResult[model.Player]{}, repository.MapPgError(err)
	}
	defer rows.Close()
	res := repository.PageResult[model.Player]{Items: make([]model.Player, 0, limit)}
	for rows.Next() {
		var it model.Player
		var total int
		if err := rows.Scan(&it.ID, &it.TeamID, &it.FirstName, &it.LastName, &it.Position, &it.CreatedAt, &it.UpdatedAt, &total); err != nil {
			return repository.PageResult[model.Player]{}, repository.MapPgError(err)
		}
		res.Items = append(res.Items, it)
		res.Total = total
	}
	return res, nil
}

// Exists performs a lightweight check to see if a player with the given ID exists.
func (r *playerRepository) Exists(ctx context.Context, id int64) (bool, error) {
	if err := ensurePool(r.pool); err != nil {
		return false, err
	}
	var exists bool
	exec := getQ(ctx, r.pool)
	err := exec.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM players WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return false, repository.MapPgError(err)
	}
	return exists, nil
}

// GetPlayerAggregatedStats calculates and returns a player's aggregated statistics.
// It can filter stats by a specific season. If season is nil, it calculates career stats.
// The query joins player_stats with games to filter by season and aggregates the results.
func (r *playerRepository) GetPlayerAggregatedStats(ctx context.Context, playerID int64, season *string) (model.PlayerAggregatedStats, error) {
	if err := ensurePool(r.pool); err != nil {
		return model.PlayerAggregatedStats{}, err
	}

	query := `
		SELECT
			COALESCE(COUNT(ps.id), 0) AS games_played,
			COALESCE(SUM(ps.points), 0) AS total_points,
			COALESCE(SUM(ps.rebounds), 0) AS total_rebounds,
			COALESCE(SUM(ps.assists), 0) AS total_assists,
			COALESCE(SUM(ps.steals), 0) AS total_steals,
			COALESCE(SUM(ps.blocks), 0) AS total_blocks,
			COALESCE(AVG(ps.points), 0) AS avg_points,
			COALESCE(AVG(ps.rebounds), 0) AS avg_rebounds,
			COALESCE(AVG(ps.assists), 0) AS avg_assists
		FROM
			player_stats ps
		INNER JOIN games g ON ps.game_id = g.id
		WHERE
			ps.player_id = $1 AND ($2::TEXT IS NULL OR g.season = $2)
	`

	exec := getQ(ctx, r.pool)
	row := exec.QueryRow(ctx, query, playerID, season)

	var stats model.PlayerAggregatedStats
	err := row.Scan(
		&stats.GamesPlayed,
		&stats.TotalPoints,
		&stats.TotalRebounds,
		&stats.TotalAssists,
		&stats.TotalSteals,
		&stats.TotalBlocks,
		&stats.AvgPoints,
		&stats.AvgRebounds,
		&stats.AvgAssists,
	)
	if err != nil {
		return model.PlayerAggregatedStats{}, repository.MapPgError(err)
	}

	return stats, nil
}

var _ repository.PlayerRepository = (*playerRepository)(nil)
