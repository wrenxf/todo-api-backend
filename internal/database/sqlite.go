package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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

// Init 初始化数据库连接并建表
func Init(path string) *sql.DB {
	//打开数据库连接
	db, err := sql.Open("sqlite3", path)
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
