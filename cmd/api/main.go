package main

import (
	"log"

	"todo-api/internal/database"
	"todo-api/internal/handler"
	"todo-api/internal/repository"
	"todo-api/internal/router"
	"todo-api/internal/service"
)

func main() {
	// 初始化数据库连接、建表与示例数据
	db := database.Init("./todos.db")
	defer db.Close()

	// 依赖组装：repository -> service -> handler
	repo := repository.NewSQLiteTodoRepository(db)
	svc := service.NewTodoService(repo)
	todoHandler := handler.NewTodoHandler(svc)
	systemHandler := handler.NewSystemHandler()

	// 构建路由
	r := router.Setup(todoHandler, systemHandler)

	// 启动服务
	addr := ":8080"
	log.Printf("Todo API 已启动: http://localhost%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("服务启动失败:", err)
	}
}
