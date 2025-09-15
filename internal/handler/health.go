package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Pinger is the minimal contract I need from a repository to check readiness.
// I keep it local to the handler package to avoid coupling and simplify tests.
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler exposes liveness and readiness endpoints.
// I keep it tiny and dependency-only to make it easy to test and wire.
type HealthHandler struct {
	repo Pinger
}

// NewHealthHandler wires a health handler with its only dependency: something that can Ping.
func NewHealthHandler(repo Pinger) *HealthHandler {
	return &HealthHandler{repo: repo}
}

// Liveness responds OK if the process is up; it doesn't check dependencies.
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// Readiness verifies critical dependencies, currently just the database.
func (h *HealthHandler) Readiness(c *gin.Context) {
	if err := h.repo.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unavailable",
			"error":  err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
