package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"todo-api/internal/model"
	"todo-api/internal/service"
)

// TodoHandler 任务相关接口的处理器，持有服务层依赖
type TodoHandler struct {
	svc service.TodoService
}

// NewTodoHandler 构造函数，注入任务服务
func NewTodoHandler(svc service.TodoService) *TodoHandler {
	return &TodoHandler{svc: svc}
}

// List 获取任务列表
func (h *TodoHandler) List(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", errPage())
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", errPageSize())
		return
	}

	status := c.Query("status")
	if status != "" && !isValidStatus(status) {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", errStatus())
		return
	}
	priority := c.Query("priority")
	if priority != "" && !isValidPriority(priority) {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", errPriority())
		return
	}

	filter := model.TodoFilter{
		Status:   status,
		Priority: priority,
		Search:   strings.TrimSpace(c.Query("search")),
		Page:     page,
		PageSize: pageSize,
	}

	todos, total, err := h.svc.List(filter)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "获取任务列表失败", err)
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	c.JSON(http.StatusOK, newResponse("获取任务列表成功", model.PaginatedResponse{
		Items:      todos,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}))
}

// Create 创建任务
func (h *TodoHandler) Create(c *gin.Context) {
	var req model.TodoCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", err)
		return
	}
	if err := validateDueDate(req.DueDate); err != nil {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", err)
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}

	todo := &model.Todo{
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Status:      "pending",
		Priority:    priority,
	}
	if req.DueDate != "" {
		due := req.DueDate
		todo.DueDate = &due
	}

	if err := h.svc.Create(todo); err != nil {
		abortWithError(c, http.StatusInternalServerError, "创建任务失败", err)
		return
	}
	c.JSON(http.StatusCreated, newResponse("任务创建成功", todo))
}

// Get 获取任务详情
func (h *TodoHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	todo, err := h.svc.GetByID(id)
	if err != nil {
		if isNotFound(err) {
			abortWithError(c, http.StatusNotFound, "任务不存在", err)
			return
		}
		abortWithError(c, http.StatusInternalServerError, "获取任务详情失败", err)
		return
	}
	c.JSON(http.StatusOK, newResponse("获取任务详情成功", todo))
}

// Update 更新任务，未传字段保持原值
func (h *TodoHandler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req model.TodoUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", err)
		return
	}

	todo, err := h.svc.GetByID(id)
	if err != nil {
		if isNotFound(err) {
			abortWithError(c, http.StatusNotFound, "任务不存在", err)
			return
		}
		abortWithError(c, http.StatusInternalServerError, "获取任务详情失败", err)
		return
	}

	if req.Title != nil {
		todo.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		todo.Description = *req.Description
	}
	if req.Priority != nil {
		todo.Priority = *req.Priority
	}
	if req.Status != nil {
		todo.Status = *req.Status
		if *req.Status == "completed" {
			if todo.CompletedAt == nil {
				now := time.Now()
				todo.CompletedAt = &now
			}
		} else {
			todo.CompletedAt = nil
		}
	}
	if req.DueDate != nil {
		if err := validateDueDate(*req.DueDate); err != nil {
			abortWithError(c, http.StatusBadRequest, "请求参数无效", err)
			return
		}
		if *req.DueDate == "" {
			todo.DueDate = nil
		} else {
			due := *req.DueDate
			todo.DueDate = &due
		}
	}

	if err := h.svc.Update(todo); err != nil {
		if isNotFound(err) {
			abortWithError(c, http.StatusNotFound, "任务不存在", err)
			return
		}
		abortWithError(c, http.StatusInternalServerError, "更新任务失败", err)
		return
	}
	c.JSON(http.StatusOK, newResponse("任务更新成功", todo))
}

// Delete 删除任务
func (h *TodoHandler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := h.svc.Delete(id); err != nil {
		if isNotFound(err) {
			abortWithError(c, http.StatusNotFound, "任务不存在", err)
			return
		}
		abortWithError(c, http.StatusInternalServerError, "删除任务失败", err)
		return
	}
	c.JSON(http.StatusOK, newResponse("任务删除成功", gin.H{"id": id}))
}

// Toggle 切换任务完成状态
func (h *TodoHandler) Toggle(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	todo, err := h.svc.ToggleStatus(id)
	if err != nil {
		if isNotFound(err) {
			abortWithError(c, http.StatusNotFound, "任务不存在", err)
			return
		}
		abortWithError(c, http.StatusInternalServerError, "切换任务状态失败", err)
		return
	}
	c.JSON(http.StatusOK, newResponse("任务状态已切换", todo))
}

// Statistics 任务统计
func (h *TodoHandler) Statistics(c *gin.Context) {
	stat, err := h.svc.Statistics()
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "获取任务统计失败", err)
		return
	}
	c.JSON(http.StatusOK, newResponse("获取任务统计成功", stat))
}
