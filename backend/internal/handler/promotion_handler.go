package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/middleware"
	"ozon-manager/internal/service"
	"ozon-manager/pkg/excel"
)

type PromotionHandler struct {
	promotionService *service.PromotionService
	shopService      *service.ShopService
}

func NewPromotionHandler(promotionService *service.PromotionService, shopService *service.ShopService) *PromotionHandler {
	return &PromotionHandler{
		promotionService: promotionService,
		shopService:      shopService,
	}
}

// BatchEnroll 批量报名促销活动 (功能1)
// POST /api/v1/promotions/batch-enroll
func (h *PromotionHandler) BatchEnroll(c *gin.Context) {
	var req dto.BatchEnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, req.ShopID, claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	// 设置shop_id供操作日志使用
	c.Set("shop_id", req.ShopID)

	resp, err := h.promotionService.BatchEnrollPromotions(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "批量报名失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "批量报名完成",
		Data:    resp,
	})
}

// ProcessLoss 处理亏损商品 (功能2)
// POST /api/v1/promotions/process-loss
func (h *PromotionHandler) ProcessLoss(c *gin.Context) {
	var req dto.ProcessLossRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, req.ShopID, claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	c.Set("shop_id", req.ShopID)

	resp, err := h.promotionService.ProcessLossProducts(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "处理亏损商品失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "处理完成",
		Data:    resp,
	})
}

// RemoveRepricePromote 移除-改价-重新推广 (功能4)
// POST /api/v1/promotions/remove-reprice-promote
func (h *PromotionHandler) RemoveRepricePromote(c *gin.Context) {
	var req dto.RemoveRepricePromoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, req.ShopID, claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	c.Set("shop_id", req.ShopID)

	if err := h.promotionService.RemoveRepricePromote(&req); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "操作失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "操作完成",
	})
}

// ImportLoss 导入亏损商品Excel
// POST /api/v1/excel/import-loss
func (h *PromotionHandler) ImportLoss(c *gin.Context) {
	shopID, _ := strconv.ParseUint(c.PostForm("shop_id"), 10, 32)
	if shopID == 0 {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "缺少shop_id参数",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, uint(shopID), claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	// 获取上传的文件
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请上传Excel文件",
		})
		return
	}
	defer file.Close()

	// 读取文件内容
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "读取文件失败",
		})
		return
	}

	// 解析Excel
	rows, err := excel.ImportLossProductsFromBytes(fileBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "解析Excel失败: " + err.Error(),
		})
		return
	}

	// 转换数据格式
	items := make([]struct {
		SourceSKU string
		NewPrice  float64
	}, len(rows))
	for i, row := range rows {
		items[i] = struct {
			SourceSKU string
			NewPrice  float64
		}{
			SourceSKU: row.SourceSKU,
			NewPrice:  row.NewPrice,
		}
	}

	// 导入亏损商品
	lossProductIDs, err := h.promotionService.ImportLossProducts(uint(shopID), items)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "导入失败: " + err.Error(),
		})
		return
	}

	c.Set("shop_id", uint(shopID))

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "导入成功",
		Data: dto.ImportLossResponse{
			Success:        true,
			ImportedCount:  len(lossProductIDs),
			LossProductIDs: lossProductIDs,
		},
	})
}

// ImportReprice 导入改价商品Excel (功能4的Excel入口)
// POST /api/v1/excel/import-reprice
func (h *PromotionHandler) ImportReprice(c *gin.Context) {
	shopID, _ := strconv.ParseUint(c.PostForm("shop_id"), 10, 32)
	if shopID == 0 {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "缺少shop_id参数",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, uint(shopID), claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	// 获取上传的文件
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请上传Excel文件",
		})
		return
	}
	defer file.Close()

	// 读取文件内容
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "读取文件失败",
		})
		return
	}

	// 解析Excel
	rows, err := excel.ImportLossProductsFromBytes(fileBytes)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "解析Excel失败: " + err.Error(),
		})
		return
	}

	// 构建请求
	req := &dto.RemoveRepricePromoteRequest{
		ShopID:   uint(shopID),
		Products: make([]dto.RepriceItem, len(rows)),
	}
	for i, row := range rows {
		req.Products[i] = dto.RepriceItem{
			SourceSKU: row.SourceSKU,
			NewPrice:  row.NewPrice,
		}
	}

	// 执行操作
	if err := h.promotionService.RemoveRepricePromote(req); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "操作失败: " + err.Error(),
		})
		return
	}

	c.Set("shop_id", uint(shopID))

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "操作成功",
		Data: map[string]int{
			"processed_count": len(rows),
		},
	})
}

