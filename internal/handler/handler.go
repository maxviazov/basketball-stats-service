package handler

import (
	"github.com/gin-gonic/gin"
)

// Register mounts all public routes on the given engine.
// I accept the minimal Pinger interface to keep routing independent from storage details.
func Register(r *gin.Engine, repo Pinger) {
	h := NewHealthHandler(repo)

	// Top-level health aliases for common probes (e.g., /ready, /live)
	r.GET("/live", h.Liveness)
	r.GET("/ready", h.Readiness)

	api := r.Group("/api")
	{
		health := api.Group("/health")
		{
			health.GET("/live", h.Liveness)
			health.GET("/ready", h.Readiness)
		}
	}
}
