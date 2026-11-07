package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"todo-api/internal/model"
)

// TodoRepository 定义任务持久化接口
type TodoRepository interface {
	Create(todo *model.Todo) error
	GetByID(id int) (*model.Todo, error)
	List(filter model.TodoFilter) ([]model.Todo, int, error)
	Update(todo *model.Todo) error
	Delete(id int) error
	Statistics() (*model.TodoStatistics, error)
}

// SQLiteTodoRepository 任务仓储的 SQLite 实现
type SQLiteTodoRepository struct {
	db *sql.DB
}

// NewSQLiteTodoRepository 构造函数，注入数据库连接
func NewSQLiteTodoRepository(db *sql.DB) TodoRepository {
	return &SQLiteTodoRepository{db: db}
}

// Create 创建一个新的待办任务
func (r *SQLiteTodoRepository) Create(todo *model.Todo) error {
	query := `
	insert into todos(title,description,status,priority,due_date,created_at,updated_at)
	values(?,?,?,?,?,?,?)
`
	result, err := r.db.Exec(query, todo.Title, todo.Description, todo.Status, todo.Priority, todo.DueDate, todo.CreatedAt, todo.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建任务失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("获取任务ID失败: %w", err)
	}
	todo.ID = int(id)
	return nil
}

// GetByID 根据指定的任务 ID 查询对应的待办任务详情
func (r *SQLiteTodoRepository) GetByID(id int) (*model.Todo, error) {
	query := `
	select id,title,description,status,priority,due_date,created_at,updated_at,completed_at
	from todos
	where id=?
`
	todo := &model.Todo{}
	var dueDate sql.NullTime
	var completedAt sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&todo.ID,
		&todo.Title,
		&todo.Description,
		&todo.Status,
		&todo.Priority,
		&dueDate,
		&todo.CreatedAt,
		&todo.UpdatedAt,
		&completedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: id=%d", model.ErrTodoNotFound, id)
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}

	if dueDate.Valid {
		value := dueDate.Time.Format("2006-01-02")
		todo.DueDate = &value
	}
	if completedAt.Valid {
		todo.CompletedAt = &completedAt.Time
	}
	return todo, nil
}

// Update 更新任务（全字段覆盖，调用方需先取回再改）
func (r *SQLiteTodoRepository) Update(todo *model.Todo) error {
	query := `
	update todos
	set title=?,description=?,status=?,priority=?,due_date=?,completed_at=?,updated_at=?
	where id=?
`
	result, err := r.db.Exec(query,
		todo.Title,
		todo.Description,
		todo.Status,
		todo.Priority,
		todo.DueDate,
		todo.CompletedAt,
		todo.UpdatedAt,
		todo.ID,
	)
	if err != nil {
		return fmt.Errorf("更新任务失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取更新结果失败: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: id=%d", model.ErrTodoNotFound, todo.ID)
	}
	return nil
}

// Delete 删除任务
func (r *SQLiteTodoRepository) Delete(id int) error {
	query := `
	delete from todos
	where id=?
`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除任务失败: %w", err)
	}
	rowAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除任务结果失败: %w", err)
	}
	if rowAffected == 0 {
		return fmt.Errorf("%w: id=%d", model.ErrTodoNotFound, id)
	}
	return nil
}

// List 分页查询任务列表，返回当页数据与总数
func (r *SQLiteTodoRepository) List(filter model.TodoFilter) ([]model.Todo, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 10
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}

	//构建where条件：统一放进切片再 join，避免拼接时空格遗漏
	conditions := []string{"1=1"}
	args := []interface{}{}

	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Priority != "" {
		conditions = append(conditions, "priority = ?")
		args = append(args, filter.Priority)
	}
	if filter.Search != "" {
		conditions = append(conditions, "(title like ? or description like ?)")
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm, searchTerm)
	}
	whereClause := " where " + strings.Join(conditions, " and ")

	//获取总数
	countQuery := "select count(*) from todos" + whereClause
	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("获取任务总数失败: %w", err)
	}

	//分页查询：priority 需按语义排序，不能依赖字典序
	query := `
	select id,title,description,status,priority,due_date,created_at,updated_at,completed_at
	from todos` + whereClause + `
	order by created_at desc,
	         case priority when 'high' then 1 when 'medium' then 2 else 3 end asc
	limit ? offset ?
`
	offset := (filter.Page - 1) * filter.PageSize
	args = append(args, filter.PageSize, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询任务列表失败: %w", err)
	}
	defer rows.Close()

	// 初始化为空切片，保证无数据时 JSON 序列化为 [] 而不是 null
	todos := make([]model.Todo, 0)
	for rows.Next() {
		var todo model.Todo
		var dueDate sql.NullTime
		var completedAt sql.NullTime

		err := rows.Scan(
			&todo.ID, &todo.Title, &todo.Description, &todo.Status, &todo.Priority,
			&dueDate, &todo.CreatedAt, &todo.UpdatedAt, &completedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描任务数据失败: %w", err)
		}
		if dueDate.Valid {
			value := dueDate.Time.Format("2006-01-02")
			todo.DueDate = &value
		}
		if completedAt.Valid {
			todo.CompletedAt = &completedAt.Time
		}
		todos = append(todos, todo)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("遍历任务数据失败: %w", err)
	}
	return todos, total, nil
}

// Statistics 任务统计数据
func (r *SQLiteTodoRepository) Statistics() (*model.TodoStatistics, error) {
	query := `
	select
		count(*),
		coalesce(sum(case when status='pending' then 1 else 0 end),0),
		coalesce(sum(case when status='completed' then 1 else 0 end),0),
		coalesce(sum(case when status='cancelled' then 1 else 0 end),0)
	from todos
`
	stat := &model.TodoStatistics{}
	err := r.db.QueryRow(query).Scan(&stat.Total, &stat.Pending, &stat.Completed, &stat.Cancelled)
	if err != nil {
		return nil, fmt.Errorf("统计任务失败: %w", err)
	}
	if stat.Total > 0 {
		stat.CompletionRate = float64(stat.Completed) / float64(stat.Total)
	}
	return stat, nil
}

// 保证编译期接口实现
var _ TodoRepository = (*SQLiteTodoRepository)(nil)
