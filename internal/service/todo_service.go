package service

import (
	"fmt"
	"time"

	"todo-api/internal/model"
	"todo-api/internal/repository"
)

// TodoService 任务服务接口
type TodoService interface {
	Create(todo *model.Todo) error
	GetByID(id int) (*model.Todo, error)
	List(filter model.TodoFilter) ([]model.Todo, int, error)
	Update(todo *model.Todo) error
	Delete(id int) error
	ToggleStatus(id int) (*model.Todo, error)
	Statistics() (*model.TodoStatistics, error)
}

// TodoServiceImpl 任务服务实现，内部持有仓储
type TodoServiceImpl struct {
	repo repository.TodoRepository
}

// NewTodoService 构造函数，通过依赖注入接收仓储，返回服务接口
func NewTodoService(repo repository.TodoRepository) TodoService {
	return &TodoServiceImpl{repo: repo}
}

// Create 创建任务：填充时间戳、默认状态与优先级
func (s *TodoServiceImpl) Create(todo *model.Todo) error {
	now := time.Now()
	todo.CreatedAt = now
	todo.UpdatedAt = now
	if todo.Status == "" {
		todo.Status = "pending"
	}
	if todo.Priority == "" {
		todo.Priority = "medium"
	}
	return s.repo.Create(todo)
}

// GetByID 根据 ID 查询任务详情
func (s *TodoServiceImpl) GetByID(id int) (*model.Todo, error) {
	return s.repo.GetByID(id)
}

// Update 更新任务（全字段覆盖，调用方需先取回再改）
func (s *TodoServiceImpl) Update(todo *model.Todo) error {
	todo.UpdatedAt = time.Now()
	return s.repo.Update(todo)
}

// Delete 删除任务
func (s *TodoServiceImpl) Delete(id int) error {
	return s.repo.Delete(id)
}

// List 分页查询任务列表
func (s *TodoServiceImpl) List(filter model.TodoFilter) ([]model.Todo, int, error) {
	return s.repo.List(filter)
}

// ToggleStatus 切换任务完成状态：pending/cancelled <-> completed
func (s *TodoServiceImpl) ToggleStatus(id int) (*model.Todo, error) {
	todo, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if todo.Status == "completed" {
		todo.Status = "pending"
		todo.CompletedAt = nil
	} else {
		todo.Status = "completed"
		now := time.Now()
		todo.CompletedAt = &now
	}
	todo.UpdatedAt = time.Now()

	if err := s.repo.Update(todo); err != nil {
		return nil, fmt.Errorf("任务状态更新失败: %w", err)
	}
	return todo, nil
}

// Statistics 任务统计数据
func (s *TodoServiceImpl) Statistics() (*model.TodoStatistics, error) {
	return s.repo.Statistics()
}

// 保证编译期接口实现
var _ TodoService = (*TodoServiceImpl)(nil)
