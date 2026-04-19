# 数据中心系统 - 前后端重构规范

## 为什么
当前数据中心系统需要完成前后端的重构和完整实现，包括后端已完成的RBAC、存储、刮削等功能，以及新增的前端登录和搜索界面。

## 什么变化
- 完成现有后端代码的文档化和规范化
- 添加前端项目，实现登录和数据搜索界面
- 确保前后端API集成正确
- 更新项目架构和文档

## 影响
- 受影响的规格：所有FR和NFR
- 受影响的代码：
  - 后端：internal/, pkg/, cmd/
  - 前端：新增frontend/目录
  - 文档：docs/

## 后端现状分析

### 已实现的后端功能
- **认证系统**：JWT认证（internal/auth/jwt.go, middleware.go）
- **RBAC系统**：角色权限管理（pkg/rbac/rbac.go, internal/storage/rbac_storage.go）
- **MongoDB存储**：数据存储和操作（internal/storage/mongodb.go）
- **日志系统**：zerolog + lumberjack（internal/logger/）
- **API处理器**：HTTP请求处理（internal/api/handlers.go）
- **数据刮削**：异步任务处理（internal/scraper/scraper.go）
- **查询语言**：JQL解析器（pkg/jql/parser.go）

### 后端技术栈
- 语言：Go 1.20
- 框架：Gin
- 数据库：MongoDB
- 认证：JWT
- 日志：zerolog + lumberjack

## 前端需求分析

### 需要实现的前端功能
- **登录页面**：用户身份验证界面
- **搜索页面**：数据查询和结果显示界面
- **API集成**：与后端RESTful API通信
- **状态管理**：用户认证状态和令牌管理
- **路由系统**：页面导航和权限控制

### 前端技术栈
- 框架：React 18 + TypeScript
- UI库：Ant Design
- 构建工具：Vite
- HTTP客户端：Axios
- 路由：React Router v6
- 状态管理：Zustand

## 项目结构

### 整体目录结构
```
datacenter/
├── cmd/
│   └── server/
│       └── main.go              # 后端入口
├── configs/
│   └── config.yaml              # 配置文件
├── docs/
│   ├── architecture.md         # 架构文档
│   ├── rbac.md                  # RBAC文档
│   ├── business-data.md         # 业务数据文档
│   ├── logging.md               # 日志文档
│   ├── authentication.md        # 认证文档
│   ├── query.md                 # 查询文档
│   └── code_structure.md         # 代码结构文档
├── internal/
│   ├── api/
│   │   └── handlers.go          # API处理器
│   ├── auth/
│   │   ├── jwt.go               # JWT实现
│   │   └── middleware.go        # 认证中间件
│   ├── logger/
│   │   ├── logger.go            # 日志器
│   │   └── middleware.go        # 日志中间件
│   ├── models/
│   │   └── models.go            # 数据模型
│   ├── scraper/
│   │   └── scraper.go           # 刮削系统
│   └── storage/
│       ├── mongodb.go           # MongoDB存储
│       └── rbac_storage.go      # RBAC存储
├── pkg/
│   ├── jql/
│   │   └── parser.go            # JQL解析器
│   └── rbac/
│       ├── rbac.go              # RBAC服务
│       └── test/
│           └── rbac_test.go     # RBAC测试
├── frontend/                    # 新增前端目录
│   ├── public/
│   │   └── index.html
│   ├── src/
│   │   ├── assets/
│   │   ├── components/
│   │   │   ├── Layout/
│   │   │   └── Loading/
│   │   ├── pages/
│   │   │   ├── Login/
│   │   │   │   └── LoginPage.tsx
│   │   │   └── Search/
│   │   │       └── SearchPage.tsx
│   │   ├── services/
│   │   │   ├── auth.ts
│   │   │   └── api.ts
│   │   ├── stores/
│   │   │   └── authStore.ts
│   │   ├── hooks/
│   │   ├── utils/
│   │   ├── types/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   └── routes.tsx
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── .env
├── .env
├── go.mod
├── go.sum
└── README.md
```

## 后端API端点

### 认证API
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/login | 用户登录 |
| POST | /api/refresh | 刷新令牌 |

