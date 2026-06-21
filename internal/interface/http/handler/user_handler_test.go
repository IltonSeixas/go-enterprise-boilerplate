package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/entity"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/valueobject"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/handler"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/middleware"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newUserWithRole(t *testing.T, email string, role entity.Role) *entity.User {
	t.Helper()
	e, err := valueobject.NewEmail(email)
	require.NoError(t, err)
	hash := valueobject.NewPasswordHashFromPHC("$argon2id$stub")
	u, err := entity.NewUser(e, hash, "Test User", role)
	require.NoError(t, err)
	return u
}

func newUserHandler(repo *testutil.StubUserRepo) *handler.UserHandler {
	return handler.NewUserHandler(
		usecase.NewGetUser(repo),
		usecase.NewListUsers(repo),
		usecase.NewUpdateProfile(repo),
		usecase.NewChangePassword(repo, testutil.NewStubHasher()),
		usecase.NewChangeUserRole(repo),
	)
}

func newAuthenticatedContext(method, path string, body []byte, claims port.AccessTokenClaims, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
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
	c.Params = params
	c.Set(middleware.AuthUserKey, claims)

	return c, w
}

func TestUserHandler_GetMe_ReturnsOwnProfile(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	user := newUserWithRole(t, "me@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), user))

	h := newUserHandler(repo)
	c, w := newAuthenticatedContext(http.MethodGet, "/v1/users/me", nil,
		port.AccessTokenClaims{UserID: user.ID().UUID(), Role: entity.RoleUser}, nil)

	h.GetMe(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Equal(t, user.ID().UUID().String(), out["ID"])
}

func TestUserHandler_GetMe_WithoutClaims_ReturnsUnauthorized(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	h := newUserHandler(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/users/me", nil)

	h.GetMe(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserHandler_GetUser_WithValidID_ReturnsProfile(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	owner := newUserWithRole(t, "owner@example.com", entity.RoleOwner)
	target := newUserWithRole(t, "target@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), owner))
	require.NoError(t, repo.Save(context.Background(), target))

	h := newUserHandler(repo)
	c, w := newAuthenticatedContext(http.MethodGet, "/v1/users/"+target.ID().UUID().String(), nil,
		port.AccessTokenClaims{UserID: owner.ID().UUID(), Role: entity.RoleOwner},
		gin.Params{{Key: "id", Value: target.ID().UUID().String()}})

	h.GetUser(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_GetUser_WithInvalidID_ReturnsBadRequest(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	owner := newUserWithRole(t, "owner@example.com", entity.RoleOwner)
	require.NoError(t, repo.Save(context.Background(), owner))

	h := newUserHandler(repo)
	c, w := newAuthenticatedContext(http.MethodGet, "/v1/users/not-a-uuid", nil,
		port.AccessTokenClaims{UserID: owner.ID().UUID(), Role: entity.RoleOwner},
		gin.Params{{Key: "id", Value: "not-a-uuid"}})

	h.GetUser(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_GetUser_WhenForbidden_ReturnsForbidden(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	requester := newUserWithRole(t, "requester@example.com", entity.RoleUser)
	target := newUserWithRole(t, "target@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), requester))
	require.NoError(t, repo.Save(context.Background(), target))

	h := newUserHandler(repo)
	c, w := newAuthenticatedContext(http.MethodGet, "/v1/users/"+target.ID().UUID().String(), nil,
		port.AccessTokenClaims{UserID: requester.ID().UUID(), Role: entity.RoleUser},
		gin.Params{{Key: "id", Value: target.ID().UUID().String()}})

	h.GetUser(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUserHandler_UpdateMe_WithValidBody_UpdatesName(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	user := newUserWithRole(t, "me@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), user))

	h := newUserHandler(repo)
	body, err := json.Marshal(map[string]string{"name": "New Name"})
	require.NoError(t, err)

	c, w := newAuthenticatedContext(http.MethodPut, "/v1/users/me", body,
		port.AccessTokenClaims{UserID: user.ID().UUID(), Role: entity.RoleUser}, nil)

	h.UpdateMe(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_UpdateMe_WithInvalidBody_ReturnsBadRequest(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	user := newUserWithRole(t, "me@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), user))

	h := newUserHandler(repo)
	c, w := newAuthenticatedContext(http.MethodPut, "/v1/users/me", []byte(`{}`),
		port.AccessTokenClaims{UserID: user.ID().UUID(), Role: entity.RoleUser}, nil)

	h.UpdateMe(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_ChangePassword_WithValidBody_ReturnsNoContent(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	user := newUserWithRole(t, "me@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), user))

	h := newUserHandler(repo)
	body, err := json.Marshal(map[string]string{
		"current_password": "old-password",
		"new_password":     "New-Password-123",
	})
	require.NoError(t, err)

	c, w := newAuthenticatedContext(http.MethodPut, "/v1/users/me/password", body,
		port.AccessTokenClaims{UserID: user.ID().UUID(), Role: entity.RoleUser}, nil)

	h.ChangePassword(c)
	c.Writer.WriteHeaderNow()

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestUserHandler_ChangePassword_WithInvalidBody_ReturnsBadRequest(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	user := newUserWithRole(t, "me@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), user))

	h := newUserHandler(repo)
	c, w := newAuthenticatedContext(http.MethodPut, "/v1/users/me/password", []byte(`{}`),
		port.AccessTokenClaims{UserID: user.ID().UUID(), Role: entity.RoleUser}, nil)

	h.ChangePassword(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_ChangeRole_ByOwner_UpdatesTargetRole(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	owner := newUserWithRole(t, "owner@example.com", entity.RoleOwner)
	target := newUserWithRole(t, "target@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), owner))
	require.NoError(t, repo.Save(context.Background(), target))

	h := newUserHandler(repo)
	body, err := json.Marshal(map[string]string{"role": string(entity.RoleAdmin)})
	require.NoError(t, err)

	c, w := newAuthenticatedContext(http.MethodPut, "/v1/users/"+target.ID().UUID().String()+"/role", body,
		port.AccessTokenClaims{UserID: owner.ID().UUID(), Role: entity.RoleOwner},
		gin.Params{{Key: "id", Value: target.ID().UUID().String()}})

	h.ChangeRole(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserHandler_ChangeRole_ByNonOwner_ReturnsForbidden(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	admin := newUserWithRole(t, "admin@example.com", entity.RoleAdmin)
	target := newUserWithRole(t, "target@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), admin))
	require.NoError(t, repo.Save(context.Background(), target))

	h := newUserHandler(repo)
	body, err := json.Marshal(map[string]string{"role": string(entity.RoleAdmin)})
	require.NoError(t, err)

	c, w := newAuthenticatedContext(http.MethodPut, "/v1/users/"+target.ID().UUID().String()+"/role", body,
		port.AccessTokenClaims{UserID: admin.ID().UUID(), Role: entity.RoleAdmin},
		gin.Params{{Key: "id", Value: target.ID().UUID().String()}})

	h.ChangeRole(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUserHandler_ChangeRole_WithInvalidID_ReturnsBadRequest(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	owner := newUserWithRole(t, "owner@example.com", entity.RoleOwner)
	require.NoError(t, repo.Save(context.Background(), owner))

	h := newUserHandler(repo)
	body, err := json.Marshal(map[string]string{"role": string(entity.RoleAdmin)})
	require.NoError(t, err)

	c, w := newAuthenticatedContext(http.MethodPut, "/v1/users/not-a-uuid/role", body,
		port.AccessTokenClaims{UserID: owner.ID().UUID(), Role: entity.RoleOwner},
		gin.Params{{Key: "id", Value: "not-a-uuid"}})

	h.ChangeRole(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_ChangeRole_WithInvalidBody_ReturnsBadRequest(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	owner := newUserWithRole(t, "owner@example.com", entity.RoleOwner)
	target := newUserWithRole(t, "target@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), owner))
	require.NoError(t, repo.Save(context.Background(), target))

	h := newUserHandler(repo)

	c, w := newAuthenticatedContext(http.MethodPut, "/v1/users/"+target.ID().UUID().String()+"/role", []byte(`{}`),
		port.AccessTokenClaims{UserID: owner.ID().UUID(), Role: entity.RoleOwner},
		gin.Params{{Key: "id", Value: target.ID().UUID().String()}})

	h.ChangeRole(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserHandler_ChangeRole_WithInvalidRole_ReturnsUnprocessableEntity(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	owner := newUserWithRole(t, "owner@example.com", entity.RoleOwner)
	target := newUserWithRole(t, "target@example.com", entity.RoleUser)
	require.NoError(t, repo.Save(context.Background(), owner))
	require.NoError(t, repo.Save(context.Background(), target))

	h := newUserHandler(repo)
	body, err := json.Marshal(map[string]string{"role": "superuser"})
	require.NoError(t, err)

	c, w := newAuthenticatedContext(http.MethodPut, "/v1/users/"+target.ID().UUID().String()+"/role", body,
		port.AccessTokenClaims{UserID: owner.ID().UUID(), Role: entity.RoleOwner},
		gin.Params{{Key: "id", Value: target.ID().UUID().String()}})

	h.ChangeRole(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUserHandler_ChangeRole_OnOwnRole_ReturnsForbidden(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	owner := newUserWithRole(t, "owner@example.com", entity.RoleOwner)
	require.NoError(t, repo.Save(context.Background(), owner))

	h := newUserHandler(repo)
	body, err := json.Marshal(map[string]string{"role": string(entity.RoleAdmin)})
	require.NoError(t, err)

	c, w := newAuthenticatedContext(http.MethodPut, "/v1/users/"+owner.ID().UUID().String()+"/role", body,
		port.AccessTokenClaims{UserID: owner.ID().UUID(), Role: entity.RoleOwner},
		gin.Params{{Key: "id", Value: owner.ID().UUID().String()}})

	h.ChangeRole(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUserHandler_ChangeRole_WithoutClaims_ReturnsUnauthorized(t *testing.T) {
	repo := testutil.NewStubUserRepo()
	h := newUserHandler(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPut, "/v1/users/"+uuid.New().String()+"/role", bytes.NewReader(nil))
	c.Params = gin.Params{{Key: "id", Value: uuid.New().String()}}

	h.ChangeRole(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
