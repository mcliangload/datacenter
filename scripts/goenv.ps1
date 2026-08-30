# 设置 Go 工具链环境（工作区内缓存 + 镜像代理），用法：
#   . .\scripts\goenv.ps1; go build ./...
# 说明：
#   - 沙箱环境下用户目录（AppData/go 等）不可写，将缓存全部重定向到工作区
#   - 默认 proxy.golang.org / sum.golang.org 网络不可达，使用 goproxy.cn 并关闭 sumdb
$root = Split-Path $PSScriptRoot -Parent

$env:GOTELEMETRY = 'off'
$env:GOENV       = Join-Path $root '.goenv'
$env:GOMODCACHE  = Join-Path $root '.gomodcache'
$env:GOCACHE     = Join-Path $root '.gocache'
$env:GOPATH      = Join-Path $root '.gopath'
$env:GOTMPDIR    = Join-Path $root '.gotmp'
$env:GOPROXY     = 'https://goproxy.cn,direct'
$env:GOSUMDB     = 'off'

# 确保缓存目录存在（GOTMPDIR 必须已存在）
foreach ($dir in @($env:GOENV, $env:GOMODCACHE, $env:GOCACHE, $env:GOPATH, $env:GOTMPDIR)) {
    if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
}
