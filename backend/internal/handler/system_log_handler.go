package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"ozon-manager/pkg/logger"
)

type SystemLogHandler struct{}

func NewSystemLogHandler() *SystemLogHandler {
	return &SystemLogHandler{}
}

// LogRequest 定义前端传入的日志格式
type LogRequest struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Url     string `json:"url"`
	Stack   string `json:"stack"`
}

// ReceiveFrontendLog 接收前端发送的错误日志并写入本地文件
func (h *SystemLogHandler) ReceiveFrontendLog(c *gin.Context) {
	var req LogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid log format"})
		return
	}
	
	// 在日志中输出带有 [FRONTEND] 标识的前端日志
	// 还可以附带用户的 IP，用户代理等信息
	userAgent := c.Request.UserAgent()
	clientIP := c.ClientIP()
	
	logMsg := fmt.Sprintf("[FRONTEND] %s", req.Message)
	
	fields := []zap.Field{
		zap.String("url", req.Url),
		zap.String("client_ip", clientIP),
		zap.String("user_agent", userAgent),
	}
	
	if req.Stack != "" {
		fields = append(fields, zap.String("stack", req.Stack))
	}

	// 根据前端传来的不同 level，打印不同级别的日志
	switch req.Level {
	case "info":
		logger.Log.Info(logMsg, fields...)
	case "warn":
		logger.Log.Warn(logMsg, fields...)
	case "error":
		logger.Log.Error(logMsg, fields...)
	case "fatal":
		logger.Log.Fatal(logMsg, fields...)
	default:
		// 默认视为 error
		logger.Log.Error(logMsg, fields...)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
