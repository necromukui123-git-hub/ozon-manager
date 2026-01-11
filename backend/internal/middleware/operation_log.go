package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"ozon-manager/internal/model"
)

// OperationLogMiddleware 操作日志记录中间件
func OperationLogMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 只记录非GET请求
		if c.Request.Method == "GET" {
			c.Next()
			return
		}

		// 获取请求体
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 记录开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 获取当前用户
		claims := GetCurrentUser(c)
		if claims == nil {
			return
		}

		// 解析操作类型
		operationType := parseOperationType(c.FullPath(), c.Request.Method)
		if operationType == "" {
			return
		}

		// 构建操作详情
		var detail map[string]interface{}
		if len(bodyBytes) > 0 {
			json.Unmarshal(bodyBytes, &detail)
		}
		detailJSON, _ := json.Marshal(detail)

		// 确定状态
		status := "success"
		var errorMessage string
		if c.Writer.Status() >= 400 {
			status = "failed"
			// 可以从响应中提取错误信息
		}

		// 获取shop_id（如果有）
		var shopID *uint
		if sid, exists := c.Get("shop_id"); exists {
			if id, ok := sid.(uint); ok {
				shopID = &id
			}
		}

		// 创建日志记录
		now := time.Now()
		log := model.OperationLog{
			UserID:          claims.UserID,
			ShopID:          shopID,
			OperationType:   operationType,
			OperationDetail: datatypes.JSON(detailJSON),
			Status:          status,
			ErrorMessage:    errorMessage,
			IPAddress:       c.ClientIP(),
			UserAgent:       c.Request.UserAgent(),
			CreatedAt:       startTime,
			CompletedAt:     &now,
		}

		// 异步保存日志
		go func() {
			db.Create(&log)
		}()
	}
}

// parseOperationType 解析操作类型
func parseOperationType(path, method string) string {
	operationMap := map[string]string{
		"POST /api/v1/promotions/batch-enroll":          "batch_enroll",
		"POST /api/v1/promotions/process-loss":          "process_loss",
		"POST /api/v1/promotions/remove-reprice-promote": "remove_reprice_promote",
		"POST /api/v1/excel/import-loss":                "import_loss",
		"POST /api/v1/excel/import-reprice":             "import_reprice",
		"POST /api/v1/products/sync":                    "sync_products",
		"POST /api/v1/users":                            "create_user",
		"PUT /api/v1/users/:id/status":                  "update_user_status",
		"PUT /api/v1/users/:id/shops":                   "update_user_shops",
		"POST /api/v1/shops":                            "create_shop",
		"PUT /api/v1/shops/:id":                         "update_shop",
		"DELETE /api/v1/shops/:id":                      "delete_shop",
	}

	key := method + " " + path
	return operationMap[key]
}
