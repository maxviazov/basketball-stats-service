package service

import (
	"context"
	"time"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/rs/zerolog"
)

type gameService struct {
	games repository.GameRepository
	teams repository.TeamRepository
	tx    repository.TxManager
	log   zerolog.Logger
}

func NewGameService(games repository.GameRepository, teams repository.TeamRepository, tx repository.TxManager, logger zerolog.Logger) GameService {
	l := logger.With().Str("module", "service").Str("component", "game").Logger()
	return &gameService{games: games, teams: teams, tx: tx, log: l}
}

func (s *gameService) CreateGame(ctx context.Context, season string, date time.Time, homeID, awayID int64, status string) (model.Game, error) {
	if homeID <= 0 || awayID <= 0 || homeID == awayID {
		return model.Game{}, ErrInvalidInput
	}
	if date.IsZero() {
		return model.Game{}, ErrInvalidInput
	}
	if !isValidGameStatus(status) {
		return model.Game{}, ErrInvalidInput
	}

	var out model.Game
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// Validate both teams exist
		if _, err := s.teams.GetByID(ctx, homeID); err != nil {
			return err
		}
		if _, err := s.teams.GetByID(ctx, awayID); err != nil {
			return err
		}
		g := model.Game{Season: season, Date: date, HomeTeamID: homeID, AwayTeamID: awayID, Status: status}
		created, err := s.games.Create(ctx, g)
		if err != nil {
			return err
		}
		out = created
		return nil
	})
	if err != nil {
		s.log.Error().Err(err).Int64("home_id", homeID).Int64("away_id", awayID).Msg("create game failed")
		return model.Game{}, err
	}
	return out, nil
}

func (s *gameService) GetGame(ctx context.Context, id int64) (model.Game, error) {
	if id <= 0 {
		return model.Game{}, ErrInvalidInput
	}
	return s.games.GetByID(ctx, id)
}

func (s *gameService) ListGames(ctx context.Context, page repository.Page) (repository.PageResult[model.Game], error) {
	p := normalizePage(page)
	res, err := s.games.List(ctx, p)
	if err != nil {
		s.log.Error().Err(err).Int("limit", p.Limit).Int("offset", p.Offset).Msg("list games failed")
		return repository.PageResult[model.Game]{}, err
	}
	return res, nil
}
