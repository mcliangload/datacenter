package web

import "embed"

// FS 前端静态资源（go:embed 打包进二进制，单文件部署）
//
//go:embed static
var FS embed.FS
