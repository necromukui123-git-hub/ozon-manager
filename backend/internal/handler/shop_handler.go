package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/middleware"
	"ozon-manager/internal/service"
)

type ShopHandler struct {
	shopService *service.ShopService
}

func NewShopHandler(shopService *service.ShopService) *ShopHandler {
	return &ShopHandler{shopService: shopService}
}

// GetShops 获取店铺列表（根据角色返回不同店铺）
// GET /api/v1/shops
func (h *ShopHandler) GetShops(c *gin.Context) {
	claims := middleware.GetCurrentUser(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, dto.Response{
			Code:    401,
			Message: "未认证",
		})
		return
	}

	shops, err := h.shopService.GetAccessibleShopsByRole(claims.UserID, claims.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "获取店铺列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    shops,
	})
}

// GetShop 获取店铺详情
// GET /api/v1/shops/:id
func (h *ShopHandler) GetShop(c *gin.Context) {
	shopID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的店铺ID",
		})
		return
	}

	// 检查访问权限（根据角色）
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, uint(shopID), claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	shop, err := h.shopService.GetShopByID(uint(shopID))
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
		Data: dto.ShopInfo{
			ID:       shop.ID,
			Name:     shop.Name,
			IsActive: shop.IsActive,
		},
	})
}

// CreateShop 创建店铺
// POST /api/v1/shops
func (h *ShopHandler) CreateShop(c *gin.Context) {
	var req dto.CreateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	shop, err := h.shopService.CreateShop(&req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrClientIDExists {
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
		Message: "店铺创建成功",
		Data:    shop,
	})
}

// UpdateShop 更新店铺
// PUT /api/v1/shops/:id
func (h *ShopHandler) UpdateShop(c *gin.Context) {
	shopID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的店铺ID",
		})
		return
	}

	var req dto.UpdateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	if err := h.shopService.UpdateShop(uint(shopID), &req); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrShopNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrClientIDExists {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, dto.Response{
			Code:    statusCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "店铺更新成功",
	})
}

// DeleteShop 删除店铺
// DELETE /api/v1/shops/:id
func (h *ShopHandler) DeleteShop(c *gin.Context) {
	shopID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的店铺ID",
		})
		return
	}

	if err := h.shopService.DeleteShop(uint(shopID)); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrShopNotFound {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, dto.Response{
			Code:    statusCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "店铺删除成功",
	})
}

// ========== 店铺管理员功能 ==========

// GetMyShops 获取自己的店铺列表
// GET /api/v1/my/shops
func (h *ShopHandler) GetMyShops(c *gin.Context) {
	ownerID := middleware.GetCurrentUserID(c)
	shops, err := h.shopService.GetMyShops(ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "获取店铺列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    shops,
	})
}

// CreateMyShop 创建自己的店铺
// POST /api/v1/my/shops
func (h *ShopHandler) CreateMyShop(c *gin.Context) {
	var req dto.CreateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	ownerID := middleware.GetCurrentUserID(c)
	shop, err := h.shopService.CreateMyShop(&req, ownerID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrClientIDExists {
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
		Message: "店铺创建成功",
		Data:    shop,
	})
}

// UpdateMyShop 更新自己的店铺
// PUT /api/v1/my/shops/:id
func (h *ShopHandler) UpdateMyShop(c *gin.Context) {
	shopID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的店铺ID",
		})
		return
	}

	var req dto.UpdateShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	ownerID := middleware.GetCurrentUserID(c)
	if err := h.shopService.UpdateMyShop(uint(shopID), &req, ownerID); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrShopNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrShopNotBelongToYou {
			statusCode = http.StatusForbidden
		} else if err == service.ErrClientIDExists {
			statusCode = http.StatusConflict
		}
		c.JSON(statusCode, dto.Response{
			Code:    statusCode,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "店铺更新成功",
	})
}

// DeleteMyShop 删除自己的店铺
// DELETE /api/v1/my/shops/:id
func (h *ShopHandler) DeleteMyShop(c *gin.Context) {
	shopID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的店铺ID",
		})
		return
	}

	ownerID := middleware.GetCurrentUserID(c)
	if err := h.shopService.DeleteMyShop(uint(shopID), ownerID); err != nil {
		statusCode := http.StatusInternalServerError
		if err == service.ErrShopNotFound {
			statusCode = http.StatusNotFound
		} else if err == service.ErrShopNotBelongToYou {
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
		Message: "店铺删除成功",
	})
}

// ========== 系统管理员功能 ==========

// GetSystemOverview 获取系统概览
// GET /api/v1/admin/overview
func (h *ShopHandler) GetSystemOverview(c *gin.Context) {
	overview, err := h.shopService.GetSystemOverview()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "获取系统概览失败",
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    overview,
	})
}
