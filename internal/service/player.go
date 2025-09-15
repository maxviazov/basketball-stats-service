package service

import (
	"context"
	"strings"
	"time"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/rs/zerolog"
)

type playerService struct {
	players repository.PlayerRepository
	teams   repository.TeamRepository
	log     zerolog.Logger
}

func NewPlayerService(players repository.PlayerRepository, teams repository.TeamRepository, logger zerolog.Logger) PlayerService {
	l := logger.With().Str("module", "service").Str("component", "player").Logger()
	return &playerService{players: players, teams: teams, log: l}
}

func (s *playerService) CreatePlayer(ctx context.Context, teamID int64, firstName, lastName, position string) (model.Player, error) {
	if teamID <= 0 {
		return model.Player{}, ErrInvalidInput
	}
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	position = strings.TrimSpace(position)
	if firstName == "" || lastName == "" || !isValidPosition(position) {
		return model.Player{}, ErrInvalidInput
	}
	// Optionally verify team exists (cheap read)
	if _, err := s.teams.GetByID(ctx, teamID); err != nil {
		return model.Player{}, err
	}
	start := time.Now()
	out, err := s.players.Create(ctx, model.Player{
		TeamID:    teamID,
		FirstName: firstName,
		LastName:  lastName,
		Position:  position,
	})
	if err != nil {
		s.log.Error().Err(err).Int64("team_id", teamID).Str("fn", firstName).Str("ln", lastName).Msg("create player failed")
		return model.Player{}, err
	}
	s.log.Info().Dur("took", time.Since(start)).Int64("player_id", out.ID).Msg("player created")
	return out, nil
}

func (s *playerService) GetPlayer(ctx context.Context, id int64) (model.Player, error) {
	if id <= 0 {
		return model.Player{}, ErrInvalidInput
	}
	return s.players.GetByID(ctx, id)
}

func (s *playerService) ListPlayersByTeam(ctx context.Context, teamID int64, page repository.Page) (repository.PageResult[model.Player], error) {
	if teamID <= 0 {
		return repository.PageResult[model.Player]{}, ErrInvalidInput
	}
	p := normalizePage(page)
	res, err := s.players.ListByTeam(ctx, teamID, p)
	if err != nil {
		s.log.Error().Err(err).Int64("team_id", teamID).Int("limit", p.Limit).Int("offset", p.Offset).Msg("list players failed")
		return repository.PageResult[model.Player]{}, err
	}
	return res, nil
}
