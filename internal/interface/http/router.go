package http

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/handler"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/interface/http/middleware"
)

func NewRouter(
	authH *handler.AuthHandler,
	userH *handler.UserHandler,
	healthH *handler.HealthHandler,
	tokens port.TokenService,
	users repository.UserRepository,
	allowedOrigins []string,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(allowedOrigins))
	r.Use(middleware.RateLimit(rate.Limit(100), 20))

	r.GET("/health", healthH.Health)
	r.GET("/ready", healthH.Ready)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

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
			protected.GET("", userH.ListUsers)
			protected.GET("/:id", userH.GetUser)
			protected.PUT("/:id/role", userH.ChangeRole)
		}
	}

	return r
}
