# 01-todo-api

基于 Go + Gin + SQLite 的任务管理 REST API，附带一个 Vue 3 + Vite 的最小联调前端。

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
| 前端 | Vue 3、Vite 5、原生 fetch |
| 中间件 | CORS、请求日志、panic 恢复 |

## 目录结构

```
01-todo-api/                    # 后端仓库
├── main.go              # 后端全部代码（模型 / 数据库 / 服务 / 路由 / 中间件）
├── go.mod
├── go.sum
├── todos.db             # 运行时自动生成，已加入 .gitignore
└── README.md

01-todo-api-frontend/           # 前端，独立目录
├── index.html
├── package.json
├── vite.config.js       # 开发服务器与 /api 代理
├── .env.example         # 后端地址配置示例
├── README.md
└── src/
    ├── main.js
    ├── api.js           # 后端请求封装
    ├── App.vue          # 页面与交互
    └── style.css
```

## 快速开始

### 1. 启动后端

需要 Go 1.21+ 与 CGO 可用的 64 位 gcc（Windows 推荐 MinGW-W64）。

```bash
cd 01-todo-api
go mod tidy
go run .
```

服务地址：<http://localhost:8080>

首次启动会自动创建 `todos.db`、建表建索引，并写入 3 条示例任务。

> Windows 若默认 gcc 是 32 位（如 Strawberry Perl 自带版本），需显式指定编译器：
>
> ```bash
> set CC=D:\IDE\mingw64\bin\gcc.exe   # cmd
> $env:CC="D:\IDE\mingw64\bin\gcc.exe" # PowerShell
> go run .
> ```

### 2. 启动前端

前端位于同级目录 `01-todo-api-frontend`：

```bash
cd ../01-todo-api-frontend
npm install
npm run dev
```

访问地址：<http://localhost:5173>

生产构建与本地预览：

```bash
npm run build
npm run preview
```

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

## 前端如何配置后端地址

前端有两种对接方式，**默认使用方式一**。

### 方式一：走 Vite 代理（推荐，默认）

`../01-todo-api-frontend/src/api.js` 中 `BASE_URL` 默认为空，请求 `/api/...` 由 Vite 转发到后端：

```js
// 01-todo-api-frontend/vite.config.js
proxy: {
  '/api': {
    target: 'http://127.0.0.1:8080',
    changeOrigin: true,
  },
}
```

浏览器看到的是同源请求（`http://localhost:5173/api/...`），因此开发阶段不涉及跨域。

后端换了端口或部署到其他机器时，修改 `VITE_PROXY_TARGET` 即可，无需改代码：

```bash
# 01-todo-api-frontend/.env.local（复制自 .env.example）
VITE_PROXY_TARGET=http://192.168.1.20:9000
```

### 方式二：浏览器直连后端

后端已启用 CORS（允许所有来源）。把 `VITE_API_BASE` 设为后端完整地址，请求将绕过代理直接发出：

```bash
# 01-todo-api-frontend/.env.local
VITE_API_BASE=http://127.0.0.1:8080
```

此时页面地址与接口地址不同源，依赖后端返回的 `Access-Control-Allow-Origin` 头。

> 修改 `.env*` 或 `vite.config.js` 后需重启 `npm run dev`。

## 已验证

- 后端 `go build` 通过，9 个接口均已用 curl 跑通（含 400 / 404 分支）
- 前端 `npm run build` 通过
- 开发服务器 `/api` 代理转发正常，CORS 预检返回 204
