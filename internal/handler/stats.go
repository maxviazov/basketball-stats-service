package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/maxviazov/basketball-stats-service/internal/model"
	"github.com/maxviazov/basketball-stats-service/internal/service"
	"github.com/maxviazov/basketball-stats-service/pkg/response"
)

type StatsHandler struct {
	svc service.StatsService
}

func NewStatsHandler(svc service.StatsService) *StatsHandler { return &StatsHandler{svc: svc} }

func (h *StatsHandler) Register(r *gin.RouterGroup) {
	// Upsert endpoint
	r.Group("/stats").POST("", h.upsert)
	// Listing by game id: /api/games/:id/stats
	r.Group("/games").GET(":id/stats", h.listByGame)
}

type upsertStatRequest struct {
	PlayerID      int64   `json:"player_id"`
	GameID        int64   `json:"game_id"`
	Points        int     `json:"points"`
	Rebounds      int     `json:"rebounds"`
	Assists       int     `json:"assists"`
	Steals        int     `json:"steals"`
	Blocks        int     `json:"blocks"`
	Fouls         int     `json:"fouls"`
	Turnovers     int     `json:"turnovers"`
	MinutesPlayed float32 `json:"minutes_played"`
}

func (h *StatsHandler) upsert(c *gin.Context) {
	var req upsertStatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, service.ErrInvalidInput)
		return
	}
	line, err := h.svc.UpsertStatLine(c.Request.Context(), model.PlayerStatLine{
		PlayerID:      req.PlayerID,
		GameID:        req.GameID,
		Points:        req.Points,
		Rebounds:      req.Rebounds,
		Assists:       req.Assists,
		Steals:        req.Steals,
		Blocks:        req.Blocks,
		Fouls:         req.Fouls,
		Turnovers:     req.Turnovers,
		MinutesPlayed: req.MinutesPlayed,
	})
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusOK, line)
}

func (h *StatsHandler) listByGame(c *gin.Context) {
	gameID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	lines, err := h.svc.ListStatsByGame(c.Request.Context(), gameID)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusOK, lines)
}
