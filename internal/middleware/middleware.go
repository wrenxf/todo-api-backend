package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"todo-api/internal/model"
)

// CORS 允许前端开发服务器跨域访问
func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	return cors.New(config)
}

// Logging 打印请求方法、路径、状态码与耗时
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("[HTTP] %s %s -> %d (%s)",
			c.Request.Method,
			c.Request.URL.RequestURI(),
			c.Writer.Status(),
			time.Since(start).Round(time.Microsecond),
		)
	}
}

// ErrorHandler 捕获 panic，统一返回 JSON
func ErrorHandler(c *gin.Context, recovered interface{}) {
	log.Printf("panic recovered: %v\n%s", recovered, debug.Stack())
	c.AbortWithStatusJSON(http.StatusInternalServerError, model.APIResponse{
		Success:   false,
		Message:   "服务器内部错误",
		Error:     fmt.Sprint(recovered),
		Timestamp: time.Now(),
	})
}
