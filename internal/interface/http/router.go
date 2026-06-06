package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/handler"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/middleware"
)

func NewRouter(
	authH *handler.AuthHandler,
	userH *handler.UserHandler,
	tokens port.TokenService,
	users repository.UserRepository,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RateLimit(rate.Limit(100), 20))

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/ready", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ready"}) })

	v1 := r.Group("/v1")
	{
		auth := v1.Group("/auth")
		auth.Use(middleware.RateLimit(rate.Limit(10), 5))
		{
			auth.POST("/register", authH.Register)
			auth.POST("/login", authH.Login)
			auth.POST("/refresh", authH.Refresh)
		}

		protected := v1.Group("/users")
		protected.Use(middleware.RequireAuth(tokens, users))
		{
			protected.GET("/me", userH.GetMe)
			protected.PUT("/me", userH.UpdateMe)
			protected.PUT("/me/password", userH.ChangePassword)
			protected.GET("/:id", userH.GetUser)
		}
	}

	return r
}
