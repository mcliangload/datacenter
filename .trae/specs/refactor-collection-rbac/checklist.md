# 验证清单

## 编译验证
- [x] `go build ./cmd/server/...` 编译通过
- [x] `go vet ./internal/... ./pkg/... ./cmd/...` 通过
- [x] `npm run build` 前端编译通过

## 功能验证
- [x] 创建集合时自动在 `rbac.permissions` 创建5条权限记录（read/write/delete/admin/field:admin）
- [x] 创建集合时自动创建3个角色（Owner/Operator/Tourist），角色名格式正确
- [x] Owner 角色包含所有5个权限（admin + read + write + delete + field:admin）
- [x] Operator 角色包含3个权限（read + write + delete）
- [x] Tourist 角色包含1个权限（read）
- [x] 创建集合时自动将 Owner 角色分配给创建者
- [x] `InitDefaultData()` 在空数据库时自动创建 admin 用户和 root 角色
- [x] `InitDefaultData()` 在已有数据时不重复创建

## 权限中间件验证
- [x] `system:admin` 超级管理员通过所有权限检查（admin登录后可创建集合/字段/数据）
- [x] Owner 可创建/查询自定义字段
- [x] 从URL参数提取module的中间件正常工作（`GET /api/business/module/:module`）
- [x] 从请求体提取module的中间件正常工作（`POST /api/business`）
- [x] 通过字段ID查找module的中间件正常工作

## API 接口验证
- [x] `GET /api/collections/:module/roles` 返回3个角色
- [x] `POST /api/collections/:module/roles/assign` 分配角色正常工作
- [x] 创建集合API返回201
- [x] 查询业务数据API正确的权限校验

## 测试用例验证
- [x] `test/collection_roles_test.go` 编译通过（需要MongoDB环境运行）
- [x] E2E烟雾测试全部通过（admin登录→创建集合→创建角色→创建字段→创建数据→分配角色）
