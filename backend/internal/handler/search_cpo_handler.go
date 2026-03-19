package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/middleware"
	"ozon-manager/internal/service"
)

type SearchCPOHandler struct {
	searchCPOService *service.SearchCPOService
	shopService      *service.ShopService
}

func NewSearchCPOHandler(searchCPOService *service.SearchCPOService, shopService *service.ShopService) *SearchCPOHandler {
	return &SearchCPOHandler{
		searchCPOService: searchCPOService,
		shopService:      shopService,
	}
}

func (h *SearchCPOHandler) GetConfig(c *gin.Context) {
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

	resp, err := h.searchCPOService.GetConfig(uint(shopID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "获取 CPO 配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "success", Data: resp})
}

func (h *SearchCPOHandler) UpdateConfig(c *gin.Context) {
	var req dto.SearchCPOConfigRequest
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

	resp, err := h.searchCPOService.UpdateConfig(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "保存 CPO 配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "保存成功", Data: resp})
}

func (h *SearchCPOHandler) ListProducts(c *gin.Context) {
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

	resp, err := h.searchCPOService.ListProducts(uint(shopID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "获取 CPO 商品失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "success", Data: resp})
}

func (h *SearchCPOHandler) RefreshProducts(c *gin.Context) {
	var req dto.SearchCPORefreshRequest
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

	resp, err := h.searchCPOService.RefreshProducts(claims.UserID, req.ShopID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "刷新成功", Data: resp})
}

func (h *SearchCPOHandler) StartRun(c *gin.Context) {
	var req dto.SearchCPORunRequest
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

	resp, err := h.searchCPOService.StartRun(claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "创建 CPO 报名任务失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "已创建 CPO 报名任务", Data: resp})
}

func (h *SearchCPOHandler) ListRuns(c *gin.Context) {
	var req dto.SearchCPORunListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "请求参数错误"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "无权访问该店铺"})
		return
	}

	resp, err := h.searchCPOService.ListRuns(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "获取 CPO 报名历史失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "success", Data: resp})
}

func (h *SearchCPOHandler) GetRunDetail(c *gin.Context) {
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

	resp, err := h.searchCPOService.GetRunDetail(uint(shopID), uint(runID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "获取 CPO 报名详情失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "success", Data: resp})
}

func (h *SearchCPOHandler) StartAutomationRun(c *gin.Context) {
	var req dto.SearchCPOAutomationRunRequest
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

	resp, err := h.searchCPOService.StartAutomationRun(claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "创建 CPO 自动化任务失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "已创建 CPO 自动化任务", Data: resp})
}

func (h *SearchCPOHandler) ListAutomationRuns(c *gin.Context) {
	var req dto.SearchCPOAutomationRunListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{Code: 400, Message: "请求参数错误"})
		return
	}

	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{Code: 403, Message: "无权访问该店铺"})
		return
	}

	resp, err := h.searchCPOService.ListAutomationRuns(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "获取 CPO 自动化历史失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "success", Data: resp})
}

func (h *SearchCPOHandler) GetAutomationRunDetail(c *gin.Context) {
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

	resp, err := h.searchCPOService.GetAutomationRunDetail(uint(shopID), uint(runID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{Code: 500, Message: "获取 CPO 自动化详情失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.Response{Code: 200, Message: "success", Data: resp})
}
