package handler

import (
	"net/http"
	"strconv"

	"ozon-manager/internal/dto"
	"ozon-manager/internal/middleware"
	"ozon-manager/internal/service"

	"github.com/gin-gonic/gin"
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

// ChangePassword 用户修改自己的密码
// PUT /api/v1/auth/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	userID := middleware.GetCurrentUserID(c)
	if err := h.userService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUserNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrWrongPassword {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, dto.Response{
			Code:    statusCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "密码修改成功",
	})
}

// ========== 系统管理员功能 ==========

// GetShopAdmins 获取所有店铺管理员
// GET /api/v1/admin/shop-admins
func (h *UserHandler) GetShopAdmins(c *gin.Context) {
	shopAdmins, err := h.userService.GetAllShopAdmins()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "获取店铺管理员列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    shopAdmins,
	})
}

// GetShopAdmin 获取店铺管理员详情
// GET /api/v1/admin/shop-admins/:id
func (h *UserHandler) GetShopAdmin(c *gin.Context) {
	shopAdminID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	detail, err := h.userService.GetShopAdminDetail(uint(shopAdminID))
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
		Data:    detail,
	})
}

// CreateShopAdmin 创建店铺管理员
// POST /api/v1/admin/shop-admins
func (h *UserHandler) CreateShopAdmin(c *gin.Context) {
	var req dto.CreateShopAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	shopAdmin, err := h.userService.CreateShopAdmin(&req)
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
		Message: "店铺管理员创建成功",
		Data:    shopAdmin,
	})
}

// UpdateShopAdminStatus 更新店铺管理员状态
// PUT /api/v1/admin/shop-admins/:id/status
func (h *UserHandler) UpdateShopAdminStatus(c *gin.Context) {
	shopAdminID, err := strconv.ParseUint(c.Param("id"), 10, 32)
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

	if err := h.userService.UpdateShopAdminStatus(uint(shopAdminID), req.Status); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUserNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrCannotModifySuperAdmin || err == service.ErrNotShopAdmin {
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

// ResetShopAdminPassword 重置店铺管理员密码
// PUT /api/v1/admin/shop-admins/:id/password
func (h *UserHandler) ResetShopAdminPassword(c *gin.Context) {
	shopAdminID, err := strconv.ParseUint(c.Param("id"), 10, 32)
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

	if err := h.userService.ResetShopAdminPassword(uint(shopAdminID), req.NewPassword); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUserNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrCannotModifySuperAdmin || err == service.ErrNotShopAdmin {
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

// DeleteShopAdmin 删除店铺管理员
// DELETE /api/v1/admin/shop-admins/:id
func (h *UserHandler) DeleteShopAdmin(c *gin.Context) {
	shopAdminID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	if err := h.userService.DeleteShopAdmin(uint(shopAdminID)); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUserNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrCannotModifySuperAdmin || err == service.ErrNotShopAdmin {
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
		Message: "店铺管理员删除成功",
	})
}

// ========== 店铺管理员功能 ==========

// GetMyStaff 获取自己的员工列表
// GET /api/v1/my/staff
func (h *UserHandler) GetMyStaff(c *gin.Context) {
	ownerID := middleware.GetCurrentUserID(c)
	staff, err := h.userService.GetMyStaff(ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "获取员工列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    staff,
	})
}

// CreateStaff 创建员工
// POST /api/v1/my/staff
func (h *UserHandler) CreateStaff(c *gin.Context) {
	var req dto.CreateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	ownerID := middleware.GetCurrentUserID(c)
	staff, err := h.userService.CreateStaff(&req, ownerID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUsernameExists {
			statusCode = http.StatusConflict
		} else if err == service.ErrShopNotBelongToYou {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, dto.Response{
			Code:    statusCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.Response{
		Code:    201,
		Message: "员工创建成功",
		Data:    staff,
	})
}

// UpdateStaffStatus 更新员工状态
// PUT /api/v1/my/staff/:id/status
func (h *UserHandler) UpdateStaffStatus(c *gin.Context) {
	staffID, err := strconv.ParseUint(c.Param("id"), 10, 32)
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

	ownerID := middleware.GetCurrentUserID(c)
	if err := h.userService.UpdateStaffStatus(uint(staffID), req.Status, ownerID); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUserNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrStaffNotBelongToYou {
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

// ResetStaffPassword 重置员工密码
// PUT /api/v1/my/staff/:id/password
func (h *UserHandler) ResetStaffPassword(c *gin.Context) {
	staffID, err := strconv.ParseUint(c.Param("id"), 10, 32)
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

	ownerID := middleware.GetCurrentUserID(c)
	if err := h.userService.ResetStaffPassword(uint(staffID), req.NewPassword, ownerID); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUserNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrStaffNotBelongToYou {
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

// UpdateStaffShops 更新员工可访问的店铺
// PUT /api/v1/my/staff/:id/shops
func (h *UserHandler) UpdateStaffShops(c *gin.Context) {
	staffID, err := strconv.ParseUint(c.Param("id"), 10, 32)
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

	ownerID := middleware.GetCurrentUserID(c)
	if err := h.userService.UpdateStaffShops(uint(staffID), req.ShopIDs, ownerID); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUserNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrStaffNotBelongToYou || err == service.ErrShopNotBelongToYou {
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

// DeleteStaff 删除员工
// DELETE /api/v1/my/staff/:id
func (h *UserHandler) DeleteStaff(c *gin.Context) {
	staffID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的用户ID",
		})
		return
	}

	ownerID := middleware.GetCurrentUserID(c)
	if err := h.userService.DeleteStaff(uint(staffID), ownerID); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrUserNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrStaffNotBelongToYou {
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
		Message: "员工删除成功",
	})
}
