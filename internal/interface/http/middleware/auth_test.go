package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/middleware"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func buildRouter(tokens port.TokenService, repo *testutil.StubUserRepo) *gin.Engine {
	r := gin.New()
	r.GET("/protected", middleware.RequireAuth(tokens, repo), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func newActiveUserInRepo(t *testing.T, repo *testutil.StubUserRepo, email string) (*entity.User, uuid.UUID) {
	t.Helper()
	e, err := valueobject.NewEmail(email)
	require.NoError(t, err)
	hash := valueobject.NewPasswordHashFromPHC("$argon2id$stub")
	u, err := entity.NewUser(e, hash, "Test User", entity.RoleUser)
	require.NoError(t, err)
	require.NoError(t, repo.Save(context.Background(), u))
	return u, u.ID().UUID()
}

func TestRequireAuth_MissingToken(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	tokens := testutil.NewStubTokenService()
	r := buildRouter(tokens, repo)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	tokens := testutil.NewStubTokenServiceRejectAll()
	r := buildRouter(tokens, repo)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAuth_ValidTokenActiveUser(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	user, userID := newActiveUserInRepo(t, repo, "active@example.com")
	_ = user

	claims := port.AccessTokenClaims{UserID: userID, Role: entity.RoleUser}
	tokens := testutil.NewStubTokenServiceWithClaims("valid-token", claims)
	r := buildRouter(tokens, repo)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAuth_ValidTokenInactiveUser(t *testing.T) {
	repo := testutil.NewStubUserRepo()

	e, err := valueobject.NewEmail("inactive@example.com")
	require.NoError(t, err)
	hash := valueobject.NewPasswordHashFromPHC("$argon2id$stub")
	u, err := entity.NewUser(e, hash, "Inactive User", entity.RoleUser)
	require.NoError(t, err)
	u.Deactivate()
	require.NoError(t, repo.Save(context.Background(), u))

	claims := port.AccessTokenClaims{UserID: u.ID().UUID(), Role: entity.RoleUser}
	tokens := testutil.NewStubTokenServiceWithClaims("valid-token-inactive", claims)
	r := buildRouter(tokens, repo)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token-inactive")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
