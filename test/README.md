# 测试数据生成说明

## 脚本功能

本脚本 `createdata.go` 用于创建完整的测试数据，包括：

1. **用户管理**
   - 4个用户，密码均为 `liangminchuan`
   - 超级管理员、数据管理员、普通用户等角色

2. **权限管理**
   - 6个系统权限
   - 3个角色（超级管理员、数据管理员、普通用户）

3. **模块管理**
   - 5个数据模块：movie、music、book、game、product

4. **刮削任务**
   - 每个模块20个刮削任务（共100个）
   - 所有任务状态为成功

5. **业务数据**
   - 每个刮削任务对应一条业务数据
   - 包含完整的刮削信息

## 执行方法

1. **启动MongoDB服务**
   ```bash
   # Windows
   net start MongoDB
   
   # Linux/macOS
   sudo systemctl start mongod
   ```

2. **运行脚本**
   ```bash
   cd d:/gocode/datacenter/test
   go run createdata.go
   ```

3. **验证数据**
   - 用户登录：使用任意用户名和密码 `liangminchuan`
   - 查看刮削任务：共100个任务，分布在5个模块
   - 查看业务数据：每个模块20条数据

## 脚本结构

- `createPermissions()`: 创建系统权限
- `createRoles()`: 创建角色并分配权限
- `createUsers()`: 创建用户并分配角色
- `createCollections()`: 创建数据模块集合
- `createScrapeData()`: 创建刮削任务和业务数据

所有数据都按照设计文档的结构创建，确保系统能够正常运行和测试。