### 用户管理API
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/users | 创建用户 |
| GET | /api/users | 获取用户列表 |
| GET | /api/users/:id | 获取用户详情 |
| PUT | /api/users/:id | 更新用户 |
| DELETE | /api/users/:id | 删除用户 |
| POST | /api/users/:id/roles | 分配角色给用户 |
| DELETE | /api/users/:id/roles/:roleId | 从用户移除角色 |
| GET | /api/users/:id/roles | 获取用户角色列表 |
| GET | /api/users/:id/permissions | 获取用户权限列表 |

### 权限管理API
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/permissions | 创建权限 |
| GET | /api/permissions | 获取权限列表 |
| GET | /api/permissions/:id | 获取权限详情 |
| PUT | /api/permissions/:id | 更新权限 |
| DELETE | /api/permissions/:id | 删除权限 |

### 角色管理API
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/roles | 创建角色 |
| GET | /api/roles | 获取角色列表 |
| GET | /api/roles/:id | 获取角色详情 |
| PUT | /api/roles/:id | 更新角色 |
| DELETE | /api/roles/:id | 删除角色 |
| POST | /api/roles/:id/permissions | 分配权限给角色 |
| DELETE | /api/roles/:id/permissions/:permissionId | 从角色移除权限 |
| GET | /api/roles/:id/permissions | 获取角色权限列表 |
| POST | /api/roles/:id/users | 分配用户给角色 |
| DELETE | /api/roles/:id/users/:userId | 从角色移除用户 |
| GET | /api/roles/:id/users | 获取角色用户列表 |

### 业务数据API
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/data | 创建业务数据 |
| GET | /api/data | 获取业务数据列表 |
| GET | /api/data/:id | 获取业务数据详情 |
| PUT | /api/data/:id | 更新业务数据 |
| DELETE | /api/data/:id | 删除业务数据 |

### 数据查询API
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/query | 执行JQL查询 |

### 集合管理API
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/collections | 创建集合 |
| GET | /api/collections | 获取集合列表 |
| GET | /api/collections/:module | 获取集合详情 |
| PUT | /api/collections/:module | 更新集合 |
| DELETE | /api/collections/:module | 删除集合 |
| POST | /api/collections/:module/indexes | 创建索引 |
| GET | /api/collections/:module/indexes | 获取索引列表 |
| DELETE | /api/collections/:module/indexes/:name | 删除索引 |

### 刮削任务API
| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/scrape/tasks | 提交刮削任务 |
| GET | /api/scrape/tasks | 获取刮削任务列表 |
| GET | /api/scrape/tasks/:id | 获取刮削任务详情 |
| DELETE | /api/scrape/tasks/:id | 删除刮削任务 |

## 前端页面设计

### 登录页面
```
┌─────────────────────────────────────┐
│                                     │
│         [系统Logo/名称]             │
│                                     │
│    ┌─────────────────────────┐     │
│    │     登录到数据中心         │     │
│    └─────────────────────────┘     │
│                                     │
│    ┌─────────────────────────┐     │
│    │ 用户名                    │     │
│    └─────────────────────────┘     │
│                                     │
│    ┌─────────────────────────┐     │
│    │ 密码                     │     │
│    └─────────────────────────┘     │
│                                     │
│    ┌─────────────────────────┐     │
│    │       登录              │     │
│    └─────────────────────────┘     │
│                                     │
└─────────────────────────────────────┘
```

### 搜索页面
```
┌─────────────────────────────────────────────────────┐
│ [Logo]  数据中心          [用户名] ▼  [退出]        │
├─────────────────────────────────────────────────────┤
│                                                     │
│  搜索                                               │
│  ┌─────────────────────────────────────────────┐   │
│  │ 输入JQL查询条件...                            │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  [搜索]  [清除]                                     │
│                                                     │
│  ─────────────────────────────────────────────────  │
│                                                     │
│  搜索结果                                            │
│  ┌─────────────────────────────────────────────┐   │
│  │ 模块    │ 描述              │ 创建时间   │ 操作 │   │
│  ├─────────────────────────────────────────────┤   │
│  │ movie   │ 电影数据            │ 2024-01-01 │ 查看 │   │
│  │ book    │ 图书数据            │ 2024-01-02 │ 查看 │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  [上一页] [1] [2] [3] [下一页]                      │
│                                                     │
└─────────────────────────────────────────────────────┘
```

## 功能需求

### FR-1: 用户认证和授权
- 基于JWT的认证，包括令牌生成和验证
- 基于角色的访问控制，具有权限继承
- 密码哈希和安全最佳实践
- 令牌刷新机制

