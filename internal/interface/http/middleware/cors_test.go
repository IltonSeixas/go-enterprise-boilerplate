package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/middleware"
)

func buildCORSRouter(allowedOrigins []string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.CORS(allowedOrigins))
	r.GET("/resource", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestCORS_AllowsListedOrigin(t *testing.T) {
	r := buildCORSRouter([]string{"https://app.example.com"})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_RejectsUnlistedOrigin(t *testing.T) {
	r := buildCORSRouter([]string{"https://app.example.com"})

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_HandlesPreflightForAllowedOrigin(t *testing.T) {
	r := buildCORSRouter([]string{"https://app.example.com"})

	req := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "https://app.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
}
