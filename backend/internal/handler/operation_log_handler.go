package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"ozon-manager/internal/dto"
	"ozon-manager/internal/repository"
)

type OperationLogHandler struct {
	logRepo *repository.OperationLogRepository
}

func NewOperationLogHandler(logRepo *repository.OperationLogRepository) *OperationLogHandler {
	return &OperationLogHandler{logRepo: logRepo}
}

// GetOperationLogs 获取操作日志列表
// GET /api/v1/operation-logs
func (h *OperationLogHandler) GetOperationLogs(c *gin.Context) {
	var req dto.OperationLogListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.Response{
			Code:    400,
			Message: "请求参数错误",
		})
		return
	}

	// 解析日期参数
	var dateFrom, dateTo time.Time
	if req.DateFrom != "" {
		dateFrom, _ = time.Parse("2006-01-02", req.DateFrom)
	}
	if req.DateTo != "" {
		dateTo, _ = time.Parse("2006-01-02", req.DateTo)
		// 设置为当天结束时间
		dateTo = dateTo.Add(24*time.Hour - time.Second)
	}

	logs, total, err := h.logRepo.FindWithFilters(
		req.UserID,
		req.ShopID,
		req.OperationType,
		dateFrom,
		dateTo,
		req.Page,
		req.PageSize,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    500,
			Message: "获取操作日志失败",
		})
		return
	}

	items := make([]dto.OperationLogItem, 0, len(logs))
	for _, log := range logs {
		item := dto.OperationLogItem{
			ID: log.ID,
			User: dto.UserInfo{
				ID:          log.User.ID,
				Username:    log.User.Username,
				DisplayName: log.User.DisplayName,
				Role:        log.User.Role,
			},
			OperationType:   log.OperationType,
			OperationDetail: log.OperationDetail,
			AffectedCount:   log.AffectedCount,
			Status:          log.Status,
			IPAddress:       log.IPAddress,
			CreatedAt:       log.CreatedAt.Format("2006-01-02 15:04:05"),
		}

		if log.Shop != nil {
			item.Shop = &dto.ShopInfo{
				ID:   log.Shop.ID,
				Name: log.Shop.Name,
			}
		}

		items = append(items, item)
	}

	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "success",
		Data: dto.OperationLogListResponse{
			Total: total,
			Items: items,
		},
	})
}

// GetOperationLogDetail 获取操作日志详情
// GET /api/v1/operation-logs/:id
func (h *OperationLogHandler) GetOperationLogDetail(c *gin.Context) {
	// 简单返回，可以复用列表中的详情
	c.JSON(http.StatusOK, dto.Response{
		Code:    200,
		Message: "请使用列表接口查看详情",
	})
}
