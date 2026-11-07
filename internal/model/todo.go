package model

import (
	"errors"
	"time"
)

// Todo 任务结构体
type Todo struct {
	ID          int        `json:"id"`
	Title       string     `json:"title" binding:"required,min=1,max=200"`
	Description string     `json:"description" binding:"max=1000"`
	Status      string     `json:"status" binding:"oneof=pending completed cancelled"`
	Priority    string     `json:"priority" binding:"oneof=low medium high"`
	DueDate     *string    `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// TodoCreateRequest 创建任务请求
type TodoCreateRequest struct {
	Title       string `json:"title" binding:"required,min=1,max=200"`
	Description string `json:"description" binding:"max=1000"`
	Priority    string `json:"priority" binding:"omitempty,oneof=low medium high"`
	DueDate     string `json:"due_date"`
}

// TodoUpdateRequest 更新任务请求
// 全部使用指针，未传的字段保持原值，实现“部分更新”
type TodoUpdateRequest struct {
	Title       *string `json:"title,omitempty" binding:"omitempty,min=1,max=200"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=1000"`
	Status      *string `json:"status,omitempty" binding:"omitempty,oneof=pending completed cancelled"`
	Priority    *string `json:"priority,omitempty" binding:"omitempty,oneof=low medium high"`
	DueDate     *string `json:"due_date,omitempty"`
}

// APIResponse 通用API响应
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// PaginatedResponse 分页响应
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	Total      int         `json:"total"`
	TotalPages int         `json:"total_pages"`
}

// TodoStatistics 任务统计
type TodoStatistics struct {
	Total          int     `json:"total"`
	Pending        int     `json:"pending"`
	Completed      int     `json:"completed"`
	Cancelled      int     `json:"cancelled"`
	CompletionRate float64 `json:"completion_rate"`
}

// TodoFilter 任务查询过滤器
type TodoFilter struct {
	Status   string //任务状态筛选
	Priority string //优先级筛选
	Search   string //关键词搜索
	Page     int    //当前页码
	PageSize int    //每页条数
}

// ErrTodoNotFound 任务不存在，供 handler 判定 404
var ErrTodoNotFound = errors.New("任务不存在")
