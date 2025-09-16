package service

import (
	"context"
	"strings"
	"time"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/rs/zerolog"
)

// teamService holds team use-case logic: validation + orchestration, no transport / SQL details.
type teamService struct {
	repo repository.TeamRepository
	log  zerolog.Logger
}

func NewTeamService(repo repository.TeamRepository, logger zerolog.Logger) TeamService {
	l := logger.With().Str("module", "service").Str("component", "team").Logger()
	return &teamService{repo: repo, log: l}
}

func (s *teamService) CreateTeam(ctx context.Context, name string) (model.Team, error) {
	start := time.Now()
	original := name
	name = strings.TrimSpace(name)

	var ferrs []FieldError
	if name == "" {
		ferrs = append(ferrs, FieldError{Field: "name", Message: "must not be empty"})
	} else {
		if ln := len([]rune(name)); ln < 2 || ln > 50 {
			ferrs = append(ferrs, FieldError{Field: "name", Message: "length must be between 2 and 50"})
		}
	}
	if err := newInvalidInput(ferrs); err != nil {
		s.log.Debug().Str("name_raw", original).Interface("field_errors", ferrs).Msg("team validation failed")
		return model.Team{}, err
	}

	out, err := s.repo.Create(ctx, model.Team{Name: name})
	if err != nil {
		// Repository surfaces domain-level errors already, do not wrap.
		s.log.Error().Err(err).Str("name", name).Msg("create team failed")
		return model.Team{}, err
	}
	s.log.Info().Dur("took", time.Since(start)).Int64("team_id", out.ID).Msg("team created")
	return out, nil
}

func (s *teamService) GetTeam(ctx context.Context, id int64) (model.Team, error) {
	if id <= 0 {
		return model.Team{}, newInvalidInput([]FieldError{{Field: "id", Message: "must be > 0"}})
	}
	return s.repo.GetByID(ctx, id)
}

func (s *teamService) ListTeams(ctx context.Context, page repository.Page) (repository.PageResult[model.Team], error) {
	p := normalizePage(page)
	res, err := s.repo.List(ctx, p)
	if err != nil {
		s.log.Error().Err(err).Int("limit", p.Limit).Int("offset", p.Offset).Msg("list teams failed")
		return repository.PageResult[model.Team]{}, err
	}
	return res, nil
}

// GetTeamAggregatedStats retrieves and validates parameters for fetching team statistics.
// It ensures the team ID is valid and the season format is correct if provided.
func (s *teamService) GetTeamAggregatedStats(ctx context.Context, teamID int64, season *string) (model.TeamAggregatedStats, error) {
	var ferrs []FieldError
	if teamID <= 0 {
		ferrs = append(ferrs, FieldError{Field: "id", Message: "must be > 0"})
	}
	// A non-nil season string must conform to the expected format.
	if season != nil && !isValidSeason(*season) {
		ferrs = append(ferrs, FieldError{Field: "season", Message: "must be in YYYY-YY format"})
	}
	if err := newInvalidInput(ferrs); err != nil {
		return model.TeamAggregatedStats{}, err
	}

	stats, err := s.repo.GetTeamAggregatedStats(ctx, teamID, season)
	if err != nil {
		// Not expecting ErrNotFound here, but logging just in case.
		s.log.Error().Err(err).Int64("team_id", teamID).Msg("failed to get team aggregated stats")
		return model.TeamAggregatedStats{}, err
	}

	return stats, nil
}
