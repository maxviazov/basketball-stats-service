package service

import (
	"context"
	"errors"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/rs/zerolog"
)

const (
	maxFouls        = 6
	maxMinutesFloat = 48.0
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
	var ferrs []FieldError
	if line.PlayerID <= 0 {
		ferrs = append(ferrs, FieldError{Field: "player_id", Message: "must be > 0"})
	}
	if line.GameID <= 0 {
		ferrs = append(ferrs, FieldError{Field: "game_id", Message: "must be > 0"})
	}
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
	if line.Fouls < 0 || line.Fouls > maxFouls {
		ferrs = append(ferrs, FieldError{Field: "fouls", Message: "must be between 0 and 6"})
	}
	if line.Turnovers < 0 {
		ferrs = append(ferrs, FieldError{Field: "turnovers", Message: "must be >= 0"})
	}
	if line.MinutesPlayed < 0 || float64(line.MinutesPlayed) > maxMinutesFloat {
		ferrs = append(ferrs, FieldError{Field: "minutes_played", Message: "must be between 0 and 48.0"})
	}

	if err := NewInvalidInputError(ferrs); err != nil {
		return model.PlayerStatLine{}, err
	}

	var existenceErrs []FieldError
	if err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := s.players.GetByID(ctx, line.PlayerID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				existenceErrs = append(existenceErrs, FieldError{Field: "player_id", Message: "player does not exist"})
				return nil // continue checks
			}
			return err
		}
		if _, err := s.games.GetByID(ctx, line.GameID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				existenceErrs = append(existenceErrs, FieldError{Field: "game_id", Message: "game does not exist"})
				return nil // continue checks
			}
			return err
		}
		return nil
	}); err != nil {
		return model.PlayerStatLine{}, err
	}

	if err := NewInvalidInputError(existenceErrs); err != nil {
		return model.PlayerStatLine{}, err
	}

	return s.stats.UpsertStatLine(ctx, line)
}

func (s *statsService) ListStatsByGame(ctx context.Context, gameID int64) ([]model.PlayerStatLine, error) {
	if gameID <= 0 {
		return nil, NewInvalidInputError([]FieldError{{Field: "game_id", Message: "must be > 0"}})
	}
	return s.stats.ListByGame(ctx, gameID)
}
