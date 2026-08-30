# 数据中心系统（Datacenter）

基于共享 NFS 存储的集群数据中心：以「集合」为单位管理计算光刻测试数据（model、版图、recipe 等），
支持自定义标签定义、用户提供刮削脚本提取标签、按标签查询。

- 后端：Go + Gin
- 存储：MongoDB（元数据/标签/权限/任务记录），文件本体仅存 NFS
- 需求文档：[需求分解.md](./需求分解.md)
- 部署指南：[部署指南.md](./部署指南.md)（发布到网上/服务器：Linux + systemd + Nginx HTTPS / Docker Compose / 局域网）
- 安全方案：[安全增强方案.md](./安全增强方案.md)（分级：P0 必修 / P1 建议 / P2 可选）
- 优化方案：[系统优化方案.md](./系统优化方案.md)（业务功能/性能/稳定性/工程四类，B1/B2/B3 批次）

## 快速开始

前置条件：Go 1.22+、MongoDB（本地或远程均可）。

```bash
# 1. 拉取依赖
go mod tidy

# 2. 启动 API 服务（默认读取 config/config.yaml，可用 DATACENTER_* 环境变量覆盖）
go run ./cmd/server -config config/config.yaml

# 3. 启动刮削子系统（独立进程，与 API 服务通过 MongoDB 任务队列协作）
go run ./cmd/scraper -config config/config.yaml
```

启动后浏览器访问 **http://localhost:8080** 即可使用 Web 界面（前端已通过 go:embed 打包进服务端二进制，无需单独部署）。

首次启动会自动：

1. 创建 MongoDB 索引（用户名/集合名唯一、分页、任务领取等）；
2. 若系统中尚无任何 admin 用户，创建默认管理员（默认 `admin / admin123`，请在生产环境通过
   `DATACENTER_BOOTSTRAP_ADMIN_USERNAME/PASSWORD` 覆盖）。

> 沙箱/离线环境提示：`scripts/goenv.ps1` 将 Go 缓存重定向到工作区并使用 goproxy.cn 镜像，
> 用法：`. .\scripts\goenv.ps1; go build ./...`

## 本地一键演示（Windows）

提供一组脚本，按顺序执行即可拉起完整环境并灌入海量演示数据：

```powershell
# 1. 重建本地 MongoDB（Docker，mongo:7）并挂载命名数据卷 datacenter-mongo-data
#    （数据持久化到卷中，容器删除/重建不丢数据；需本机已装 Docker Desktop）
& .\scripts\mongo-up.ps1

# 2. 灌入演示数据：model/case/layout/layer 四个集合 + 真实文件树（.nfsdata）+ 关联关系
#    默认 3000 模型 + 2000 版图 + 每版图随机 4~14 图层（约 1.8 万）+ 3000 用例，共约 2.6 万数据项、
#    2.4 万条关联边；可用参数调整数量（-Models/-Layouts/-Layers/-Cases），可重复执行（默认先清空重建）
& .\scripts\seed.ps1

# 3. 启动 API 服务 + 内嵌前端（默认 :18080，数据库 datacenter）
& .\scripts\start-server.ps1

# 4. 启动刮削子系统（独立进程）
& .\scripts\start-scraper.ps1

# 5. 验证：登录 → 概览统计 → DQL 查询（含 parent/ancestor 关系限定）
& .\scripts\verify.ps1
```

演示数据说明（字段基于计算光刻领域知识定义，标签类型覆盖 string/int/float/bool/date/enum/array/object）：

| 集合 | 含义 | 主要标签 |
|---|---|---|
| model | OPC 模型（光学/光刻胶/刻蚀） | node、model_type、lib_type、source_shape、wavelength、na、sigma_in/out、polarization、flare、mask3d、accuracy_rms、anchor_points、status、version、calibration（object）、keywords |
| layout | 版图设计数据（GDS/OASIS/DEF） | node、cell、format、file_size、area_um2、density、layer_count、drc_status、library、modified |
| layer | 版图图层（版图的**子项**，parent_child 关系） | name（M1/V2/OD/POLY…）、layer_type、purpose、min_width、min_space、pitch、density_min/max、opc_treatment、epe_violations、lvs_status |
| case | OPC 测试用例（**引用** model/layout） | node、corner、purpose、priority、status、expect/measured_cd、cd_error、meef、dose、focus、wafer_count、start_date |

