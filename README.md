# 01-todo-api

基于 Go + Gin + SQLite 的任务管理 REST API。

> 前端为独立仓库，不在本仓库内。

## 功能

- 任务 CRUD：创建、读取、更新、删除
- 状态管理：`pending` / `completed` / `cancelled`
- 优先级：`low` / `medium` / `high`
- 截止日期（`YYYY-MM-DD`）
- 关键词搜索（标题 + 描述）
- 分页查询、状态/优先级过滤
- 任务统计（总数、各状态数量、完成率）

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.26、Gin v1.12、database/sql |
| 数据库 | SQLite（mattn/go-sqlite3，CGO） |
| 中间件 | CORS、请求日志、panic 恢复 |

## 目录结构

后端采用 `cmd + internal` 标准分层结构，依赖方向单向流动：
`handler → service → repository → database`。

```
01-todo-api/                        # 后端仓库
├── cmd/
│   └── api/
│       └── main.go                 # 程序入口：只做依赖组装与启动
├── internal/                       # 内部包（不可被外部模块导入）
│   ├── model/
│   │   └── todo.go                 # 结构体、请求/响应体、ErrTodoNotFound
│   ├── database/
│   │   └── sqlite.go               # 连接、建表建索引、示例数据 seed
│   ├── repository/
│   │   └── todo_repository.go      # 仓储接口 + SQLite 实现（所有 SQL）
│   ├── service/
│   │   └── todo_service.go         # 业务逻辑（默认值、时间戳、状态切换）
│   ├── middleware/
│   │   └── middleware.go           # CORS、请求日志、panic 恢复
│   ├── handler/
│   │   ├── response.go             # 统一响应与参数校验辅助
│   │   ├── errors.go               # 参数错误构造函数
│   │   ├── system_handler.go       # 首页 / 健康检查 / 接口文档
│   │   └── todo_handler.go         # 任务接口（结构体方法）
│   └── router/
│       └── router.go               # 集中注册路由
├── go.mod
├── go.sum
├── todos.db                        # 运行时自动生成，已加入 .gitignore
└── README.md
```

## 快速开始

### 启动后端

需要 Go 1.21+ 与 CGO 可用的 64 位 gcc（Windows 推荐 MinGW-W64）。

在**项目根目录**执行：

```bash
cd 01-todo-api
go mod tidy
go run ./cmd/api
```

> 入口文件位于 `cmd/api/main.go`，因此不能直接在根目录用 `go run .`（根目录没有 Go 文件）。
> 也可以进入入口目录再运行：`cd cmd\api && go run .`

服务地址：<http://localhost:8080>

首次启动会自动创建 `todos.db`、建表建索引，并写入 3 条示例任务。

> **Windows 编译器提示**：若默认 gcc 是 32 位（如 Strawberry Perl 自带版本），`go build` 会报
> `cc1.exe: sorry, unimplemented: 64-bit mode not compiled in`。需显式指定 64 位编译器：
>
> ```powershell
> # 当前终端临时生效
> $env:CC="D:\IDE\mingw64\bin\gcc.exe"
> go run ./cmd/api
> ```
>
> 想一劳永逸，可写入用户环境变量（`setx` 只对**新开**的终端生效）：
>
> ```powershell
> setx CC "D:\IDE\mingw64\bin\gcc.exe"
> ```
>
> cmd 用户对应写法：`set CC=D:\IDE\mingw64\bin\gcc.exe`

## 接口一览

基础路径 `/api`，统一响应结构：

```json
{ "success": true, "message": "...", "data": {}, "timestamp": "..." }
```

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/health` | 健康检查 |
| GET | `/api/docs` | 接口清单 |
| GET | `/api/todos` | 任务列表，支持 `page`、`page_size`、`status`、`priority`、`search` |
| POST | `/api/todos` | 创建任务，成功返回 201 |
| GET | `/api/todos/:id` | 任务详情 |
| PUT | `/api/todos/:id` | 更新任务（未传字段保持原值） |
| DELETE | `/api/todos/:id` | 删除任务 |
| PATCH | `/api/todos/:id/toggle` | 切换完成状态 |
| GET | `/api/todos/statistics` | 任务统计 |

分页响应：

```json
{
  "items": [],
  "page": 1,
  "page_size": 10,
  "total": 25,
  "total_pages": 3
}
```

### 验证示例

```bash
curl http://localhost:8080/api/health

curl -X POST http://localhost:8080/api/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"学习 Go","description":"完成基础教程","priority":"high","due_date":"2026-12-01"}'

curl "http://localhost:8080/api/todos?page=1&page_size=5&status=pending"
```

## 已验证

- 后端 `go build ./...` 通过（`cmd/api` + `internal` 全部包），9 个接口均已用 curl 跑通（含 400 / 404 分支）
