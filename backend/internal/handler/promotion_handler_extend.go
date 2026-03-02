package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/middleware"
)

// UnifiedEnroll 统一报名（自动判断官方/店铺）
// POST /api/v1/promotions/unified-enroll
func (h *PromotionHandler) UnifiedEnroll(c *gin.Context) {
	var req dto.UnifiedEnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	c.Set("shop_id", req.ShopID)

	resp, err := h.promotionService.UnifiedEnroll(claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "操作失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: resp.Message,
		Data:    resp,
	})
}

// UnifiedRemove 统一退出（自动判断官方/店铺）
// POST /api/v1/promotions/unified-remove
func (h *PromotionHandler) UnifiedRemove(c *gin.Context) {
	var req dto.UnifiedRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	c.Set("shop_id", req.ShopID)

	resp, err := h.promotionService.UnifiedRemove(claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "操作失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: resp.Message,
		Data:    resp,
	})
}

// UnifiedProcessLoss 统一亏损处理（自动判断官方/店铺）
// POST /api/v1/promotions/unified-process-loss
func (h *PromotionHandler) UnifiedProcessLoss(c *gin.Context) {
	var req dto.UnifiedProcessLossRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	c.Set("shop_id", req.ShopID)

	resp, err := h.promotionService.UnifiedProcessLoss(claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "操作失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: resp.Message,
		Data:    resp,
	})
}

// UnifiedRepricePromote 统一改价推广（自动判断官方/店铺）
// POST /api/v1/promotions/unified-reprice-promote
func (h *PromotionHandler) UnifiedRepricePromote(c *gin.Context) {
	var req dto.UnifiedRepricePromoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	c.Set("shop_id", req.ShopID)

	resp, err := h.promotionService.UnifiedRepricePromote(claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "操作失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: resp.Message,
		Data:    resp,
	})
}
