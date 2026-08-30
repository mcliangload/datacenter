# start-scraper.ps1 — 启动刮削子系统（独立进程，与 server 共享 MongoDB 任务队列）
# Usage: .\scripts\start-scraper.ps1 [-DbName datacenter]
param(
    [string]$DbName = 'datacenter'
)
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
. (Join-Path $PSScriptRoot 'goenv.ps1')
$env:DATACENTER_DATABASE_URI = 'mongodb://localhost:27017'
$env:DATACENTER_DATABASE_NAME = $DbName
$env:DATACENTER_DATA_ROOT_DIR = (Join-Path $root '.nfsdata')
Set-Location $root
Write-Host "[scraper] starting, db=$DbName (Ctrl+C to stop)"
go run ./cmd/scraper -config config\config.yaml
