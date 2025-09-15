package service

import (
	"context"
	"errors"
	"strings"
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
	// Normalize input strings
	seasonTrimmed := strings.TrimSpace(season)
	statusNorm := normalizeStatus(status)

	var ferrs []FieldError
	if homeID <= 0 {
		ferrs = append(ferrs, FieldError{Field: "home_team_id", Message: "must be > 0"})
	}
	if awayID <= 0 {
		ferrs = append(ferrs, FieldError{Field: "away_team_id", Message: "must be > 0"})
	}
	if homeID > 0 && awayID > 0 && homeID == awayID {
		ferrs = append(ferrs, FieldError{Field: "teams", Message: "home and away must differ"})
	}
	if date.IsZero() {
		ferrs = append(ferrs, FieldError{Field: "date", Message: "must be set"})
	}
	if seasonTrimmed == "" || !isValidSeason(seasonTrimmed) {
		ferrs = append(ferrs, FieldError{Field: "season", Message: "invalid format, expected YYYY-YY"})
	}
	if !isValidGameStatus(statusNorm) {
		ferrs = append(ferrs, FieldError{Field: "status", Message: "must be one of scheduled|in_progress|finished"})
	}

	// Early exit if basic structure is invalid – do not touch the database.
	if err := newInvalidInput(ferrs); err != nil {
		s.log.Debug().Interface("field_errors", ferrs).Msg("game validation failed (structure)")
		return model.Game{}, err
	}

	// Existence checks before attempting persistence.
	var existenceErrs []FieldError
	if _, err := s.teams.GetByID(ctx, homeID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			existenceErrs = append(existenceErrs, FieldError{Field: "home_team_id", Message: "team does not exist"})
		} else {
			return model.Game{}, err
		}
	}
	if _, err := s.teams.GetByID(ctx, awayID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			existenceErrs = append(existenceErrs, FieldError{Field: "away_team_id", Message: "team does not exist"})
		} else {
			return model.Game{}, err
		}
	}
	if err := newInvalidInput(existenceErrs); err != nil {
		s.log.Debug().Interface("field_errors", existenceErrs).Msg("game validation failed (existence)")
		return model.Game{}, err
	}

	// One INSERT – transaction is redundant, but we leave the generalization: maybe accompanying records will appear.
	var out model.Game
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		created, err := s.games.Create(ctx, model.Game{Season: seasonTrimmed, Date: date, HomeTeamID: homeID, AwayTeamID: awayID, Status: statusNorm})
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
		return model.Game{}, newInvalidInput([]FieldError{{Field: "id", Message: "must be > 0"}})
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
