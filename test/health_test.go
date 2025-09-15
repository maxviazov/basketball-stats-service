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

type stubPinger struct{ err error }

func (s stubPinger) Ping(ctx context.Context) error { return s.err }

func TestReadiness_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.Register(r, stubPinger{err: nil})

	req := httptest.NewRequest(http.MethodGet, "/api/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestReadiness_Unavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.Register(r, stubPinger{err: errors.New("db down")})

	req := httptest.NewRequest(http.MethodGet, "/api/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestLivenessRoot_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.Register(r, stubPinger{err: nil})

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestReadinessRoot_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.Register(r, stubPinger{err: nil})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestReadinessRoot_Unavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler.Register(r, stubPinger{err: errors.New("db down")})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d, body=%s", w.Code, w.Body.String())
	}
}
