package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/middleware"
)

type UserHandler struct {
	getUser        *usecase.GetUser
	updateProfile  *usecase.UpdateProfile
	changePassword *usecase.ChangePassword
}

func NewUserHandler(
	getUser *usecase.GetUser,
	updateProfile *usecase.UpdateProfile,
	changePassword *usecase.ChangePassword,
) *UserHandler {
	return &UserHandler{
		getUser:        getUser,
		updateProfile:  updateProfile,
		changePassword: changePassword,
	}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	out, err := h.getUser.Execute(c.Request.Context(), claims.UserID)
	if err != nil {
		c.JSON(domainStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if claims.UserID != targetID && !claims.Role.CanManageRoles() {
		c.JSON(http.StatusForbidden, gin.H{"error": apperror.ErrInsufficientPerms.Error()})
		return
	}

	out, err := h.getUser.Execute(c.Request.Context(), targetID)
	if err != nil {
		c.JSON(domainStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

type updateProfileRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.updateProfile.Execute(c.Request.Context(), claims.UserID, dto.UpdateProfileInput{Name: req.Name})
	if err != nil {
		c.JSON(domainStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password"     binding:"required"`
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.changePassword.Execute(c.Request.Context(), claims.UserID, dto.ChangePasswordInput{
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	})
	if err != nil {
		c.JSON(domainStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
