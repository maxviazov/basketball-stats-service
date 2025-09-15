package test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxviazov/basketball-stats-service/internal/handler"
)

// stubPinger implements handler.Pinger for health endpoints.
type stubPinger struct{ err error }

func (s stubPinger) Ping(ctx context.Context) error { return s.err }

func newEngine(p handler.Pinger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// pass nil services – we only exercise health routes here
	handler.Register(r, p, nil, nil, nil, nil)
	return r
}

func TestReadiness_OK(t *testing.T) {
	r := newEngine(stubPinger{err: nil})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/health/ready", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestReadiness_Unavailable(t *testing.T) {
	r := newEngine(stubPinger{err: errors.New("db down")})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/health/ready", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestLivenessRoot_OK(t *testing.T) {
	r := newEngine(stubPinger{err: nil})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/live", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestReadinessRoot_OK(t *testing.T) {
	r := newEngine(stubPinger{err: nil})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestReadinessRoot_Unavailable(t *testing.T) {
	r := newEngine(stubPinger{err: errors.New("db down")})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}
}

func TestHealth_NotFound(t *testing.T) {
	r := newEngine(stubPinger{err: nil})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/no-such", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestReadiness_MethodNotAllowed(t *testing.T) {
	r := newEngine(stubPinger{err: nil})
	w := httptest.NewRecorder()
	// Gin by default returns 404 for unknown method if route only registered for GET.
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/health/ready", nil))
	if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 404 or 405, got %d", w.Code)
	}
}
