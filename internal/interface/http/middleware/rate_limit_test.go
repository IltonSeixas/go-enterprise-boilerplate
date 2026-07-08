package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
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

// fakeAuth injects claims for userID into the Gin context the same way
// RequireAuth does, so UserRateLimit (which must run after RequireAuth) can
// be tested in isolation without a real TokenService/UserRepository.
func fakeAuth(userID uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.AuthUserKey, port.AccessTokenClaims{UserID: userID, Role: entity.RoleUser})
		c.Next()
	}
}

func buildUserRateLimitedRouter(r rate.Limit, b int, authenticated bool, userID uuid.UUID) *gin.Engine {
	router := gin.New()
	if authenticated {
		router.Use(fakeAuth(userID))
	}
	router.Use(middleware.UserRateLimit(r, b))
	router.PUT("/resource", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func TestUserRateLimit_AllowsRequestsWithinBurst(t *testing.T) {
	router := buildUserRateLimitedRouter(rate.Limit(1), 2, true, uuid.New())

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPut, "/resource", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestUserRateLimit_RejectsRequestsExceedingBurst(t *testing.T) {
	router := buildUserRateLimitedRouter(rate.Limit(1), 2, true, uuid.New())

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPut, "/resource", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
	}

	req := httptest.NewRequest(http.MethodPut, "/resource", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
}

func TestUserRateLimit_TracksLimitsPerAuthenticatedUser(t *testing.T) {
	router := gin.New()
	limiter := middleware.UserRateLimit(rate.Limit(1), 1)

	userA, userB := uuid.New(), uuid.New()
	router.PUT("/resource", func(c *gin.Context) {
		userID := userA
		if c.GetHeader("X-Test-User") == "b" {
			userID = userB
		}
		fakeAuth(userID)(c)
		if c.IsAborted() {
			return
		}
		limiter(c)
		if c.IsAborted() {
			return
		}
		c.Status(http.StatusOK)
	})

	reqA := httptest.NewRequest(http.MethodPut, "/resource", nil)
	recA := httptest.NewRecorder()
	router.ServeHTTP(recA, reqA)

	reqB := httptest.NewRequest(http.MethodPut, "/resource", nil)
	reqB.Header.Set("X-Test-User", "b")
	recB := httptest.NewRecorder()
	router.ServeHTTP(recB, reqB)

	assert.Equal(t, http.StatusOK, recA.Code)
	assert.Equal(t, http.StatusOK, recB.Code, "a different authenticated user must have an independent limit")
}

func TestUserRateLimit_FallsThrough_WhenNoAuthenticatedClaims(t *testing.T) {
	router := buildUserRateLimitedRouter(rate.Limit(1), 1, false, uuid.Nil)

	req := httptest.NewRequest(http.MethodPut, "/resource", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "with no claims present, UserRateLimit must not block the request itself")
}
