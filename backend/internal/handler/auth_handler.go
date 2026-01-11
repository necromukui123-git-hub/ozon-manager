package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/middleware"
	"ozon-manager/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login 用户登录
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	resp, err := h.authService.Login(&req)
	if err != nil {
		statusCode := http.StatusUnauthorized
		if err == service.ErrUserDisabled {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, dto.Response{
			Code:    statusCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "登录成功",
		Data:    resp,
	})
}

// Logout 用户登出
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT是无状态的，登出只需要前端删除token即可
	// 如果需要实现token黑名单，可以在这里添加逻辑
	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "登出成功",
	})
}

// GetCurrentUser 获取当前用户信息
// GET /api/v1/auth/me
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID := middleware.GetCurrentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, dto.Response{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	userInfo, err := h.authService.GetCurrentUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Response{
			Code:    404,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    userInfo,
	})
}
