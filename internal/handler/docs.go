package handler

import (
	_ "embed"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// Minimal HTML that loads Swagger UI from a CDN and points to /openapi.yaml.
// This avoids bundling assets and keeps the binary small.
//
//go:embed swagger.html
var swaggerHTML string

// RegisterDocs mounts documentation endpoints at the root:
//   - GET /openapi.yaml: raw OpenAPI spec served from repository path api/openapi.yaml
//   - GET /docs: Swagger UI rendering of the spec
func RegisterDocs(r *gin.Engine) {
	r.GET("/openapi.yaml", func(c *gin.Context) {
		data, err := os.ReadFile("api/openapi.yaml")
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to read openapi spec: %v", err)
			return
		}
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", data)
	})
	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})
}
