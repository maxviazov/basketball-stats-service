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
	start := time.Now()
	rawFirst, rawLast, rawPos := firstName, lastName, position

	// Normalize early so validation and persistence see canonical values.
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	position = normalizePosition(position)

	var ferrs []FieldError
	if teamID <= 0 {
		ferrs = append(ferrs, FieldError{Field: "team_id", Message: "must be > 0"})
	}
	if firstName == "" {
		ferrs = append(ferrs, FieldError{Field: "first_name", Message: "must not be empty"})
	} else if ln := len([]rune(firstName)); ln > 50 {
		ferrs = append(ferrs, FieldError{Field: "first_name", Message: "length must be <= 50"})
	}
	if lastName == "" {
		ferrs = append(ferrs, FieldError{Field: "last_name", Message: "must not be empty"})
	} else if ln := len([]rune(lastName)); ln > 50 {
		ferrs = append(ferrs, FieldError{Field: "last_name", Message: "length must be <= 50"})
	}
	if !isValidPosition(position) { // after normalizePosition
		ferrs = append(ferrs, FieldError{Field: "position", Message: "must be one of PG, SG, SF, PF, C"})
	}

	if err := newInvalidInput(ferrs); err != nil {
		s.log.Debug().Interface("field_errors", ferrs).Str("fn_raw", rawFirst).Str("ln_raw", rawLast).Str("pos_raw", rawPos).Msg("player validation failed")
		return model.Player{}, err
	}

	// Existence check improves client UX vs deferring to FK violation.
	if _, err := s.teams.GetByID(ctx, teamID); err != nil {
		if err == repository.ErrNotFound { // cheap direct compare; could use errors.Is if wrapped
			ferrs = append(ferrs, FieldError{Field: "team_id", Message: "team does not exist"})
			return model.Player{}, newInvalidInput(ferrs)
		}
		return model.Player{}, err
	}

	out, err := s.players.Create(ctx, model.Player{TeamID: teamID, FirstName: firstName, LastName: lastName, Position: position})
	if err != nil {
		s.log.Error().Err(err).Int64("team_id", teamID).Str("fn", firstName).Str("ln", lastName).Msg("create player failed")
		return model.Player{}, err
	}
	s.log.Info().Dur("took", time.Since(start)).Int64("player_id", out.ID).Msg("player created")
	return out, nil
}

func (s *playerService) GetPlayer(ctx context.Context, id int64) (model.Player, error) {
	if id <= 0 {
		return model.Player{}, newInvalidInput([]FieldError{{Field: "id", Message: "must be > 0"}})
	}
	return s.players.GetByID(ctx, id)
}

func (s *playerService) ListPlayersByTeam(ctx context.Context, teamID int64, page repository.Page) (repository.PageResult[model.Player], error) {
	if teamID <= 0 {
		return repository.PageResult[model.Player]{}, newInvalidInput([]FieldError{{Field: "team_id", Message: "must be > 0"}})
	}
	p := normalizePage(page)
	res, err := s.players.ListByTeam(ctx, teamID, p)
	if err != nil {
		s.log.Error().Err(err).Int64("team_id", teamID).Int("limit", p.Limit).Int("offset", p.Offset).Msg("list players failed")
		return repository.PageResult[model.Player]{}, err
	}
	return res, nil
}

// GetPlayerAggregatedStats retrieves and validates parameters for fetching player statistics.
// It ensures the player ID is valid and the season format is correct if provided.
func (s *playerService) GetPlayerAggregatedStats(ctx context.Context, playerID int64, season *string) (model.PlayerAggregatedStats, error) {
	var ferrs []FieldError
	if playerID <= 0 {
		ferrs = append(ferrs, FieldError{Field: "id", Message: "must be > 0"})
	}
	if season != nil && !isValidSeason(*season) {
		ferrs = append(ferrs, FieldError{Field: "season", Message: "must be in YYYY-YY format"})
	}
	if err := newInvalidInput(ferrs); err != nil {
		return model.PlayerAggregatedStats{}, err
	}

	stats, err := s.players.GetPlayerAggregatedStats(ctx, playerID, season)
	if err != nil {
		s.log.Error().Err(err).Int64("player_id", playerID).Msg("failed to get player aggregated stats")
		return model.PlayerAggregatedStats{}, err
	}

	return stats, nil
}
