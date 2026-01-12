package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
)

// SuperAdminOnlyMiddleware 仅系统管理员可访问
func SuperAdminOnlyMiddleware() gin.HandlerFunc {
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

		if claims.Role != model.RoleSuperAdmin {
			c.JSON(http.StatusForbidden, dto.Response{
				Code:    403,
				Message: "权限不足，仅系统管理员可访问",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ShopAdminOnlyMiddleware 仅店铺管理员可访问
func ShopAdminOnlyMiddleware() gin.HandlerFunc {
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

		if claims.Role != model.RoleShopAdmin {
			c.JSON(http.StatusForbidden, dto.Response{
				Code:    403,
				Message: "权限不足，仅店铺管理员可访问",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ShopAdminOrStaffMiddleware 店铺管理员或员工可访问（业务操作）
func ShopAdminOrStaffMiddleware() gin.HandlerFunc {
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

		if claims.Role != model.RoleShopAdmin && claims.Role != model.RoleStaff {
			c.JSON(http.StatusForbidden, dto.Response{
				Code:    403,
				Message: "权限不足",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
