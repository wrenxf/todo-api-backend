package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// ============================================================
// 数据模型 (Models)
// ============================================================

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

// ============================================================
// 数据库 (Database)
// ============================================================

// createTable 创建任务表及索引
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
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("创建任务表失败: %w", err)
	}

	indexes := []string{
		`create index if not exists idx_todos_status on todos(status)`,
		`create index if not exists idx_todos_priority on todos(priority)`,
		`create index if not exists idx_todos_created_at on todos(created_at)`,
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
	}
	return nil
}

// seedTodos 空库时写入示例数据，方便前端联调时直接看到内容
func seedTodos(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`select count(*) from todos`).Scan(&count); err != nil {
		return fmt.Errorf("统计任务数量失败: %w", err)
	}
	if count > 0 {
		return nil
	}

	now := time.Now()
	samples := []struct {
		title    string
		desc     string
		priority string
		due      string
	}{
		{"学习 Go 语言", "完成 Go 语言基础教程的学习", "high", now.AddDate(0, 0, 3).Format("2006-01-02")},
		{"整理项目文档", "补全 README 中的接口说明", "medium", now.AddDate(0, 0, 7).Format("2006-01-02")},
		{"看一部电影", "周末放松一下", "low", ""},
	}

	for _, s := range samples {
		var due interface{}
		if s.due != "" {
			due = s.due
		}
		_, err := db.Exec(
			`insert into todos(title,description,status,priority,due_date,created_at,updated_at) values(?,?,?,?,?,?,?)`,
			s.title, s.desc, "pending", s.priority, due, now, now,
		)
		if err != nil {
			return fmt.Errorf("写入示例数据失败: %w", err)
		}
	}
	log.Printf("已写入 %d 条示例任务", len(samples))
	return nil
}

// initDatabase 初始化数据库连接并建表
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

	// SQLite 同一时刻只允许一个写连接，限制为 1 可避免 database is locked
	db.SetMaxOpenConns(1)

	//建表
	if err := createTable(db); err != nil {
		log.Fatal("数据库创建失败:", err)
	}

	//空库时写入示例数据
	if err := seedTodos(db); err != nil {
		log.Fatal("初始化示例数据失败:", err)
	}

	return db
}

// ============================================================
// 服务层 (Services)
// ============================================================

