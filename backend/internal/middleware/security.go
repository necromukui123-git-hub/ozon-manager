package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders 添加安全响应头
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// HSTS: 强制 HTTPS (1 年)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// XSS 保护
		c.Header("X-XSS-Protection", "1; mode=block")

		// 点击劫持保护
		c.Header("X-Frame-Options", "DENY")

		// CSP 策略 (允许同源和开发环境的前端)
		c.Header("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}