关联关系：`layout → layer`（parent_child，单父树，图层物化路径 `ancestors` 已写入）；`case → model`、
`case → layout`（reference，meta 标注 usage）。DQL 中可用 `ancestor = "<layoutId>"` 查整个版图图层子树、
`parent = "<layerId>"` 查其所属版图。

## 前端界面

DeepSeek 风格 Web 界面（蓝色主调 `#4d6bfe`、**毛玻璃质感**、卡片悬浮高亮、数据列表化），
hash 路由单页应用，**左侧固定导航栏**布局：

| 导航 | 页面 | 功能 |
|---|---|---|
| 📊 仪表盘 | 概览 | 集合/数据项/刮削任务/**关联关系**统计卡片（按状态/类型），最近集合与最近任务列表 |
| 🔍 数据查询 | DQL | 输入 DQL 语句跨集合查询数据项（AND/OR、括号、=、!=、>、>=、<、<=、IN、EXISTS、LIKE、`collection` 集合限定、**`parent`/`ancestor` 关联限定**）；内置「💡 查询示例」提示；结果列表含**关联徽标**，支持详情/重刮/**策略化删除弹窗**；「新增数据项」入口 |
| 📖 DQL 语法 | 帮助页 | 独立语法说明页：运算符/字段（含关联字段）/值类型规则表 + 示例（复制按钮） |
| 🗂 集合管理 | 集合列表 | 卡片网格 + 分页 + **悬浮高亮**；admin 新建集合（指定初始集合管理员） |
| — | 集合详情 | 仅数据管理：概览（编辑描述/更换管理员/删除集合）、标签定义（增删，enum/array/object 约束）、刮削脚本配置、成员授权/移除操作工、**删除策略配置**（children/incoming，集合管理员） |
| ⚙ 刮削管理 | 全局任务 | 全部刮削任务列表（状态过滤 + 分页），普通用户仅可见自己参与的集合 |
| 👥 权限管理（admin） | 用户管理 | 创建用户（角色选择）、启用/禁用、删除 |
| 👤 个人设置 | 个人中心 | 个人信息展示、修改本人密码（校验原密码） |
| — | 数据项详情 | 标签展示与编辑（手动标签优先）、**关联关系卡片（出/入边、添加关联（搜索+批量+父子快捷字段）、树视图）**、手动刮削、刮削历史 |

视觉特征：左侧毛玻璃侧边栏（`backdrop-filter: blur` + 半透明），卡片/统计卡/列表行悬浮时
上浮 + 蓝色描边高亮；集合以卡片展开，数据（数据项/任务/用户/成员/标签）以列表/表格展示；
**全局 Toast 提示**（右下角毛玻璃，成功/错误/信息分色，自动消失、点击关闭）——所有操作反馈
不再以页面顶部横幅出现。

前端代码位于 `internal/web/static/`（原生 JS，无构建依赖），通过 `//go:embed` 内嵌进二进制。

### 按钮 → 接口对接对照表

| 前端按钮/操作 | 后端接口 |
|---|---|
| 登录 / 退出 | `POST /auth/login`、`GET /auth/me` |
| 仪表盘统计 | `GET /stats/overview` |
| 新建集合 | `POST /collections` |
| 集合卡片 → 详情、刷新 | `GET /collections/:id` |
| 编辑描述 | `PATCH /collections/:id` |
| 更换集合管理员（admin） | `PUT /collections/:id/admin` |
| 删除集合（admin） | `DELETE /collections/:id` |
| 添加/删除标签（集合管理员） | `PUT /collections/:id/tags`（全量替换） |
| 配置刮削脚本（集合管理员） | `PUT /collections/:id/script` |
| 授权/移除操作工（集合管理员） | `POST /collections/:id/members`、`DELETE /collections/:id/members/:userId` |
| 添加数据项（直接/刮削） | `POST /collections/:id/items` |
| **DQL 查询（数据查询页）** | `POST /dql/query` `{dql, page, page_size}` |
| 数据项重刮 / 详情页手动刮削 | `POST /items/:id/scrape` |
| 数据项详情 / 编辑（标签+路径） | `GET /items/:itemId`、`PATCH /items/:itemId` |
| 删除数据项 | `DELETE /items/:itemId` |
| 刮削历史 / 全局任务列表（刮削管理） | `GET /items/:itemId/scrape-tasks`、`GET /scrape-tasks?status=` |
| 创建用户（admin） | `POST /users` |
| 禁用/启用、删除用户（admin） | `PATCH /users/:id`、`DELETE /users/:id` |
| 修改本人密码（个人设置） | `POST /auth/password` |

## DQL（DataQueryLanguage）

类似 JQL/SQL 的数据查询语言，**跨集合**查询数据项（标签字段），由后端词法/语法解析 + AST 构建 MongoDB 过滤条件执行。

### 语法

```
expr     := orExpr
orExpr   := andExpr (OR andExpr)*        -- AND 优先级高于 OR
andExpr  := primary (AND primary)*
primary  := '(' expr ')' | 条件
条件     := 字段 运算符 值
运算符   := = | != | > | >= | < | <= | IN | EXISTS | LIKE
```

- **字段**：标签名（裸标识符或双引号包裹，支持中文）；`collection` 为特殊字段（限定集合范围）
- **值**：字符串（单/双引号，支持 `\` 转义）、整数/浮点（含负数）、`true`/`false`；`IN` 接值列表（可带括号）；裸词（如 `name = demo`）按字符串处理
- **LIKE**：string/enum 字段的包含匹配（大小写不敏感，正则特殊字符自动转义）
- **EXISTS**：`field EXISTS true|false`（判断标签是否存在）
- **collection**：`collection = "集合名"`、`collection != "x"`、`collection IN ("a","b")`；不写则查询用户有权访问的全部集合

### 示例

```
model_name = "demo-model"
age >= 3 AND stage IN ("dev", "test")
(name LIKE "opc" OR version = "1.0") AND age EXISTS true
collection = "光刻模型库" AND age >= 1
```

### 类型校验

按目标集合的标签定义做类型校验与值规范化（int/float/bool/date/enum/string）；
跨集合类型不一致时数值宽松转换为 float64（MongoDB 数值跨类型按值比较）；范围查询仅限 int/float/date；
LIKE 仅限 string/enum；引用不存在的标签或集合、无权限集合都会给出明确错误。

### 前端对接冒烟测试

```bash
node .tools/ui-test/ui.mjs   # jsdom 加载真实前端代码，点击真实按钮，请求打到真实后端
```

覆盖 50 项断言：登录/退出、仪表盘统计与最近列表、集合管理（新建/标签/脚本/成员授权、**不展示数据**、
**删除策略配置与保存**）、权限管理（创建用户）、**数据查询（DQL 新增数据项、等值/LIKE 查询、语法错误提示、
示例提示点击填入、DQL 语法帮助页、结果行重刮/详情/编辑）**、**关联关系（详情卡片、添加关联（搜索+批量）、
出边列表、树视图、DQL parent 查询、策略化删除（deny 拒绝 → 级联删除））**、刮削管理（状态过滤）、
个人设置（原密码校验），全部走真实 HTTP 请求。

## 验证

```bash
# 健康检查（探活 MongoDB）
curl http://localhost:8080/healthz

# 登录获取 token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 携带 token 访问受保护接口
curl http://localhost:8080/api/v1/auth/me -H "Authorization: Bearer <token>"
```

统一响应格式：

```json
{ "code": 0, "message": "ok", "data": { } }
```

`code` 为业务码（0 成功；1xxx 通用；2xxx 用户/权限；3xxx 集合；4xxx 数据项；5xxx 刮削）；
`data` 缺省表示无返回数据。

## API 一览（/api/v1）

### 认证（公开）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /auth/login | 登录，返回 JWT |
| GET | /auth/me | 当前用户（需登录） |
| POST | /auth/logout | 登出（无状态，客户端丢弃 token） |

### 用户（admin 专属，公共权限）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /users | 创建用户 {username, password, role} |
| GET | /users?page=&page_size=&keyword= | 用户列表 |
| PATCH | /users/:id | 修改 {password?, role?, status?} |
| DELETE | /users/:id | 删除用户 |

### 集合（集合级权限逐集合判定，见需求分解 §4）

| 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|
| POST | /collections | admin | 创建集合 {name, description?, tag_schema?, initial_admin_id} |
| GET | /collections | 登录 | 列表（admin 全部；其他用户仅参与的集合） |
| GET | /collections/:id | 成员 | 详情 |
| PATCH | /collections/:id | 集合管理员 | 修改基础信息 {description} |
| DELETE | /collections/:id | admin | 删除（级联删元数据，不动 NFS 文件） |
| GET | /collections/:id/tags | 成员 | 标签定义 |
| PUT | /collections/:id/tags | 集合管理员 | 全量替换标签定义 |
| PATCH | /collections/:id/tags | 集合管理员 | 增量合并标签定义 |
| PUT | /collections/:id/script | 集合管理员 | 注册刮削脚本 {path}（绝对路径、每集合唯一） |
| GET | /collections/:id/members | 成员 | 成员列表 |
| POST | /collections/:id/members | 集合管理员 | 授权操作工 {user_id} |
| DELETE | /collections/:id/members/:userId | 集合管理员 | 移除操作工 |
| PUT | /collections/:id/admin | admin | 更换集合管理员 {user_id} |

标签定义支持类型：`string / int / float / bool / date / enum / array / object`（object 可递归嵌套子字段）。

### 数据项与刮削（操作工/集合管理员，逐集合判定）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | /collections/:id/items | 添加 {path, tags?, auto_scrape?}；默认刮削添加，false 为直接添加；均校验路径存在性 |
| GET | /collections/:id/items | 按标签查询，支持 `tag=值`、`tag.gt/gte/lt/lte`、`tag.in=a,b`、`tag.exists=true`，分页 |
| GET | /items/:itemId | 详情 |
| PATCH | /items/:itemId | 修改 {path?, tags?}（标签全量替换；路径可改） |
| DELETE | /items/:itemId | 删除（仅元数据，级联删任务） |
| POST | /items/:itemId/scrape | 手动触发刮削（不支持定时，Q2） |
| GET | /items/:itemId/scrape-tasks | 刮削历史 |
| GET | /scrape-tasks/:taskId | 任务详情 |

## 刮削子系统

- 独立进程 `cmd/scraper`：Worker 池（`scrape.worker_count`，默认 8）轮询 MongoDB 任务队列；
- 脚本约定（Q3）：`<script_path> <data_path>` 单入参，stdout 输出 JSON 标签对象即可；
  成功判据 = 可解析 JSON 且通过集合标签定义校验（必填/类型/enum/array/object）；
- 退出码仅记录诊断；超时（默认 1800s，可配）与输出大小（默认 1MB）受限；
- 僵死任务回收：running 超过 `scrape.reclaim_seconds`（默认 3600s）的任务会被重新领取（进程崩溃自愈）；
- 每个集合有且仅有一个刮削脚本（路径注册，由集合管理员维护），任务记录脚本路径快照。

## 测试

```powershell
# 单元测试（无需外部依赖）
$env:TEMP='<workspace>\.gotmp'; $env:TMP='<workspace>\.gotmp'  # 沙箱环境下重定向临时目录
go test ./...

# 端到端冒烟测试（需先启动 server 与 scraper，MongoDB 就绪）
# 默认 http://localhost:18080；内部使用 DATACENTER_DATABASE_NAME=datacenter_testN 等环境变量隔离
& .\scripts\e2e.ps1 -BaseUrl http://localhost:18080
```

单元测试覆盖：标签定义校验（重复/保留字段/类型约束）、标签值校验与规范化（必填、类型转换、
enum/array/嵌套 object、json.Number）、查询过滤器构建（等值/范围/in/exists/未知标签/非法操作符）、
路径校验（根目录逃逸/存在性/相对路径）、标签定义增量合并、刮削脚本路径校验、角色满足矩阵、
集合成员角色查询。

端到端覆盖：RBAC 登录/建用户/权限隔离、集合与标签定义（含 enum/嵌套 object）、直接添加/刮削添加、
标签校验入库、等值/范围/in/exists 查询、操作工不可改标签定义、路径修改、手动重刮与混合标签、
手动标签始终优先（含 mixed 二次重刮）、删除仅元数据、20 项并发刮削。当前 37 项断言全部通过（支持同库重复执行）。

## 实现设计决策（详见 需求分解.md §12.2）

- `PATCH /items/:id` 的 `tags` 为**全量覆盖**（须满足必填与类型校验），同时写入 `manual_tags` 并将 `tag_source` 置回 `manual`；未传 `tags` 时仅改路径；
- **手动标签始终优先**：手动标签持久化于 `manual_tags`，刮削结果仅补充手动未产出的标签，冲突键保留手动值（存在手动标签时 `tag_source=mixed`）；无论重刮多少次，手动标签均不被刮削覆盖；
- 手动场景**拒绝**未知标签；刮削场景**忽略**未知标签（脚本输出任意 JSON 的容错）；
- 删除数据项级联删除其刮削任务（仅元数据）；`running` 超过回收阈值（默认 3600s）的任务由 Worker 重新领取（崩溃自愈）。

## 注意事项（详见 需求分解.md §12.3）

- 生产环境必须覆盖 `jwt.secret` / bootstrap 管理员密码 / `data.root_dir` / scrape 超时等配置，禁止使用默认值；
- 刮削协议目前仅验证了可执行文件输出 JSON，**NFS 上的真实 shell/python 脚本需在目标集群实测**；
- 冒烟级验证（20 并发）已通过；**正式压测待补充**；
- 开发与冒烟测试在 Windows 完成，生产为 Linux + NFS 语义（可执行位、路径分隔符），部署前需回归；
- 标签数量无上限（Q11），标签定义较多时 `tags.$**` 通配符索引的查询性能需在真实数据量（单集合约 10 万项）下评估。

本地演示刮削脚本：`scripts/scrape_demo`（固定输出一组 JSON 标签，用于冒烟测试）。

## 项目结构

```
├── cmd/
│   ├── server/            # API 服务入口
│   ├── scraper/           # 刮削子系统入口（独立进程）
│   └── seeder/            # 演示数据灌入工具（model/case/layout/layer + 关系 + 文件树）
├── internal/
│   ├── config/            # 配置加载（yaml + 环境变量）
│   ├── logger/            # zap 日志
│   ├── database/          # MongoDB 连接、索引、种子数据
│   ├── model/             # 领域模型（用户/集合/数据项/任务/审计）
│   ├── store/             # 数据访问层
│   ├── service/           # 业务逻辑（权限逐集合判定、标签校验、查询构建）
│   ├── scrape/            # 刮削 Worker 池（任务领取/执行/回收）
│   ├── web/               # 前端静态资源（go:embed 内嵌）
│   ├── handler/           # HTTP 处理器
│   ├── middleware/        # JWT 鉴权、请求日志、全局角色校验
│   ├── errno/             # 业务错误码
│   ├── response/          # 统一响应
│   └── router/            # 路由注册与依赖装配
├── scripts/               # goenv.ps1（缓存重定向）、e2e.ps1（冒烟测试）、scrape_demo、
│                          #   mongo-up.ps1（MongoDB 容器+数据卷）、seed.ps1（灌数）、
│                          #   start-server.ps1 / start-scraper.ps1（启动）、verify.ps1（验证）
├── config/config.yaml     # 配置示例
└── 需求分解.md            # 需求文档
```

## 开发迭代进度（对应需求分解 §11）

- [x] P0 地基：骨架、配置、MongoDB、日志、统一响应/错误码、JWT 中间件
- [x] P1 RBAC 与用户：用户模型、登录、admin 用户管理、鉴权中间件（公共全局 + 集合逐集合）
- [x] P2 集合与标签定义：集合 CRUD、标签定义（string/int/float/bool/date/enum/array/object 递归校验）、成员授权
- [x] P3 数据项 CRUD + 查询：直接添加/刮削添加（路径存在性校验）、增删改查、标签校验、分页与多操作符查询
- [x] P4 刮削系统：脚本注册、任务模型、Worker 池并发、JSON 结果解析入库、手动重试、僵死任务回收
- [x] P5 收尾：审计日志、任务恢复（reclaim）、索引、冒烟级并发验证、文档补齐
