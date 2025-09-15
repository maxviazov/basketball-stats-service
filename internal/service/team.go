package service

import (
	"context"
	"strings"
	"time"

	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/rs/zerolog"
)

type teamService struct {
	repo repository.TeamRepository
	log  zerolog.Logger
}

func NewTeamService(repo repository.TeamRepository, logger zerolog.Logger) TeamService {
	l := logger.With().Str("module", "service").Str("component", "team").Logger()
	return &teamService{repo: repo, log: l}
}

func (s *teamService) CreateTeam(ctx context.Context, name string) (model.Team, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Team{}, ErrInvalidInput
	}
	start := time.Now()
	out, err := s.repo.Create(ctx, model.Team{Name: name})
	if err != nil {
		s.log.Error().Err(err).Str("name", name).Msg("create team failed")
		return model.Team{}, err
	}
	s.log.Info().Dur("took", time.Since(start)).Int64("team_id", out.ID).Msg("team created")
	return out, nil
}

func (s *teamService) GetTeam(ctx context.Context, id int64) (model.Team, error) {
	if id <= 0 {
		return model.Team{}, ErrInvalidInput
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
