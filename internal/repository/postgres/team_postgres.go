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

var _ repository.TeamRepository = (*teamRepository)(nil)
