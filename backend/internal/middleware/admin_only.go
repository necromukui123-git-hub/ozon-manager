package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
)

// AdminOnlyMiddleware 仅管理员可访问的中间件
func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetCurrentUser(c)
		if claims == nil {
			c.JSON(http.StatusUnauthorized, dto.Response{
				Code:    401,
				Message: "未认证",
			})
			c.Abort()
			return
		}

		if claims.Role != "admin" {
			c.JSON(http.StatusForbidden, dto.Response{
				Code:    403,
				Message: "权限不足，仅管理员可访问",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
