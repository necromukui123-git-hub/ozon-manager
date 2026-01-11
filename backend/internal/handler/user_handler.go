package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/middleware"
	"ozon-manager/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetUsers 获取用户列表（员工）
// GET /api/v1/users
func (h *UserHandler) GetUsers(c *gin.Context) {
	users, err := h.userService.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "获取用户列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    users,
	})
}

// CreateUser 创建用户
// POST /api/v1/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	createdBy := middleware.GetCurrentUserID(c)
	user, err := h.userService.CreateUser(&req, createdBy)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUsernameExists {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, dto.Response{
			Code:    statusCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.Response{
		Code:    201,
		Message: "用户创建成功",
		Data:    user,
	})
}

// UpdateUserStatus 更新用户状态
// PUT /api/v1/users/:id/status
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	var req dto.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	if err := h.userService.UpdateUserStatus(uint(userID), req.Status); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUserNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrCannotModifyAdmin {
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
		Message: "状态更新成功",
	})
}

// UpdateUserPassword 重置用户密码
// PUT /api/v1/users/:id/password
func (h *UserHandler) UpdateUserPassword(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	var req dto.UpdateUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	if err := h.userService.UpdateUserPassword(uint(userID), req.NewPassword); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUserNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrCannotModifyAdmin {
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
		Message: "密码重置成功",
	})
}

// UpdateUserShops 更新用户可访问的店铺
// PUT /api/v1/users/:id/shops
func (h *UserHandler) UpdateUserShops(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	var req dto.UpdateUserShopsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	if err := h.userService.UpdateUserShops(uint(userID), req.ShopIDs); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUserNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrCannotModifyAdmin {
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
		Message: "店铺权限更新成功",
	})
}

// GetUser 获取用户详情
// GET /api/v1/users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	user, err := h.userService.GetUserByID(uint(userID))
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
		Data:    user,
	})
}
