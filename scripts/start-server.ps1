# start-server.ps1 — 启动 API 服务 + 内嵌前端（默认 :18080，数据库 datacenter）
# Usage: .\scripts\start-server.ps1 [-Port 18080] [-DbName datacenter]
param(
    [int]$Port = 18080,
    [string]$DbName = 'datacenter'
)
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
. (Join-Path $PSScriptRoot 'goenv.ps1')
$env:DATACENTER_SERVER_PORT = "$Port"
$env:DATACENTER_DATABASE_URI = 'mongodb://localhost:27017'
$env:DATACENTER_DATABASE_NAME = $DbName
$env:DATACENTER_DATA_ROOT_DIR = (Join-Path $root '.nfsdata')
$env:DATACENTER_SCRAPE_POLL_INTERVAL_MS = '1000'
Set-Location $root
Write-Host "[server] starting on :$Port db=$DbName (Ctrl+C to stop)"
go run ./cmd/server -config config\config.yaml