// SyncActions 同步促销活动
// POST /api/v1/promotions/sync-actions
func (h *PromotionHandler) SyncActions(c *gin.Context) {
	var req struct {
		ShopID uint `json:"shop_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, req.ShopID, claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	actions, err := h.promotionService.SyncPromotionActions(req.ShopID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "同步促销活动失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "同步成功",
		Data:    actions,
	})
}

// GetActions 获取促销活动列表
// GET /api/v1/promotions/actions
func (h *PromotionHandler) GetActions(c *gin.Context) {
	var req dto.GetActionsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, req.ShopID, claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	actions, err := h.promotionService.GetPromotionActions(req.ShopID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "获取促销活动失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    actions,
	})
}

// CreateManualAction 手动添加促销活动
// POST /api/v1/promotions/actions/manual
func (h *PromotionHandler) CreateManualAction(c *gin.Context) {
	var req dto.CreateManualActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, req.ShopID, claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	action, err := h.promotionService.CreateManualAction(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "添加促销活动失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "添加成功",
		Data:    action,
	})
}

// DeleteAction 删除促销活动
// DELETE /api/v1/promotions/actions/:id
func (h *PromotionHandler) DeleteAction(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的活动ID",
		})
		return
	}

	shopID, _ := strconv.ParseUint(c.Query("shop_id"), 10, 32)
	if shopID == 0 {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "缺少shop_id参数",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, uint(shopID), claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	if err := h.promotionService.DeletePromotionAction(uint(shopID), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "删除成功",
	})
}

// BatchEnrollV2 批量报名到指定活动
// POST /api/v1/promotions/batch-enroll-v2
func (h *PromotionHandler) BatchEnrollV2(c *gin.Context) {
	var req dto.BatchEnrollV2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, req.ShopID, claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	c.Set("shop_id", req.ShopID)

	resp, err := h.promotionService.BatchEnrollToActions(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "批量报名失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "批量报名完成",
		Data:    resp,
	})
}

// ProcessLossV2 处理亏损商品（支持选择活动）
// POST /api/v1/promotions/process-loss-v2
func (h *PromotionHandler) ProcessLossV2(c *gin.Context) {
	var req dto.ProcessLossV2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, req.ShopID, claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	c.Set("shop_id", req.ShopID)

	resp, err := h.promotionService.ProcessLossProductsV2(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "处理亏损商品失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "处理完成",
		Data:    resp,
	})
}

// RemoveRepricePromoteV2 移除-改价-重新推广（支持选择活动）
// POST /api/v1/promotions/remove-reprice-promote-v2
func (h *PromotionHandler) RemoveRepricePromoteV2(c *gin.Context) {
	var req dto.RemoveRepricePromoteV2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 检查权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccess(claims.UserID, req.ShopID, claims.Role == "admin"); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	c.Set("shop_id", req.ShopID)

	if err := h.promotionService.RemoveRepricePromoteV2(&req); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "操作失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "操作完成",
	})
}

// DownloadLossTemplate 下载亏损商品导入模板
// GET /api/v1/excel/template/loss
func (h *PromotionHandler) DownloadLossTemplate(c *gin.Context) {
	f, err := excel.CreateLossTemplate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "生成模板失败",
		})
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=loss_template.xlsx")

	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "下载失败",
		})
	}
}
