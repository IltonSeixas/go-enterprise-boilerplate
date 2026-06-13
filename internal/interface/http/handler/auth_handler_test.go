package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/handler"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

func newAuthHandler(repo *testutil.StubUserRepo, tokens port.TokenService) *handler.AuthHandler {
	hasher := testutil.NewStubHasher()
	return handler.NewAuthHandler(
		usecase.NewRegisterUser(repo, hasher, tokens),
		usecase.NewLoginUser(repo, hasher, tokens),
		usecase.NewRefreshToken(repo, tokens),
	)
}

func newUnauthenticatedContext(method, path string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	return c, w
}

func TestAuthHandler_Register_WithValidBody_ReturnsCreated(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	h := newAuthHandler(repo, testutil.NewStubTokenService())

	body, err := json.Marshal(map[string]string{
		"email":    "new-user@example.com",
		"password": "Valid-Password-123",
		"name":     "New User",
	})
	require.NoError(t, err)

	c, w := newUnauthenticatedContext(http.MethodPost, "/v1/auth/register", body)

	h.Register(c)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestAuthHandler_Register_WithInvalidBody_ReturnsBadRequest(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	h := newAuthHandler(repo, testutil.NewStubTokenService())

	c, w := newUnauthenticatedContext(http.MethodPost, "/v1/auth/register", []byte(`{}`))

	h.Register(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Register_WithWeakPassword_ReturnsUnprocessableEntity(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	h := newAuthHandler(repo, testutil.NewStubTokenService())

	body, err := json.Marshal(map[string]string{
		"email":    "new-user@example.com",
		"password": "short",
		"name":     "New User",
	})
	require.NoError(t, err)

	c, w := newUnauthenticatedContext(http.MethodPost, "/v1/auth/register", body)

	h.Register(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestAuthHandler_Login_WithValidCredentials_ReturnsOK(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	user := newUserWithRole(t, "login@example.com", entity.RoleUser)
	repo.SetFindByEmailResult(user, nil)
	require.NoError(t, repo.Save(context.Background(), user))

	h := newAuthHandler(repo, testutil.NewStubTokenService())

	body, err := json.Marshal(map[string]string{
		"email":    "login@example.com",
		"password": "Valid-Password-123",
	})
	require.NoError(t, err)

	c, w := newUnauthenticatedContext(http.MethodPost, "/v1/auth/login", body)

	h.Login(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Login_WithInvalidBody_ReturnsBadRequest(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	h := newAuthHandler(repo, testutil.NewStubTokenService())

	c, w := newUnauthenticatedContext(http.MethodPost, "/v1/auth/login", []byte(`{}`))

	h.Login(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Login_WithUnknownEmail_ReturnsUnauthorized(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	repo.SetFindByEmailResult(nil, apperror.ErrUserNotFound)
	h := newAuthHandler(repo, testutil.NewStubTokenService())

	body, err := json.Marshal(map[string]string{
		"email":    "unknown@example.com",
		"password": "Valid-Password-123",
	})
	require.NoError(t, err)

	c, w := newUnauthenticatedContext(http.MethodPost, "/v1/auth/login", body)

	h.Login(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Refresh_WithValidToken_ReturnsOK(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	user := newUserWithRole(t, "refresh@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), user))

	tokens := testutil.NewStubTokenServiceWithClaims("access-stub", port.AccessTokenClaims{
		UserID: user.ID().UUID(),
		Role:   entity.RoleUser,
	})
	h := newAuthHandler(repo, tokens)

	body, err := json.Marshal(map[string]string{"refresh_token": tokens.ValidRefreshToken})
	require.NoError(t, err)

	c, w := newUnauthenticatedContext(http.MethodPost, "/v1/auth/refresh", body)

	h.Refresh(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Refresh_WithInvalidBody_ReturnsBadRequest(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	h := newAuthHandler(repo, testutil.NewStubTokenService())

	c, w := newUnauthenticatedContext(http.MethodPost, "/v1/auth/refresh", []byte(`{}`))

	h.Refresh(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_Refresh_WithInvalidToken_ReturnsUnauthorized(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	h := newAuthHandler(repo, testutil.NewStubTokenServiceRejectAll())

	body, err := json.Marshal(map[string]string{"refresh_token": "invalid-token"})
	require.NoError(t, err)

	c, w := newUnauthenticatedContext(http.MethodPost, "/v1/auth/refresh", body)

	h.Refresh(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