### FR-2: 用户和权限管理
- 创建、读取、更新、删除用户
- 创建、读取、更新、删除角色
- 创建、读取、更新、删除权限
- 向用户分配角色（多对多）
- 向角色分配权限（多对多）

### FR-3: 业务数据管理
- 创建、读取、更新、删除业务数据
- 不同模块的动态集合管理
- 字段定义和验证
- 软删除和数据恢复

### FR-4: 数据刮削系统
- 提交刮削任务
- 带工作池的异步处理
- 任务状态管理
- 结果存储和错误处理

### FR-5: API端点
- 所有系统功能的RESTful API
- 用于认证和授权的中间件
- 请求验证和错误处理

### FR-6: 日志记录和审计
- 具有不同日志级别的结构化日志
- 文件轮换和日志管理
- 关键操作的审计跟踪

### FR-7: 查询语言支持
- 用于数据过滤的JQL（JSON查询语言）
- 支持复杂查询

### FR-8: 前端登录界面（新增）
- 用户名和密码输入
- 表单验证
- 错误提示
- 登录状态管理

### FR-9: 前端搜索界面（新增）
- JQL查询输入
- 搜索结果展示
- 分页功能
- 数据详情查看

## 非功能需求

### NFR-1: 性能
- 正常操作的API响应时间低于200ms
- 支持并发刮削任务
- 高效的MongoDB查询性能

### NFR-2: 安全性
- 使用bcrypt进行密码加密
- JWT令牌验证和过期
- 所有受保护资源的权限检查
- 输入验证以防止注入攻击

### NFR-3: 可扩展性
- 模块化架构，便于扩展
- 支持多个MongoDB集合
- 可配置的刮削工作池大小

### NFR-4: 可靠性
- 优雅的错误处理
- 健壮的MongoDB连接管理
- 刮削任务的重试机制

### NFR-5: 可维护性
- 清晰的代码结构和组织
- 全面的文档
- 组件之间定义明确的接口

### NFR-6: 前端用户体验
- 响应式设计
- 清晰的加载状态
- 友好的错误提示

## 验收标准

### AC-1: 后端用户认证
- **Given**: 具有有效凭证的用户
- **When**: 他们向/api/login发送POST请求
- **Then**: 他们收到JWT令牌和用户信息
- **验证**: `programmatic`

### AC-2: 后端RBAC权限检查
- **Given**: 具有特定角色和权限的用户
- **When**: 他们访问受保护的资源
- **Then**: 根据他们的权限授予或拒绝访问
- **验证**: `programmatic`

### AC-3: 后端业务数据管理
- **Given**: 具有适当权限的用户
- **When**: 他们创建业务数据
- **Then**: 数据存储在适当的MongoDB集合中
- **验证**: `programmatic`

### AC-4: 后端数据刮削
- **Given**: 提交了有效的刮削任务
- **When**: 系统处理任务
- **Then**: 任务状态更新，结果存储
- **验证**: `programmatic`

### AC-5: 后端日志记录
- **Given**: 执行系统操作
- **When**: 生成日志
- **Then**: 日志格式正确并存储
- **验证**: `human-judgment`

### AC-6: 前端登录页面UI
- **Given**: 用户在浏览器中访问登录页面
- **When**: 页面加载完成
- **Then**: 显示用户名和密码输入框及登录按钮
- **验证**: `human-judgment`

### AC-7: 前端登录功能
- **Given**: 用户在登录页面输入有效凭证
- **When**: 点击登录按钮
- **Then**: 系统验证并跳转或显示错误
- **验证**: `programmatic`

### AC-8: 前端搜索页面UI
- **Given**: 已登录用户在浏览器中访问搜索页面
- **When**: 页面加载完成
- **Then**: 显示搜索输入框和搜索按钮
- **验证**: `human-judgment`

### AC-9: 前端搜索功能
- **Given**: 已登录用户在搜索页面输入查询条件
- **When**: 点击搜索按钮
- **Then**: 显示搜索结果列表
- **验证**: `programmatic`

### AC-10: 前后端集成
- **Given**: 前端发送API请求
- **When**: 请求到达后端
- **Then**: 前后端正确通信和数据交换
- **验证**: `programmatic`

## 开放问题
- [ ] 前端是否需要用户注册功能？
- [ ] 搜索结果是否需要支持导出？
- [ ] 是否需要实现数据详情编辑页面？
- [ ] 是否需要实现数据导入功能？