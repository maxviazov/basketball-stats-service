package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/service"
	"github.com/maxviazov/basketball-stats-service/pkg/response"
)

type GameHandler struct {
	svc service.GameService
}

func NewGameHandler(svc service.GameService) *GameHandler { return &GameHandler{svc: svc} }

func (h *GameHandler) Register(r *gin.RouterGroup) {
	g := r.Group("/games")
	{
		g.POST("", h.create)
		g.GET(":id", h.getByID)
		g.GET("", h.list)
	}
}

type createGameRequest struct {
	Season   string `json:"season"`
	Date     string `json:"date"` // RFC3339
	HomeTeam int64  `json:"home_team_id"`
	AwayTeam int64  `json:"away_team_id"`
	Status   string `json:"status"`
}

func (h *GameHandler) create(c *gin.Context) {
	var req createGameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, service.ErrInvalidInput)
		return
	}
	parsedDate, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		response.WriteError(c, service.ErrInvalidInput)
		return
	}
	game, err := h.svc.CreateGame(c.Request.Context(), req.Season, parsedDate, req.HomeTeam, req.AwayTeam, req.Status)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusCreated, game)
}

func (h *GameHandler) getByID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	game, err := h.svc.GetGame(c.Request.Context(), id)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusOK, game)
}

func (h *GameHandler) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	page := repository.Page{Limit: limit, Offset: offset}
	res, err := h.svc.ListGames(c.Request.Context(), page)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusOK, res)
}
