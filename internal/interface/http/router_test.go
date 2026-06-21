package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	httpinterface "github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/handler"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func buildTestRouter() *gin.Engine {
	repo := testutil.NewStubUserRepo()
	hasher := testutil.NewStubHasher()
	tokens := testutil.NewStubTokenService()

	authH := handler.NewAuthHandler(
		usecase.NewRegisterUser(repo, hasher, tokens),
		usecase.NewLoginUser(repo, hasher, tokens),
		usecase.NewRefreshToken(repo, tokens),
	)
	userH := handler.NewUserHandler(
		usecase.NewGetUser(repo),
		usecase.NewListUsers(repo),
		usecase.NewUpdateProfile(repo),
		usecase.NewChangePassword(repo, hasher),
		usecase.NewChangeUserRole(repo),
	)

	return httpinterface.NewRouter(authH, userH, tokens, repo, []string{"https://app.example.com"})
}

func TestRouter_HealthEndpoint(t *testing.T) {
	r := buildTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_ReadyEndpoint(t *testing.T) {
	r := buildTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_MetricsEndpoint(t *testing.T) {
	r := buildTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRouter_ProtectedRoute_RequiresAuth(t *testing.T) {
	r := buildTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRouter_AuthRoute_ReachableWithoutToken(t *testing.T) {
	r := buildTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.NotEqual(t, http.StatusUnauthorized, rec.Code)
	require.NotEqual(t, http.StatusNotFound, rec.Code)
}

func TestRouter_SecurityHeadersApplied(t *testing.T) {
	r := buildTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.NotEmpty(t, rec.Header().Get("X-Content-Type-Options"))
	assert.NotEmpty(t, rec.Header().Get("X-Frame-Options"))
}
