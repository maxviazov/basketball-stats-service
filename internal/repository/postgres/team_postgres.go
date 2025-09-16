package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
)

type teamRepository struct{ pool *pgxpool.Pool }

func NewTeamRepository(pool *pgxpool.Pool) repository.TeamRepository {
	return &teamRepository{pool: pool}
}

func (r *teamRepository) Create(ctx context.Context, t model.Team) (model.Team, error) {
	if err := ensurePool(r.pool); err != nil {
		return model.Team{}, err
	}
	exec := getQ(ctx, r.pool)
	row := exec.QueryRow(ctx,
		`INSERT INTO teams (name) VALUES ($1)
		 RETURNING id, name, created_at, updated_at`,
		t.Name,
	)
	var out model.Team
	if err := row.Scan(&out.ID, &out.Name, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return model.Team{}, repository.MapPgError(err)
	}
	return out, nil
}

func (r *teamRepository) GetByID(ctx context.Context, id int64) (model.Team, error) {
	if err := ensurePool(r.pool); err != nil {
		return model.Team{}, err
	}
	exec := getQ(ctx, r.pool)
	row := exec.QueryRow(ctx,
		`SELECT id, name, created_at, updated_at FROM teams WHERE id = $1`, id,
	)
	var out model.Team
	if err := row.Scan(&out.ID, &out.Name, &out.CreatedAt, &out.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Team{}, repository.ErrNotFound
		}
		return model.Team{}, repository.MapPgError(err)
	}
	return out, nil
}

func (r *teamRepository) List(ctx context.Context, p repository.Page) (repository.PageResult[model.Team], error) {
	if err := ensurePool(r.pool); err != nil {
		return repository.PageResult[model.Team]{}, err
	}
	limit, offset := sanitizeLimitOffset(p.Limit, p.Offset)
	exec := getQ(ctx, r.pool)
	rows, err := exec.Query(ctx,
		`SELECT id, name, created_at, updated_at, COUNT(*) OVER() AS total
		 FROM teams
		 ORDER BY id
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return repository.PageResult[model.Team]{}, repository.MapPgError(err)
	}
	defer rows.Close()
	res := repository.PageResult[model.Team]{Items: make([]model.Team, 0, limit)}
	for rows.Next() {
		var t model.Team
		var total int
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt, &total); err != nil {
			return repository.PageResult[model.Team]{}, repository.MapPgError(err)
		}
		res.Items = append(res.Items, t)
		res.Total = total
	}
	return res, nil
}

// Exists performs a lightweight check to see if a team with the given ID exists.
func (r *teamRepository) Exists(ctx context.Context, id int64) (bool, error) {
	if err := ensurePool(r.pool); err != nil {
		return false, err
	}
	var exists bool
	exec := getQ(ctx, r.pool)
	err := exec.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM teams WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return false, repository.MapPgError(err)
	}
	return exists, nil
}

// GetTeamAggregatedStats calculates and returns a team's aggregated statistics.
// It can filter by season; a nil season returns career stats.
// This query is complex:
// 1. It first calculates the total points for each team in each finished game (game_team_scores CTE).
// 2. It then determines the winner and loser for each game by comparing home and away scores (game_results CTE).
// 3. Finally, it aggregates wins, losses, and points for the specified team, filtering by season if provided.
func (r *teamRepository) GetTeamAggregatedStats(ctx context.Context, teamID int64, season *string) (model.TeamAggregatedStats, error) {
	if err := ensurePool(r.pool); err != nil {
		return model.TeamAggregatedStats{}, err
	}

	query := `
		WITH game_team_scores AS (
			-- Calculate total points for each team in each game from player stats.
			SELECT
				g.id AS game_id,
				p.team_id,
				SUM(ps.points) AS points
			FROM player_stats ps
			JOIN games g ON ps.game_id = g.id
			JOIN players p ON ps.player_id = p.id
			WHERE g.status = 'finished'
			GROUP BY g.id, p.team_id
		),
		game_results AS (
			-- Determine winner, loser, and points for each game.
			SELECT
				g.id AS game_id,
				g.home_team_id,
				g.away_team_id,
				COALESCE(hts.points, 0) AS home_points,
				COALESCE(ats.points, 0) AS away_points,
				CASE
					WHEN COALESCE(hts.points, 0) > COALESCE(ats.points, 0) THEN g.home_team_id
					ELSE g.away_team_id
				END AS winner_id,
				CASE
					WHEN COALESCE(hts.points, 0) < COALESCE(ats.points, 0) THEN g.home_team_id
					ELSE g.away_team_id
				END AS loser_id
			FROM games g
			LEFT JOIN game_team_scores hts ON g.id = hts.game_id AND g.home_team_id = hts.team_id
			LEFT JOIN game_team_scores ats ON g.id = ats.game_id AND g.away_team_id = ats.team_id
			WHERE g.status = 'finished' AND ($2::TEXT IS NULL OR g.season = $2)
		)
		-- Aggregate stats for the specified team.
		SELECT
			COALESCE(SUM(CASE WHEN winner_id = $1 THEN 1 ELSE 0 END), 0) AS wins,
			COALESCE(SUM(CASE WHEN loser_id = $1 THEN 1 ELSE 0 END), 0) AS losses,
			COALESCE(SUM(CASE WHEN home_team_id = $1 THEN home_points ELSE away_points END), 0) AS total_points_scored,
			COALESCE(SUM(CASE WHEN home_team_id = $1 THEN away_points ELSE home_points END), 0) AS total_points_allowed,
			COALESCE(ROUND(AVG(CASE WHEN home_team_id = $1 THEN home_points ELSE away_points END), 2), 0) AS avg_points_scored,
			COALESCE(ROUND(AVG(CASE WHEN home_team_id = $1 THEN away_points ELSE home_points END), 2), 0) AS avg_points_allowed
		FROM game_results
		WHERE home_team_id = $1 OR away_team_id = $1
	`

	exec := getQ(ctx, r.pool)
	row := exec.QueryRow(ctx, query, teamID, season)

	var stats model.TeamAggregatedStats
	err := row.Scan(
		&stats.Wins,
		&stats.Losses,
		&stats.TotalPointsScored,
		&stats.TotalPointsAllowed,
		&stats.AvgPointsScored,
		&stats.AvgPointsAllowed,
	)
	if err != nil {
		// pgx.ErrNoRows is not expected here since aggregates with COALESCE should always return a row.
		// However, we map the error just in case.
		return model.TeamAggregatedStats{}, repository.MapPgError(err)
	}

	return stats, nil
}

var _ repository.TeamRepository = (*teamRepository)(nil)
