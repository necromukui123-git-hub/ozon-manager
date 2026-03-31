package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/config"
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

	session, err := h.authService.Login(&req, buildAuthSessionMeta(c))
	if err != nil {
		statusCode := authErrorStatus(err)
		c.JSON(statusCode, dto.Response{
			Code:    statusCode,
			Message: err.Error(),
		})
		return
	}

	setRefreshTokenCookie(c, session.RefreshToken)

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "登录成功",
		Data:    session.Response,
	})
}

// Logout 用户登出
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, _ := c.Cookie(refreshCookieName())
	if err := h.authService.Logout(refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}

	clearRefreshTokenCookie(c)

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "登出成功",
	})
}

// Refresh 刷新 access token
// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshCookieName())
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.Response{
			Code:    401,
			Message: service.ErrRefreshTokenRequired.Error(),
		})
		return
	}

	session, err := h.authService.Refresh(refreshToken, buildAuthSessionMeta(c))
	if err != nil {
		statusCode := authErrorStatus(err)
		c.JSON(statusCode, dto.Response{
			Code:    statusCode,
			Message: err.Error(),
		})
		return
	}

	setRefreshTokenCookie(c, session.RefreshToken)

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "刷新成功",
		Data:    session.Response,
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

func buildAuthSessionMeta(c *gin.Context) service.AuthSessionMeta {
	return service.AuthSessionMeta{
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	}
}

func authErrorStatus(err error) int {
	switch err {
	case service.ErrInvalidCredentials, service.ErrRefreshTokenRequired, service.ErrInvalidRefreshToken:
		return http.StatusUnauthorized
	case service.ErrUserDisabled:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

func setRefreshTokenCookie(c *gin.Context, token string) {
	cfg := config.GetConfig()
	maxAge := int(refreshCookieTTL(cfg.JWT).Seconds())
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cfg.JWT.RefreshCookieName, token, maxAge, "/api/v1/auth", "", cfg.JWT.RefreshCookieSecure, true)
}

func clearRefreshTokenCookie(c *gin.Context) {
	cfg := config.GetConfig()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cfg.JWT.RefreshCookieName, "", -1, "/api/v1/auth", "", cfg.JWT.RefreshCookieSecure, true)
}

func refreshCookieName() string {
	cfg := config.GetConfig()
	if cfg.JWT.RefreshCookieName != "" {
		return cfg.JWT.RefreshCookieName
	}
	return "refresh_token"
}

func refreshCookieTTL(cfg config.JWTConfig) time.Duration {
	if cfg.RefreshExpireHours > 0 {
		return time.Duration(cfg.RefreshExpireHours) * time.Hour
	}
	if cfg.ExpireHours > 0 {
		return time.Duration(cfg.ExpireHours) * time.Hour
	}
	return 7 * 24 * time.Hour
}
