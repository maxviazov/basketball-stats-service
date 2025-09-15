package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
)

type statsRepository struct{ pool *pgxpool.Pool }

func NewStatsRepository(pool *pgxpool.Pool) repository.StatsRepository {
	return &statsRepository{pool: pool}
}

func (r *statsRepository) UpsertStatLine(ctx context.Context, s model.PlayerStatLine) (model.PlayerStatLine, error) {
	if err := ensurePool(r.pool); err != nil {
		return model.PlayerStatLine{}, err
	}
	exec := getQ(ctx, r.pool)
	row := exec.QueryRow(ctx,
		`INSERT INTO player_stats (
			player_id, game_id, points, rebounds, assists, steals, blocks, fouls, turnovers, minutes_played
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (player_id, game_id)
		DO UPDATE SET
			points = EXCLUDED.points,
			rebounds = EXCLUDED.rebounds,
			assists = EXCLUDED.assists,
			steals = EXCLUDED.steals,
			blocks = EXCLUDED.blocks,
			fouls = EXCLUDED.fouls,
			turnovers = EXCLUDED.turnovers,
			minutes_played = EXCLUDED.minutes_played,
			updated_at = NOW()
		RETURNING id, player_id, game_id, points, rebounds, assists, steals, blocks, fouls, turnovers, minutes_played, created_at, updated_at`,
		s.PlayerID, s.GameID, s.Points, s.Rebounds, s.Assists, s.Steals, s.Blocks, s.Fouls, s.Turnovers, s.MinutesPlayed,
	)
	var out model.PlayerStatLine
	if err := row.Scan(&out.ID, &out.PlayerID, &out.GameID, &out.Points, &out.Rebounds, &out.Assists, &out.Steals, &out.Blocks, &out.Fouls, &out.Turnovers, &out.MinutesPlayed, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return model.PlayerStatLine{}, repository.MapPgError(err)
	}
	return out, nil
}

func (r *statsRepository) ListByGame(ctx context.Context, gameID int64) ([]model.PlayerStatLine, error) {
	if err := ensurePool(r.pool); err != nil {
		return nil, err
	}
	exec := getQ(ctx, r.pool)
	rows, err := exec.Query(ctx,
		`SELECT id, player_id, game_id, points, rebounds, assists, steals, blocks, fouls, turnovers, minutes_played, created_at, updated_at
		 FROM player_stats WHERE game_id = $1 ORDER BY id`, gameID,
	)
	if err != nil {
		return nil, repository.MapPgError(err)
	}
	defer rows.Close()
	res := make([]model.PlayerStatLine, 0, 8)
	for rows.Next() {
		var it model.PlayerStatLine
		if err := rows.Scan(&it.ID, &it.PlayerID, &it.GameID, &it.Points, &it.Rebounds, &it.Assists, &it.Steals, &it.Blocks, &it.Fouls, &it.Turnovers, &it.MinutesPlayed, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, repository.MapPgError(err)
		}
		res = append(res, it)
	}
	return res, nil
}

var _ repository.StatsRepository = (*statsRepository)(nil)
