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

var _ repository.PlayerRepository = (*playerRepository)(nil)
