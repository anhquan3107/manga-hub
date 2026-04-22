package auth

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mangahub/pkg/utils"
)

const (
	ContextUserIDKey   = "user_id"
	ContextUsernameKey = "username"
)

func Middleware(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			utils.Error(c, 401, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			utils.Error(c, 401, "invalid authorization header")
			c.Abort()
			return
		}

		claims, err := service.ParseToken(parts[1])
		if err != nil {
			utils.Error(c, 401, "invalid token")
			c.Abort()
			return
		}

		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextUsernameKey, claims.Username)
		c.Next()
	}
}
