package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
)

type gameRepository struct{ pool *pgxpool.Pool }

func NewGameRepository(pool *pgxpool.Pool) repository.GameRepository {
	return &gameRepository{pool: pool}
}

func (r *gameRepository) Create(ctx context.Context, g model.Game) (model.Game, error) {
	if err := ensurePool(r.pool); err != nil {
		return model.Game{}, err
	}
	exec := getQ(ctx, r.pool)
	row := exec.QueryRow(ctx,
		`INSERT INTO games (season, date, home_team_id, away_team_id, status)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, season, date, home_team_id, away_team_id, status, created_at, updated_at`,
		g.Season, g.Date, g.HomeTeamID, g.AwayTeamID, g.Status,
	)
	var out model.Game
	if err := row.Scan(&out.ID, &out.Season, &out.Date, &out.HomeTeamID, &out.AwayTeamID, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return model.Game{}, repository.MapPgError(err)
	}
	return out, nil
}

func (r *gameRepository) GetByID(ctx context.Context, id int64) (model.Game, error) {
	if err := ensurePool(r.pool); err != nil {
		return model.Game{}, err
	}
	exec := getQ(ctx, r.pool)
	row := exec.QueryRow(ctx,
		`SELECT id, season, date, home_team_id, away_team_id, status, created_at, updated_at
		 FROM games WHERE id = $1`, id,
	)
	var out model.Game
	if err := row.Scan(&out.ID, &out.Season, &out.Date, &out.HomeTeamID, &out.AwayTeamID, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Game{}, repository.ErrNotFound
		}
		return model.Game{}, repository.MapPgError(err)
	}
	return out, nil
}

func (r *gameRepository) List(ctx context.Context, p repository.Page) (repository.PageResult[model.Game], error) {
	if err := ensurePool(r.pool); err != nil {
		return repository.PageResult[model.Game]{}, err
	}
	limit, offset := sanitizeLimitOffset(p.Limit, p.Offset)
	exec := getQ(ctx, r.pool)
	rows, err := exec.Query(ctx,
		`SELECT id, season, date, home_team_id, away_team_id, status, created_at, updated_at, COUNT(*) OVER() AS total
		 FROM games
		 ORDER BY date DESC, id DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return repository.PageResult[model.Game]{}, repository.MapPgError(err)
	}
	defer rows.Close()
	res := repository.PageResult[model.Game]{Items: make([]model.Game, 0, limit)}
	for rows.Next() {
		var it model.Game
		var total int
		if err := rows.Scan(&it.ID, &it.Season, &it.Date, &it.HomeTeamID, &it.AwayTeamID, &it.Status, &it.CreatedAt, &it.UpdatedAt, &total); err != nil {
			return repository.PageResult[model.Game]{}, repository.MapPgError(err)
		}
		res.Items = append(res.Items, it)
		res.Total = total
	}
	return res, nil
}

var _ repository.GameRepository = (*gameRepository)(nil)