// TodoService 任务服务接口
type TodoService interface {
	Create(todo *Todo) error
	GetByID(id int) (*Todo, error)
	List(filter TodoFilter) ([]Todo, int, error)
	Update(todo *Todo) error
	Delete(id int) error
	ToggleStatus(id int) (*Todo, error)
	Statistics() (*TodoStatistics, error)
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
	if todo.Priority == "" {
		todo.Priority = "medium"
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
func (s *TodoServiceImpl) GetByID(id int) (*Todo, error) {
	query := `
	select id,title,description,status,priority,due_date,created_at,updated_at,completed_at
	from todos
	where id=?
`
	todo := &Todo{}
	var dueDate sql.NullTime
	var completedAt sql.NullTime

	err := s.db.QueryRow(query, id).Scan(
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
			return nil, fmt.Errorf("%w: id=%d", ErrTodoNotFound, id)
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
func (s *TodoServiceImpl) Update(todo *Todo) error {
	query := `
	update todos
	set title=?,description=?,status=?,priority=?,due_date=?,completed_at=?,updated_at=?
	where id=?
`
	todo.UpdatedAt = time.Now()

	result, err := s.db.Exec(query,
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
		return fmt.Errorf("%w: id=%d", ErrTodoNotFound, todo.ID)
	}
	return nil
}

// Delete 删除任务
func (s *TodoServiceImpl) Delete(id int) error {
	query := `
	delete from todos
	where id=?
`
	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除任务失败: %w", err)
	}
	rowAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取删除任务结果失败: %w", err)
	}
	if rowAffected == 0 {
		return fmt.Errorf("%w: id=%d", ErrTodoNotFound, id)
	}
	return nil
}

// List 分页查询任务列表，返回当页数据与总数
func (s *TodoServiceImpl) List(filter TodoFilter) ([]Todo, int, error) {
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
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
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

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询任务列表失败: %w", err)
	}
	defer rows.Close()

	// 初始化为空切片，保证无数据时 JSON 序列化为 [] 而不是 null
	todos := make([]Todo, 0)
	for rows.Next() {
		var todo Todo
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

// ToggleStatus 切换任务完成状态：pending/cancelled <-> completed
func (s *TodoServiceImpl) ToggleStatus(id int) (*Todo, error) {
	todo, err := s.GetByID(id)
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

	_, err = s.db.Exec(`
		update todos set status = ?, completed_at = ?, updated_at = ? where id = ?`,
		todo.Status,
		todo.CompletedAt,
		todo.UpdatedAt,
		todo.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("任务状态更新失败: %w", err)
	}
	return todo, nil
}

// Statistics 任务统计数据
func (s *TodoServiceImpl) Statistics() (*TodoStatistics, error) {
	query := `
	select
		count(*),
		coalesce(sum(case when status='pending' then 1 else 0 end),0),
		coalesce(sum(case when status='completed' then 1 else 0 end),0),
		coalesce(sum(case when status='cancelled' then 1 else 0 end),0)
	from todos
`
	stat := &TodoStatistics{}
	err := s.db.QueryRow(query).Scan(&stat.Total, &stat.Pending, &stat.Completed, &stat.Cancelled)
	if err != nil {
		return nil, fmt.Errorf("统计任务失败: %w", err)
	}
	if stat.Total > 0 {
		stat.CompletionRate = float64(stat.Completed) / float64(stat.Total)
	}
	return stat, nil
}

// ============================================================
// 中间件 (Middleware)
// ============================================================

// corsMiddleware 允许前端开发服务器跨域访问
func corsMiddleware() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	return cors.New(config)
}

// loggingMiddleware 打印请求方法、路径、状态码与耗时
func loggingMiddleware() gin.HandlerFunc {
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

// errorHandlerMiddleware 捕获 panic，统一返回 JSON
func errorHandlerMiddleware(c *gin.Context, recovered interface{}) {
	log.Printf("panic recovered: %v\n%s", recovered, debug.Stack())
	c.AbortWithStatusJSON(http.StatusInternalServerError, APIResponse{
		Success:   false,
		Message:   "服务器内部错误",
		Error:     fmt.Sprint(recovered),
		Timestamp: time.Now(),
	})
}

// ============================================================
// 路由处理 (Handlers)
// ============================================================

func newResponse(message string, data interface{}) APIResponse {
	return APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

func abortWithError(c *gin.Context, code int, message string, err error) {
	resp := APIResponse{
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

// handleHome 首页，返回服务基本信息
func handleHome(c *gin.Context) {
	c.JSON(http.StatusOK, newResponse("Todo API 服务运行中", gin.H{
		"name":    "todo-api",
		"version": "1.0.0",
		"docs":    "/api/docs",
		"health":  "/api/health",
	}))
}

// handleHealth 健康检查
func handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, newResponse("服务运行正常", gin.H{
		"status": "ok",
		"time":   time.Now(),
	}))
}

// handleDocs 返回接口清单，方便前端查阅
func handleDocs(c *gin.Context) {
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

// handleListTodos 获取任务列表
func handleListTodos(c *gin.Context, svc TodoService) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", fmt.Errorf("page 必须为正整数"))
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", fmt.Errorf("page_size 取值范围为 1-100"))
		return
	}

	status := c.Query("status")
	if status != "" && !isValidStatus(status) {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", fmt.Errorf("status 只能为 pending/completed/cancelled"))
		return
	}
	priority := c.Query("priority")
	if priority != "" && !isValidPriority(priority) {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", fmt.Errorf("priority 只能为 low/medium/high"))
		return
	}

	filter := TodoFilter{
		Status:   status,
		Priority: priority,
		Search:   strings.TrimSpace(c.Query("search")),
		Page:     page,
		PageSize: pageSize,
	}

	todos, total, err := svc.List(filter)
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "获取任务列表失败", err)
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	c.JSON(http.StatusOK, newResponse("获取任务列表成功", PaginatedResponse{
		Items:      todos,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}))
}

// handleCreateTodo 创建任务
func handleCreateTodo(c *gin.Context, svc TodoService) {
	var req TodoCreateRequest
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

	todo := &Todo{
		Title:       strings.TrimSpace(req.Title),
		Description: req.Description,
		Status:      "pending",
		Priority:    priority,
	}
	if req.DueDate != "" {
		due := req.DueDate
		todo.DueDate = &due
	}

	if err := svc.Create(todo); err != nil {
		abortWithError(c, http.StatusInternalServerError, "创建任务失败", err)
		return
	}
	c.JSON(http.StatusCreated, newResponse("任务创建成功", todo))
}

// handleGetTodo 获取任务详情
func handleGetTodo(c *gin.Context, svc TodoService) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	todo, err := svc.GetByID(id)
	if err != nil {
		if errors.Is(err, ErrTodoNotFound) {
			abortWithError(c, http.StatusNotFound, "任务不存在", err)
			return
		}
		abortWithError(c, http.StatusInternalServerError, "获取任务详情失败", err)
		return
	}
	c.JSON(http.StatusOK, newResponse("获取任务详情成功", todo))
}

// handleUpdateTodo 更新任务，未传字段保持原值
func handleUpdateTodo(c *gin.Context, svc TodoService) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var req TodoUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abortWithError(c, http.StatusBadRequest, "请求参数无效", err)
		return
	}

	todo, err := svc.GetByID(id)
	if err != nil {
		if errors.Is(err, ErrTodoNotFound) {
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

	if err := svc.Update(todo); err != nil {
		if errors.Is(err, ErrTodoNotFound) {
			abortWithError(c, http.StatusNotFound, "任务不存在", err)
			return
		}
		abortWithError(c, http.StatusInternalServerError, "更新任务失败", err)
		return
	}
	c.JSON(http.StatusOK, newResponse("任务更新成功", todo))
}

// handleDeleteTodo 删除任务
func handleDeleteTodo(c *gin.Context, svc TodoService) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	if err := svc.Delete(id); err != nil {
		if errors.Is(err, ErrTodoNotFound) {
			abortWithError(c, http.StatusNotFound, "任务不存在", err)
			return
		}
		abortWithError(c, http.StatusInternalServerError, "删除任务失败", err)
		return
	}
	c.JSON(http.StatusOK, newResponse("任务删除成功", gin.H{"id": id}))
}

