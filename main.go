package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
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
	Priority    string `json:"priority" binding:"oneof=low medium high"`
	DueDate     string `json:"due_date"`
}

// TodoUpdateRequest 更新任务请求
type TodoUpdateRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	State       *string `json:"state,omitempty" binding:"oneof=pending completed cancelled"`
	Priority    *string `json:"priority,omitempty" binding:"oneof=low medium high"`
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
	Items     interface{} `json:"items"`
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
	Total     int         `json:"total"`
	TotalPage int         `json:"total_page"`
}

// 创建数据库表
func createTable(db *sql.DB) error {
	query := `
	create table if not exists todos(
	id integer primary key autoincrement,
	title varchar(200) not null,
	description text,
	status varchar(20) default 'pending' check(status in('pending','completed','cancelled')),
	priority varchar(10) default 'medium' check(priority in('low','medium','high')),
	due_date date,
	created_at datetime default current_timestamp,
	updated_at datetime default current_timestamp,
	completed_at datetime
	);
`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("创建任务表失败:%w", err)
	}
	return nil
}

// 初始化数据库
func initDatabase() *sql.DB {
	//打开数据库连接
	db, err := sql.Open("sqlite3", "./todos.db")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	//检查连接是否健康,测试连接
	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}

	//建表
	if err := createTable(db); err != nil {
		log.Fatal("数据库创建失败:", err)
	}

	return db
}

// TodoService 任务服务接口
type TodoService interface {
	Create(todo *Todo) error
	GetByID(id int) (*Todo, error)
	Update(todo *Todo) error
	Delete(id int) error
	List(filter TodoFilter) ([]Todo, int, error)
	ToggleStatus(id int, status string) (*Todo, error)
}

// TodoFilter 任务查询过滤器
type TodoFilter struct {
	Status   string //任务状态筛选
	Priority string //优先级筛选
	Search   string //关键词搜索
	Page     int    //当前页码
	PageSize int    //每页条数
}

// TodoServiceImpl 任务服务实现
// TodoServiceImpl 是任务服务接口的具体实现结构体，内部持有数据库连接
type TodoServiceImpl struct {
	db *sql.DB
}

// NewTodoService 是任务服务的构造函数，通过依赖注入的方式接收数据库连接，并返回服务接口
func NewTodoService(db *sql.DB) TodoService {
	return &TodoServiceImpl{db: db}
}

