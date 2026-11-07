package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"todo-api/internal/handler"
	"todo-api/internal/middleware"
	"todo-api/internal/model"
)

// Setup 构建并返回配置好中间件与路由的 gin 引擎
func Setup(todoHandler *handler.TodoHandler, systemHandler *handler.SystemHandler) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Logging(), gin.CustomRecovery(middleware.ErrorHandler), middleware.CORS())

	// 首页
	r.GET("/", systemHandler.Home)

	//api路由
	api := r.Group("/api")
	{
		//健康检查
		api.GET("/health", systemHandler.Health)

		//api文档
		api.GET("/docs", systemHandler.Docs)

		//任务路由
		todos := api.Group("/todos")
		{
			// 获取任务列表
			todos.GET("", todoHandler.List)
			// 创建新任务
			todos.POST("", todoHandler.Create)
			// 获取任务统计信息（静态路径先于 /:id 注册）
			todos.GET("/statistics", todoHandler.Statistics)
			// 获取单个任务详情
			todos.GET("/:id", todoHandler.Get)
			// 更新某个任务
			todos.PUT("/:id", todoHandler.Update)
			// 删除某个任务
			todos.DELETE("/:id", todoHandler.Delete)
			// 切换某个任务的完成状态
			todos.PATCH("/:id/toggle", todoHandler.Toggle)
		}
	}

	// 未匹配路由统一返回 JSON
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, model.APIResponse{
			Success:   false,
			Message:   "接口不存在",
			Error:     c.Request.Method + " " + c.Request.URL.Path,
			Timestamp: time.Now(),
		})
	})

	return r
}
