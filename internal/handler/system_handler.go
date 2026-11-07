package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SystemHandler 系统级接口（首页、健康检查、接口文档）
type SystemHandler struct{}

// NewSystemHandler 构造函数
func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

// Home 首页，返回服务基本信息
func (h *SystemHandler) Home(c *gin.Context) {
	c.JSON(http.StatusOK, newResponse("Todo API 服务运行中", gin.H{
		"name":    "todo-api",
		"version": "1.0.0",
		"docs":    "/api/docs",
		"health":  "/api/health",
	}))
}

// Health 健康检查
func (h *SystemHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, newResponse("服务运行正常", gin.H{
		"status": "ok",
		"time":   time.Now(),
	}))
}

// Docs 返回接口清单，方便前端查阅
func (h *SystemHandler) Docs(c *gin.Context) {
	endpoints := []gin.H{
		{"method": "GET", "path": "/api/health", "description": "健康检查"},
		{"method": "GET", "path": "/api/todos", "description": "获取任务列表（page/page_size/status/priority/search）"},
		{"method": "POST", "path": "/api/todos", "description": "创建任务"},
		{"method": "GET", "path": "/api/todos/:id", "description": "获取任务详情"},
		{"method": "PUT", "path": "/api/todos/:id", "description": "更新任务（支持部分字段）"},
		{"method": "DELETE", "path": "/api/todos/:id", "description": "删除任务"},
		{"method": "PATCH", "path": "/api/todos/:id/toggle", "description": "切换完成状态"},
		{"method": "GET", "path": "/api/todos/statistics", "description": "任务统计"},
	}
	c.JSON(http.StatusOK, newResponse("接口文档", endpoints))
}
