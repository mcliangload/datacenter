// Package version 应用版本号。
// 默认值为源码常量；构建时可用 -ldflags 注入覆盖：
//
//	go build -ldflags "-X datacenter/internal/version.Version=v1.2.3" ./cmd/server
//
// 版本会出现在 GET /healthz 与前端侧边栏底部，便于确认部署实例的代码版本。
package version

// Version 当前版本号（与 需求分解.md 变更记录保持一致）
var Version = "v0.14"
