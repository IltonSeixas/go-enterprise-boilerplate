package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/handler"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

func performHealthRequest(h *handler.HealthHandler, path string) *httptest.ResponseRecorder {
	r := gin.New()
	r.GET("/health", h.Health)
	r.GET("/ready", h.Ready)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestHealthHandler_Health_AlwaysOK(t *testing.T) {
	h := handler.NewHealthHandler(testutil.NewStubPinger(errors.New("redis down")), nil)

	rec := performHealthRequest(h, "/health")

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHealthHandler_Ready_AllChecksPass(t *testing.T) {
	h := handler.NewHealthHandler(testutil.NewStubPinger(nil), testutil.NewStubPinger(nil))

	rec := performHealthRequest(h, "/ready")

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ready", body["status"])

	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ok", checks["redis"])
	assert.Equal(t, "ok", checks["postgres"])
}

func TestHealthHandler_Ready_RedisFailure_Returns503(t *testing.T) {
	h := handler.NewHealthHandler(testutil.NewStubPinger(errors.New("redis down")), testutil.NewStubPinger(nil))

	rec := performHealthRequest(h, "/ready")

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "not_ready", body["status"])

	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "error", checks["redis"])
	assert.Equal(t, "ok", checks["postgres"])
}

func TestHealthHandler_Ready_PostgresFailure_Returns503(t *testing.T) {
	h := handler.NewHealthHandler(testutil.NewStubPinger(nil), testutil.NewStubPinger(errors.New("postgres down")))

	rec := performHealthRequest(h, "/ready")

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "not_ready", body["status"])

	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ok", checks["redis"])
	assert.Equal(t, "error", checks["postgres"])
}

func TestHealthHandler_Ready_NoDatabaseConfigured_OmitsPostgresCheck(t *testing.T) {
	h := handler.NewHealthHandler(testutil.NewStubPinger(nil), (handler.Pinger)(nil))

	rec := performHealthRequest(h, "/ready")

	require.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	checks, ok := body["checks"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ok", checks["redis"])
	_, hasPostgres := checks["postgres"]
	assert.False(t, hasPostgres)
}
