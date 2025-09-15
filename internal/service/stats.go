package service

import (
	"context"
	"time"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/rs/zerolog"
)

type statsService struct {
	stats   repository.StatsRepository
	players repository.PlayerRepository
	games   repository.GameRepository
	tx      repository.TxManager
	log     zerolog.Logger
}

func NewStatsService(stats repository.StatsRepository, players repository.PlayerRepository, games repository.GameRepository, tx repository.TxManager, logger zerolog.Logger) StatsService {
	l := logger.With().Str("module", "service").Str("component", "stats").Logger()
	return &statsService{stats: stats, players: players, games: games, tx: tx, log: l}
}

func (s *statsService) UpsertStatLine(ctx context.Context, line model.PlayerStatLine) (model.PlayerStatLine, error) {
	if line.PlayerID <= 0 || line.GameID <= 0 {
		return model.PlayerStatLine{}, ErrInvalidInput
	}
	// Optional: ensure player and game exist (lets domain FK handle if omitted)
	if s.players != nil {
		if _, err := s.players.GetByID(ctx, line.PlayerID); err != nil {
			return model.PlayerStatLine{}, err
		}
	}
	if s.games != nil {
		if _, err := s.games.GetByID(ctx, line.GameID); err != nil {
			return model.PlayerStatLine{}, err
		}
	}
	start := time.Now()
	out, err := s.stats.UpsertStatLine(ctx, line)
	if err != nil {
		s.log.Error().Err(err).Int64("player_id", line.PlayerID).Int64("game_id", line.GameID).Msg("upsert stat line failed")
		return model.PlayerStatLine{}, err
	}
	s.log.Info().Dur("took", time.Since(start)).Int64("stat_id", out.ID).Msg("stat line upserted")
	return out, nil
}

func (s *statsService) ListStatsByGame(ctx context.Context, gameID int64) ([]model.PlayerStatLine, error) {
	if gameID <= 0 {
		return nil, ErrInvalidInput
	}
	res, err := s.stats.ListByGame(ctx, gameID)
	if err != nil {
		s.log.Error().Err(err).Int64("game_id", gameID).Msg("list stats by game failed")
		return nil, err
	}
	return res, nil
}
