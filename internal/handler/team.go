package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/service"
	"github.com/maxviazov/basketball-stats-service/pkg/response"
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
	}
}

type createTeamRequest struct {
	Name string `json:"name"`
}

func (h *TeamHandler) create(c *gin.Context) {
	var req createTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.WriteError(c, service.ErrInvalidInput) // не расшифровываем внутренние детали парсинга
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
	id, _ := strconv.ParseInt(idStr, 10, 64)
	team, err := h.svc.GetTeam(c.Request.Context(), id)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteData(c, http.StatusOK, team)
}

func (h *TeamHandler) list(c *gin.Context) {
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
