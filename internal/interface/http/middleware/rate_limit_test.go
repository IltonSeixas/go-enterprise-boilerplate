package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/middleware"
)

func buildRateLimitedRouter(r rate.Limit, b int) *gin.Engine {
	router := gin.New()
	router.Use(middleware.RateLimit(r, b))
	router.GET("/resource", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func TestRateLimit_AllowsRequestsWithinBurst(t *testing.T) {
	router := buildRateLimitedRouter(rate.Limit(1), 2)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestRateLimit_RejectsRequestsExceedingBurst(t *testing.T) {
	router := buildRateLimitedRouter(rate.Limit(1), 2)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/resource", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
}

func TestRateLimit_TracksLimitsPerClientIP(t *testing.T) {
	router := buildRateLimitedRouter(rate.Limit(1), 1)

	reqA := httptest.NewRequest(http.MethodGet, "/resource", nil)
	reqA.RemoteAddr = "10.0.0.1:1234"
	recA := httptest.NewRecorder()
	router.ServeHTTP(recA, reqA)

	reqB := httptest.NewRequest(http.MethodGet, "/resource", nil)
	reqB.RemoteAddr = "10.0.0.2:5678"
	recB := httptest.NewRecorder()
	router.ServeHTTP(recB, reqB)

	assert.Equal(t, http.StatusOK, recA.Code)
	assert.Equal(t, http.StatusOK, recB.Code)
}
