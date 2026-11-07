package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"todo-api/internal/model"
)

func newResponse(message string, data interface{}) model.APIResponse {
	return model.APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

func abortWithError(c *gin.Context, code int, message string, err error) {
	resp := model.APIResponse{
		Success:   false,
		Message:   message,
		Timestamp: time.Now(),
	}
	if err != nil {
		resp.Error = err.Error()
	}
	c.JSON(code, resp)
}

// parseID 解析路径参数 id，非法时直接写入 400 响应
func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		abortWithError(c, http.StatusBadRequest, "无效的任务 ID", fmt.Errorf("id 必须为正整数"))
		return 0, false
	}
	return id, true
}

func isValidStatus(value string) bool {
	switch value {
	case "pending", "completed", "cancelled":
		return true
	}
	return false
}

func isValidPriority(value string) bool {
	switch value {
	case "low", "medium", "high":
		return true
	}
	return false
}

func validateDueDate(value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return fmt.Errorf("due_date 必须为 YYYY-MM-DD 格式")
	}
	return nil
}

// isNotFound 判断错误是否为任务不存在
func isNotFound(err error) bool {
	return errors.Is(err, model.ErrTodoNotFound)
}
