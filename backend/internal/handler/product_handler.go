package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/middleware"
	"ozon-manager/internal/service"
	"ozon-manager/pkg/excel"
)

type ProductHandler struct {
	productService *service.ProductService
	shopService    *service.ShopService
}

func NewProductHandler(productService *service.ProductService, shopService *service.ShopService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		shopService:    shopService,
	}
}

// GetProducts 获取商品列表
// GET /api/v1/products
func (h *ProductHandler) GetProducts(c *gin.Context) {
	var req dto.ProductListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 检查店铺访问权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	resp, err := h.productService.GetProducts(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "获取商品列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    resp,
	})
}

// GetProduct 获取商品详情
// GET /api/v1/products/:id
func (h *ProductHandler) GetProduct(c *gin.Context) {
	productID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "无效的商品ID",
		})
		return
	}

	product, err := h.productService.GetProductByID(uint(productID))
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Response{
			Code:    404,
			Message: "商品不存在",
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    product,
	})
}

// SyncProducts 同步商品
// POST /api/v1/products/sync
func (h *ProductHandler) SyncProducts(c *gin.Context) {
	var req dto.SyncProductsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 检查店铺访问权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, req.ShopID, claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	count, err := h.productService.SyncProducts(req.ShopID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "同步商品失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "同步成功",
		Data: map[string]int{
			"synced_count": count,
		},
	})
}

// GetStats 获取统计数据
// GET /api/v1/stats/overview
func (h *ProductHandler) GetStats(c *gin.Context) {
	shopID, _ := strconv.ParseUint(c.Query("shop_id"), 10, 32)
	if shopID == 0 {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "缺少shop_id参数",
		})
		return
	}

	// 检查店铺访问权限
	claims := middleware.GetCurrentUser(c)
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, uint(shopID), claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	stats, err := h.productService.GetStats(uint(shopID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "获取统计数据失败",
		})
		return
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data:    stats,
	})
}

// ExportPromotable 导出可推广商品
// GET /api/v1/excel/export-promotable
func (h *ProductHandler) ExportPromotable(c *gin.Context) {
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
	if err := h.shopService.CheckUserAccessByRole(claims.UserID, uint(shopID), claims.Role); err != nil {
		c.JSON(http.StatusForbidden, dto.Response{
			Code:    403,
			Message: "无权访问该店铺",
		})
		return
	}

	// 获取店铺信息
	shop, err := h.shopService.GetShopByID(uint(shopID))
	if err != nil {
		c.JSON(http.StatusNotFound, dto.Response{
			Code:    404,
			Message: "店铺不存在",
		})
		return
	}

	// 获取可推广商品
	products, err := h.productService.GetPromotableProducts(uint(shopID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "获取可推广商品失败",
		})
		return
	}

	// 构建导出数据
	exportData := make([]excel.PromotableProduct, 0, len(products))
	for _, p := range products {
		exportData = append(exportData, excel.PromotableProduct{
			SourceSKU: p.SourceSKU,
			ShopName:  shop.Name,
			Name:      p.Name,
			Price:     p.CurrentPrice,
		})
	}

	// 生成Excel
	f, err := excel.ExportPromotableProducts(exportData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "生成Excel失败",
		})
		return
	}

	// 返回Excel文件
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=promotable_products.xlsx")

	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "写入Excel失败",
		})
	}
}
