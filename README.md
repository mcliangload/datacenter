# 数据中心系统

一个基于Go和React的数据中心管理系统，提供用户认证、RBAC权限管理、业务数据管理和数据刮削功能。

## 技术栈

### 后端
- **语言**: Go 1.20
- **框架**: Gin
- **数据库**: MongoDB
- **认证**: JWT
- **日志**: zerolog + lumberjack

### 前端
- **框架**: React 18 + TypeScript
- **构建工具**: Vite
- **UI组件库**: Ant Design
- **HTTP客户端**: Axios
- **路由**: React Router v6
- **状态管理**: Zustand

## 项目结构

```
datacenter/
├── cmd/server/          # 后端入口
├── configs/             # 配置文件
├── docs/                # 文档
├── internal/             # 内部包
│   ├── api/            # API处理器
│   ├── auth/           # 认证
│   ├── logger/         # 日志
│   ├── models/         # 数据模型
│   ├── scraper/        # 数据刮削
│   └── storage/        # 存储层
├── pkg/                # 公共包
│   ├── jql/           # JQL解析器
│   └── rbac/          # RBAC服务
├── frontend/           # 前端项目
└── docs/              # 文档
```

## 后端运行

### 环境要求
- Go 1.20+
- MongoDB 4.4+
- 环境变量配置

### 环境变量
在项目根目录创建 `.env` 文件：

```env
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=datacenter
JWT_SECRET=your-secret-key
JWT_EXPIRATION=24h
REFRESH_EXPIRATION=168h
LOG_LEVEL=info
LOG_PATH=./logs
PORT=8080
SCRAPER_WORKERS=4
```

### 启动后端

```bash
# 安装依赖
go mod tidy

# 编译
go build ./...

# 运行
go run cmd/server/main.go
```

后端服务将在 http://localhost:8080 启动。

### API端点

认证：
- POST /api/auth/login - 用户登录

用户管理：
- GET/POST/PUT/DELETE /api/users

权限管理：
- GET/POST/PUT/DELETE /api/permissions

角色管理：
- GET/POST/PUT/DELETE /api/roles

业务数据：
- GET/POST/PUT/DELETE /api/business

数据查询：
- POST /api/query

集合管理：
- GET/POST/PUT/DELETE /api/collections

刮削任务：
- GET/POST/DELETE /api/scraper/tasks

## 前端运行

### 环境要求
- Node.js 16+
- npm 或 yarn

### 启动前端

```bash
# 进入前端目录
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

前端将在 http://localhost:5173 启动。

### 构建生产版本

```bash
npm run build
```

构建产物将输出到 `frontend/dist` 目录。

## 功能特性

### 用户认证
- JWT令牌认证
- 令牌刷新机制
- 密码加密存储

### RBAC权限管理
- 用户-角色多对多关系
- 角色-权限多对多关系
- 权限继承机制

### 业务数据管理
- CRUD操作
- 动态集合管理
- 字段定义验证
- 软删除和数据恢复

### 数据刮削
- 异步任务处理
- 工作池并发处理
- 任务状态管理
- 错误重试机制

### JQL查询
- 支持条件查询（=, !=, >, <, >=, <=, ~）
- 支持逻辑操作符（AND, OR, NOT）
- 支持括号分组

## 测试

### 后端测试

```bash
go test ./...
```

### 前端测试

```bash
cd frontend
npm test
```

## 文档

更多文档请参考 `docs/` 目录：
- [架构文档](docs/architecture.md)
- [RBAC文档](docs/rbac.md)
- [业务数据文档](docs/business-data.md)
- [日志文档](docs/logging.md)
- [认证文档](docs/authentication.md)
- [查询文档](docs/query.md)