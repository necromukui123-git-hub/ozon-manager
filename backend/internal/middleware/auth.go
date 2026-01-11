package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/pkg/jwt"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	ContextUserKey      = "user"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, dto.Response{
				Code:    401,
				Message: "未提供认证令牌",
			})
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			c.JSON(http.StatusUnauthorized, dto.Response{
				Code:    401,
				Message: "认证令牌格式错误",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		claims, err := jwt.ParseToken(tokenString)
		if err != nil {
			message := "无效的认证令牌"
			if err == jwt.ErrExpiredToken {
				message = "认证令牌已过期"
			}
			c.JSON(http.StatusUnauthorized, dto.Response{
				Code:    401,
				Message: message,
			})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set(ContextUserKey, claims)
		c.Next()
	}
}

// GetCurrentUser 从上下文获取当前用户信息
func GetCurrentUser(c *gin.Context) *jwt.Claims {
	if claims, exists := c.Get(ContextUserKey); exists {
		if userClaims, ok := claims.(*jwt.Claims); ok {
			return userClaims
		}
	}
	return nil
}

// GetCurrentUserID 从上下文获取当前用户ID
func GetCurrentUserID(c *gin.Context) uint {
	if claims := GetCurrentUser(c); claims != nil {
		return claims.UserID
	}
	return 0
}
