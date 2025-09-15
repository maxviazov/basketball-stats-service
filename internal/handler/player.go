package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/service"
	"github.com/maxviazov/basketball-stats-service/pkg/response"
)

type PlayerHandler struct {
	svc service.PlayerService
}

func NewPlayerHandler(svc service.PlayerService) *PlayerHandler { return &PlayerHandler{svc: svc} }

func (h *PlayerHandler) Register(r *gin.RouterGroup) {
	g := r.Group("/players")
	{
		g.POST("", h.create)
		g.GET(":id", h.getByID)
	}
	// Nested listing: /api/teams/:team_id/players
	r.Group("/teams").GET(":team_id/players", h.listByTeam)
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
	id, _ := strconv.ParseInt(idStr, 10, 64)
	player, err := h.svc.GetPlayer(c.Request.Context(), id)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusOK, player)
}

func (h *PlayerHandler) listByTeam(c *gin.Context) {
	idStr := c.Param("team_id")
	teamID, _ := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
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
