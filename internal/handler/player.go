package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/service"
	"github.com/maxviazov/basketball-stats-service/pkg/response"
	"github.com/rs/zerolog/log"
)

const serviceTimeout = 5 * time.Second

// parseBoolQuery is a helper to flexibly parse boolean-like query parameters.
func parseBoolQuery(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1"
}

type PlayerHandler struct {
	svc service.PlayerService
}

func NewPlayerHandler(svc service.PlayerService) *PlayerHandler { return &PlayerHandler{svc: svc} }

func (h *PlayerHandler) Register(r *gin.RouterGroup) {
	g := r.Group("/players")
	{
		g.POST("", h.create)
		g.GET("/:id", h.getByID)
		g.GET("/:id/aggregates", h.getAggregatedStats)
		// Compatibility alias: keep alternative path style without breaking current contract
		g.GET("/:id/stats/aggregate", h.getAggregatedStats)
	}
	// Nested listing: /api/v1/teams/:team_id/players
	r.Group("/teams").GET("/:team_id/players", h.listByTeam)
}

type createPlayerRequest struct {
	TeamID    int64  `json:"team_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Position  string `json:"position"`
}

func (h *PlayerHandler) create(c *gin.Context) {
	var req createPlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, service.ErrInvalidInput)
		return
	}
	player, err := h.svc.CreatePlayer(c.Request.Context(), req.TeamID, req.FirstName, req.LastName, req.Position)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusCreated, player)
}

func (h *PlayerHandler) getByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteError(c, service.NewInvalidInputError([]service.FieldError{{Field: "id", Message: "must be a valid integer"}}))
		return
	}
	player, err := h.svc.GetPlayer(c.Request.Context(), id)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusOK, player)
}

func (h *PlayerHandler) listByTeam(c *gin.Context) {
	idStr := c.Param("team_id")
	teamID, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
	if err != nil {
		response.WriteError(c, service.NewInvalidInputError([]service.FieldError{{Field: "team_id", Message: "must be a valid integer"}}))
		return
	}
	// Atoi errors are ignored intentionally, as 0 is a valid default for limit/offset, handled by the service layer.
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	page := repository.Page{Limit: limit, Offset: offset}
	res, err := h.svc.ListPlayersByTeam(c.Request.Context(), teamID, page)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusOK, res)
}

// getAggregatedStats handles requests for a player's aggregated statistics.
func (h *PlayerHandler) getAggregatedStats(c *gin.Context) {
	start := time.Now()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.WriteError(c, service.NewInvalidInputError([]service.FieldError{{Field: "id", Message: "must be a valid integer"}}))
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

	stats, err := h.svc.GetPlayerAggregatedStats(ctx, id, season)

	logger := log.With().
		Str("path", c.Request.URL.Path).
		Str("query", c.Request.URL.RawQuery).
		Int64("player_id", id).
		Dur("duration", time.Since(start)).
		Logger()

	if err != nil {
		status, _ := response.MapError(err)
		logger.Error().Err(err).Int("status", status).Msg("failed to get player aggregates")
		response.WriteError(c, err)
		return
	}

	logger.Info().Int("status", http.StatusOK).Msg("player aggregates retrieved")
	response.WriteData(c, http.StatusOK, stats)
}