// Create 创建一个新的待办任务
// 该方法会自动设置任务的创建时间、更新时间，如果状态为空则默认为 pending
// 插入成功后，会将数据库生成的自增 ID 回填到传入的 todo 结构体中
func (s *TodoServiceImpl) Create(todo *Todo) error {
	query := `
	insert into todos(title,description,status,priority,due_date,created_at,updated_at)
	values(?,?,?,?,?,?,?)
`
	now := time.Now()
	todo.CreatedAt = now
	todo.UpdatedAt = now
	if todo.Status == "" {
		todo.Status = "pending"
	}
	result, err := s.db.Exec(query, todo.Title, todo.Description, todo.Status, todo.Priority, todo.DueDate, todo.CreatedAt, todo.UpdatedAt)
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
// 参数 id: 待查询的任务唯一标识
// 返回值: 查询成功返回包含任务信息的结构体指针；失败返回带有上下文信息的错误
func (s TodoServiceImpl) GetByID(id int) (*Todo, error) {
	query := `
	select id,title,description,status,priority,due_date,created_at,updated_at
	from todo
	where id=?
`
	todo := &Todo{}
	err := s.db.QueryRow(query, id).Scan(
		&todo.ID,
		&todo.Title,
		&todo.Description,
		&todo.Status,
		&todo.Priority,
		&todo.DueDate,
		&todo.CreatedAt,
		&todo.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("任务不存在: id=%d", id)
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	return todo, nil
}

func (s TodoServiceImpl) Update(todo *Todo) error {
	query := `
	update todos
	set title=?,description=?,status=?,priority=?,due_date=?,updated_at=?
	where id=?
`
	todo.UpdatedAt = time.Now()

	result, err := s.db.Exec(query,
		todo.Title,
		todo.Description,
		todo.Status,
		todo.Priority,
		todo.DueDate,
		todo.UpdatedAt,
		todo.ID,
	)

	if err != nil {
		return fmt.Errorf("更新任务失败:%w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取更新结果失败:%w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("任务不存在:id=%d", todo.ID)
	}
	return nil
}

func (s TodoServiceImpl) Delete(id int) error {
	query := `
	delete from todos
	where id=?
`
	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("更新任务失败:%w", err)
	}
	rowAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取更新任务结果失败:%w", err)
	}
	if rowAffected == 0 {
		return fmt.Errorf("任务不存在:id=%d", id)
	}
	return nil
}

func (s TodoServiceImpl) List(filter TodoFilter) ([]Todo, int, error) {
	//构建where条件
	whereClause := "where 1=1"
	args := []interface{}{}

	if filter.Status != "" {
		whereClause += "and status = ?"
		args = append(args, filter.Status)
	}

	if filter.Priority != "" {
		whereClause += "and priority = ?"
		args = append(args, filter.Priority)
	}

	if filter.Search != "" {
		whereClause += "and (title like ? or description like ?)"
		searchTerm := "%" + whereClause + "%"
		args = append(args, searchTerm, searchTerm)
	}

	//获取总数
	countQuery := "select count(*) from todos" + whereClause
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("获取任务总数失败: %w", err)
	}

	//分页查询
	query := `
	select id,title,description,status,priority,due_date,created_at,updated_at,completed_at
	from todos` + whereClause + `
	order by created_at desc,priority desc
	limit ? offset ?
`
	offset := (filter.Page - 1) * filter.PageSize
	args = append(args, filter.PageSize, offset)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询任务列表失败:%w", err)
	}
	defer rows.Close()

	var todos []Todo

	for rows.Next() {
		todo := &Todo{}
		var completedAt sql.NullTime

		err := rows.Scan(
			&todo.ID, &todo.Title, &todo.Description, &todo.Status, &todo.Priority,
			&todo.DueDate, &todo.CreatedAt, &todo.UpdatedAt, &completedAt,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("扫描任务数据失败: %w", err)
		}
		if completedAt.Valid {
			todo.CompletedAt = &completedAt.Time
		}
		todos = append(todos, *todo)
	}
	return todos, total, nil
}

// ToggleStatus 切换任务完成状态：pending <-> completed
func (s TodoServiceImpl) ToggleStatus(id int, status string) (*Todo, error) {
	var todo Todo
	var completdAt sql.NullTime

	err := s.db.QueryRow(
		`select id,title,description,status,priority,due_date,created_at,updated_at,completed_at
			   from todos where id = ?`,
		id).Scan(&todo.ID,
		&todo.Title,
		&todo.Description,
		&todo.Status,
		&todo.Priority,
		&todo.DueDate,
		&todo.CreatedAt,
		&todo.UpdatedAt,
		&completdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("任务不存在:%d", id)
		}
		return nil, fmt.Errorf("查询任务失败%w", err)
	}
	if completdAt.Valid {
		todo.CompletedAt = &completdAt.Time
	}

	if todo.Status == "completed" {
		todo.Status = "pending"
		todo.CompletedAt = nil
	} else {
		todo.Status = "completed"

		now := time.Now()
		todo.CompletedAt = &now
	}

	// 将新状态和完成时间写回数据库
	_, err = s.db.Exec(`
		update todos set status = ?,completed_at = ?,updated_at = ? where id =?`,
		todo.Status,
		todo.CompletedAt,
		todo.UpdatedAt,
		todo.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("任务状态更新失败: %w", err)
	}

	return &todo, nil
}

func setupRoutes(r *gin.Engine) {
	//api路由
	api := r.Group("/api")
	{
		//健康检查
		api.GET("/health", handleHealth)

		//api文档
		api.GET("/docs", handleDocs)

		//任务路由
		todos := api.Group("/todos")
		{
			// 获取任务列表
			todos.GET("", handleListTodos)
			// 创建新任务
			todos.POST("", handleCreateTodo)
			// 获取单个任务详情
			todos.GET("/:id", handleGetTodo)
			// 完整更新某个任务
			todos.PUT("/:id", handleUpdateTodo)
			// 删除某个任务
			todos.DELETE("/:id", handleDeleteTodo)
			// 切换某个任务的完成状态
			todos.PATCH("/:id/toggle", handleToggleTodo)
			// 获取任务统计信息
			todos.GET("/statistics", handleTodoStatistics)
		}
	}
}
