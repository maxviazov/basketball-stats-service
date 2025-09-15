package service

import (
	"context"
	"errors"
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
	start := time.Now()
	var ferrs []FieldError

	if line.PlayerID <= 0 {
		ferrs = append(ferrs, FieldError{Field: "player_id", Message: "must be > 0"})
	}
	if line.GameID <= 0 {
		ferrs = append(ferrs, FieldError{Field: "game_id", Message: "must be > 0"})
	}
	// Basic numeric ranges; negative values are never meaningful here.
	if line.Points < 0 {
		ferrs = append(ferrs, FieldError{Field: "points", Message: "must be >= 0"})
	}
	if line.Rebounds < 0 {
		ferrs = append(ferrs, FieldError{Field: "rebounds", Message: "must be >= 0"})
	}
	if line.Assists < 0 {
		ferrs = append(ferrs, FieldError{Field: "assists", Message: "must be >= 0"})
	}
	if line.Steals < 0 {
		ferrs = append(ferrs, FieldError{Field: "steals", Message: "must be >= 0"})
	}
	if line.Blocks < 0 {
		ferrs = append(ferrs, FieldError{Field: "blocks", Message: "must be >= 0"})
	}
	if line.Fouls < 0 {
		ferrs = append(ferrs, FieldError{Field: "fouls", Message: "must be >= 0"})
	}
	if line.Turnovers < 0 {
		ferrs = append(ferrs, FieldError{Field: "turnovers", Message: "must be >= 0"})
	}
	if line.MinutesPlayed < 0 || line.MinutesPlayed > 60 {
		ferrs = append(ferrs, FieldError{Field: "minutes_played", Message: "must be between 0 and 60"})
	}

	if err := newInvalidInput(ferrs); err != nil {
		s.log.Debug().Interface("field_errors", ferrs).Int64("player_id", line.PlayerID).Int64("game_id", line.GameID).Msg("stat line validation failed (structure)")
		return model.PlayerStatLine{}, err
	}

	// Existence checks yield field errors instead of pushing FK errors upward.
	var existenceErrs []FieldError
	if s.players != nil {
		if _, err := s.players.GetByID(ctx, line.PlayerID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				existenceErrs = append(existenceErrs, FieldError{Field: "player_id", Message: "player does not exist"})
			} else {
				return model.PlayerStatLine{}, err
			}
		}
	}
	if s.games != nil {
		if _, err := s.games.GetByID(ctx, line.GameID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				existenceErrs = append(existenceErrs, FieldError{Field: "game_id", Message: "game does not exist"})
			} else {
				return model.PlayerStatLine{}, err
			}
		}
	}
	if err := newInvalidInput(existenceErrs); err != nil {
		s.log.Debug().Interface("field_errors", existenceErrs).Int64("player_id", line.PlayerID).Int64("game_id", line.GameID).Msg("stat line validation failed (existence)")
		return model.PlayerStatLine{}, err
	}

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
		return nil, newInvalidInput([]FieldError{{Field: "game_id", Message: "must be > 0"}})
	}
	res, err := s.stats.ListByGame(ctx, gameID)
	if err != nil {
		s.log.Error().Err(err).Int64("game_id", gameID).Msg("list stats by game failed")
		return nil, err
	}
	return res, nil
}
