package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/application/port"
	"github.com/IltonSeixas/go-enterprise-boilerplate/internal/domain/repository"
)

const AuthUserKey = "authenticated_user"

func RequireAuth(tokens port.TokenService, users repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := tokens.ValidateAccessToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		user, err := users.FindByID(c.Request.Context(), claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		if !user.IsActive() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "account is inactive"})
			return
		}

		c.Set(AuthUserKey, claims)
		c.Next()
	}
}

func GetClaims(c *gin.Context) (port.AccessTokenClaims, bool) {
	v, ok := c.Get(AuthUserKey)
	if !ok {
		return port.AccessTokenClaims{}, false
	}
	claims, ok := v.(port.AccessTokenClaims)
	return claims, ok
}