// handleToggleTodo 切换任务完成状态
func handleToggleTodo(c *gin.Context, svc TodoService) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	todo, err := svc.ToggleStatus(id)
	if err != nil {
		if errors.Is(err, ErrTodoNotFound) {
			abortWithError(c, http.StatusNotFound, "任务不存在", err)
			return
		}
		abortWithError(c, http.StatusInternalServerError, "切换任务状态失败", err)
		return
	}
	c.JSON(http.StatusOK, newResponse("任务状态已切换", todo))
}

// handleTodoStatistics 任务统计
func handleTodoStatistics(c *gin.Context, svc TodoService) {
	stat, err := svc.Statistics()
	if err != nil {
		abortWithError(c, http.StatusInternalServerError, "获取任务统计失败", err)
		return
	}
	c.JSON(http.StatusOK, newResponse("获取任务统计成功", stat))
}

// ============================================================
// 路由注册与入口
// ============================================================

func setupRoutes(r *gin.Engine, svc TodoService) {
	// 首页
	r.GET("/", handleHome)

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
			todos.GET("", func(c *gin.Context) { handleListTodos(c, svc) })
			// 创建新任务
			todos.POST("", func(c *gin.Context) { handleCreateTodo(c, svc) })
			// 获取任务统计信息（静态路径先于 /:id 注册）
			todos.GET("/statistics", func(c *gin.Context) { handleTodoStatistics(c, svc) })
			// 获取单个任务详情
			todos.GET("/:id", func(c *gin.Context) { handleGetTodo(c, svc) })
			// 更新某个任务
			todos.PUT("/:id", func(c *gin.Context) { handleUpdateTodo(c, svc) })
			// 删除某个任务
			todos.DELETE("/:id", func(c *gin.Context) { handleDeleteTodo(c, svc) })
			// 切换某个任务的完成状态
			todos.PATCH("/:id/toggle", func(c *gin.Context) { handleToggleTodo(c, svc) })
		}
	}

	// 未匹配路由统一返回 JSON
	r.NoRoute(func(c *gin.Context) {
		abortWithError(c, http.StatusNotFound, "接口不存在", fmt.Errorf("%s %s", c.Request.Method, c.Request.URL.Path))
	})
}

func main() {
	db := initDatabase()
	defer db.Close()

	svc := NewTodoService(db)

	r := gin.New()
	r.Use(loggingMiddleware(), gin.CustomRecovery(errorHandlerMiddleware), corsMiddleware())
	setupRoutes(r, svc)

	addr := ":8080"
	log.Printf("Todo API 已启动: http://localhost%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("服务启动失败:", err)
	}
}
