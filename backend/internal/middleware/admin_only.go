package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/model"
)

// AdminOnlyMiddleware 仅管理员可访问的中间件（兼容旧代码，super_admin 和 shop_admin 都算管理员）
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

		if claims.Role != model.RoleSuperAdmin && claims.Role != model.RoleShopAdmin {
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

// RoleMiddleware 通用角色检查中间件
func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
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

		allowed := false
		for _, role := range allowedRoles {
			if claims.Role == role {
				allowed = true
				break
			}
		}

		if !allowed {
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
