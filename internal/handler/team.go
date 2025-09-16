package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/service"
	"github.com/maxviazov/basketball-stats-service/pkg/response"
	"github.com/rs/zerolog/log"
)

type TeamHandler struct {
	svc service.TeamService
}

func NewTeamHandler(svc service.TeamService) *TeamHandler { return &TeamHandler{svc: svc} }

func (h *TeamHandler) Register(r *gin.RouterGroup) {
	g := r.Group("/teams")
	{
		g.POST("", h.create)
		// Use a stable wildcard name (team_id) so nested routes (e.g. players) can reuse it without Gin conflicts.
		g.GET("/:team_id", h.getByID)
		g.GET("", h.list)
		g.GET("/:team_id/aggregates", h.getAggregatedStats)
		// Compatibility alias to support alternative path shape without changing contract
		g.GET("/:team_id/stats/aggregate", h.getAggregatedStats)
	}
}

type createTeamRequest struct {
	Name string `json:"name"`
}

func (h *TeamHandler) create(c *gin.Context) {
	var req createTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, service.ErrInvalidInput)
		return
	}
	team, err := h.svc.CreateTeam(c.Request.Context(), req.Name)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusCreated, team)
}

func (h *TeamHandler) getByID(c *gin.Context) {
	idStr := c.Param("team_id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteError(c, service.NewInvalidInputError([]service.FieldError{{Field: "team_id", Message: "must be a valid integer"}}))
		return
	}
	team, err := h.svc.GetTeam(c.Request.Context(), id)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusOK, team)
}

func (h *TeamHandler) list(c *gin.Context) {
	// Atoi errors are ignored intentionally, as 0 is a valid default for limit/offset, handled by the service layer.
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	page := repository.Page{Limit: limit, Offset: offset}
	res, err := h.svc.ListTeams(c.Request.Context(), page)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusOK, res)
}

// getAggregatedStats handles requests for a team's aggregated statistics.
func (h *TeamHandler) getAggregatedStats(c *gin.Context) {
	start := time.Now()
	idStr := c.Param("team_id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteError(c, service.NewInvalidInputError([]service.FieldError{{Field: "team_id", Message: "must be a valid integer"}}))
		return
	}

	seasonQuery := c.Query("season")
	careerQuery := c.Query("career")

	// Enforce mutual exclusion for query parameters
	if seasonQuery != "" && careerQuery != "" {
		response.WriteError(c, service.NewInvalidInputError([]service.FieldError{{
			Field:   "query",
			Message: "'season' and 'career' parameters are mutually exclusive",
		}}))
		return
	}

	var season *string
	if seasonQuery != "" {
		season = &seasonQuery
	} else if parseBoolQuery(careerQuery) {
		season = nil // Explicitly nil for career stats
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), serviceTimeout)
	defer cancel()

	stats, err := h.svc.GetTeamAggregatedStats(ctx, id, season)

	logger := log.With().
		Str("path", c.Request.URL.Path).
		Str("query", c.Request.URL.RawQuery).
		Int64("team_id", id).
		Dur("duration", time.Since(start)).
		Logger()

	if err != nil {
		status, _ := response.MapError(err)
		logger.Error().Err(err).Int("status", status).Msg("failed to get team aggregates")
		response.WriteError(c, err)
		return
	}

	logger.Info().Int("status", http.StatusOK).Msg("team aggregates retrieved")
	response.WriteData(c, http.StatusOK, stats)
}
