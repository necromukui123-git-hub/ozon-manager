package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/middleware"
	"ozon-manager/internal/service"
)

type AutoPromotionHandler struct {
	autoPromotionService *service.AutoPromotionService
	shopService          *service.ShopService
}

func NewAutoPromotionHandler(autoPromotionService *service.AutoPromotionService, shopService *service.ShopService) *AutoPromotionHandler {
	return &AutoPromotionHandler{
		autoPromotionService: autoPromotionService,
		shopService:          shopService,
	}
}

func (h *AutoPromotionHandler) GetConfig(c *gin.Context) {
	shopID, err := strconv.ParseUint(c.Query("shop_id"), 10, 32)
	if err != nil || shopID == 0 {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "缺少shop_id参数"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, uint(shopID), claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "无权访问该店铺"})
		return
	}

	resp, err := h.autoPromotionService.GetConfig(uint(shopID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "获取自动加促销配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "success", Data: resp})
}

func (h *AutoPromotionHandler) UpdateConfig(c *gin.Context) {
	var req dto.AutoPromotionConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "请求参数错误"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "无权访问该店铺"})
		return
	}

	c.Set("shop_id", req.ShopID)

	resp, err := h.autoPromotionService.UpdateConfig(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "保存自动加促销配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "保存成功", Data: resp})
}

func (h *AutoPromotionHandler) StartRun(c *gin.Context) {
	var req dto.AutoPromotionRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "请求参数错误"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "无权访问该店铺"})
		return
	}

	c.Set("shop_id", req.ShopID)

	resp, err := h.autoPromotionService.StartManualRun(claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "触发自动加促销失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "已创建自动加促销任务", Data: resp})
}

func (h *AutoPromotionHandler) ListRuns(c *gin.Context) {
	var req dto.AutoPromotionRunListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "请求参数错误"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "无权访问该店铺"})
		return
	}

	resp, err := h.autoPromotionService.ListRuns(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "获取自动加促销历史失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "success", Data: resp})
}

func (h *AutoPromotionHandler) GetRunDetail(c *gin.Context) {
	runID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || runID == 0 {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "无效的任务ID"})
		return
	}

	shopID, err := strconv.ParseUint(c.Query("shop_id"), 10, 32)
	if err != nil || shopID == 0 {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "缺少shop_id参数"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, uint(shopID), claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "无权访问该店铺"})
		return
	}

	resp, err := h.autoPromotionService.GetRunDetail(uint(shopID), uint(runID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "获取自动加促销详情失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "success", Data: resp})
}
