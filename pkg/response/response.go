// Package response centralizes HTTP response shapes and helpers.
// Handlers rely on it to keep controllers thin and uniform.
package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maxviazov/basketball-stats-service/internal/repository"
	"github.com/maxviazov/basketball-stats-service/internal/service"
)

// ErrorPayload is the canonical error envelope returned by the API.
type ErrorPayload struct {
	Error       string               `json:"error"`
	Message     string               `json:"message,omitempty"`
	FieldErrors []service.FieldError `json:"field_errors,omitempty"`
}

// MapError converts a domain / infrastructure error into an HTTP status and payload.
// Extend here as new domain error categories emerge.
func MapError(err error) (int, ErrorPayload) {
	if err == nil {
		return http.StatusOK, ErrorPayload{Error: "ok"}
	}

	if errors.Is(err, service.ErrInvalidInput) {
		return http.StatusBadRequest, ErrorPayload{
			Error:       "invalid_input",
			Message:     "one or more fields are invalid",
			FieldErrors: service.FieldErrors(err),
		}
	}

	switch {
	case errors.Is(err, repository.ErrNotFound):
		return http.StatusNotFound, ErrorPayload{Error: "not_found"}
	case errors.Is(err, repository.ErrAlreadyExists):
		return http.StatusConflict, ErrorPayload{Error: "already_exists"}
	case errors.Is(err, repository.ErrConflict):
		return http.StatusConflict, ErrorPayload{Error: "conflict"}
	default:
		return http.StatusInternalServerError, ErrorPayload{Error: "internal_error"}
	}
}

// WriteError writes an error response and aborts the context.
func WriteError(c *gin.Context, err error) {
	status, payload := MapError(err)
	c.AbortWithStatusJSON(status, payload)
}

// WriteData writes a successful JSON response.
func WriteData(c *gin.Context, status int, data any) {
	c.JSON(status, data)
}
