package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/dto"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/usecase"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/apperror"
)

type AuthHandler struct {
	register *usecase.RegisterUser
	login    *usecase.LoginUser
	refresh  *usecase.RefreshToken
}

func NewAuthHandler(
	register *usecase.RegisterUser,
	login *usecase.LoginUser,
	refresh *usecase.RefreshToken,
) *AuthHandler {
	return &AuthHandler{register: register, login: login, refresh: refresh}
}

type registerRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name"     binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.register.Execute(c.Request.Context(), dto.RegisterInput{
		Email: req.Email, Password: req.Password, Name: req.Name,
	})
	if err != nil {
		c.JSON(domainStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, out)
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.login.Execute(c.Request.Context(), dto.LoginInput{
		Email: req.Email, Password: req.Password,
	})
	if err != nil {
		c.JSON(domainStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	out, err := h.refresh.Execute(c.Request.Context(), dto.RefreshInput{RefreshToken: req.RefreshToken})
	if err != nil {
		c.JSON(domainStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func domainStatus(err error) int {
	switch err {
	case apperror.ErrInvalidEmail,
		apperror.ErrInvalidPassword,
		apperror.ErrInvalidName,
		apperror.ErrInvalidRole:
		return http.StatusUnprocessableEntity
	case apperror.ErrEmailAlreadyExists:
		return http.StatusConflict
	case apperror.ErrUserNotFound:
		return http.StatusNotFound
	case apperror.ErrInvalidCredentials,
		apperror.ErrTokenInvalid:
		return http.StatusUnauthorized
	case apperror.ErrAccountInactive,
		apperror.ErrInsufficientPerms:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
