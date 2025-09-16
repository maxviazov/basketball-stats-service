package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/maxviazov/basketball-stats-service/internal/service"
)

// Register mounts all public routes on the given engine.
// Accepts service layer dependencies for API endpoints.
func Register(r *gin.Engine, repo Pinger, teamSvc service.TeamService, playerSvc service.PlayerService, gameSvc service.GameService, statsSvc service.StatsService) {
	h := NewHealthHandler(repo)

	// Health probes
	r.GET("/live", h.Liveness)
	r.GET("/ready", h.Readiness)

	// Docs endpoints (root-level)
	RegisterDocs(r)

	api := r.Group(APIV1Prefix) // Versioning added via single source of truth
	{
		health := api.Group("/health")
		{
			health.GET("/live", h.Liveness)
			health.GET("/ready", h.Readiness)
		}
		NewTeamHandler(teamSvc).Register(api)
		NewPlayerHandler(playerSvc).Register(api)
		NewGameHandler(gameSvc).Register(api)
		NewStatsHandler(statsSvc).Register(api)
	}
}
