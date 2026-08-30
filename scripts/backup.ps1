# backup.ps1 — MongoDB 与 NFS 数据目录备份（系统优化 3.2）
# 用法：.\scripts\backup.ps1 [-BackupDir D:\backups] [-Retain 7]
#   1) mongodump 当前数据库（默认 datacenter，可用 -DbName 指定）
#   2) 打包 NFS 数据目录（data.root_dir，默认 .nfsdata）
#   3) 轮转：仅保留最近 -Retain 份备份
# Linux 服务器可参考 部署指南.md §6 用 mongodump + tar + cron 实现同样效果。
param(
    [string]$BackupDir = '',
    [int]$Retain = 7,
    [string]$DbName = 'datacenter',
    [string]$MongoUri = 'mongodb://localhost:27017'
)
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent

if (-not $BackupDir) { $BackupDir = Join-Path $root 'backups' }
New-Item -ItemType Directory -Force -Path $BackupDir | Out-Null

$stamp = Get-Date -Format 'yyyyMMdd_HHmmss'
$target = Join-Path $BackupDir "backup_$stamp"
New-Item -ItemType Directory -Force -Path $target | Out-Null

# 1. mongodump（需要 mongodump 在 PATH；Docker 部署可用 docker exec datacenter-mongo mongodump ...）
Write-Host "[1/3] mongodump db=$DbName -> $target\dump"
$dumpOk = $true
mongodump --uri "$MongoUri/$DbName" --out (Join-Path $target 'dump') 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "  mongodump 不可用，尝试 docker exec ..."
    docker exec datacenter-mongo mongodump --db $DbName --out /tmp/dump *> $null
    if ($LASTEXITCODE -eq 0) {
        docker cp datacenter-mongo:/tmp/dump (Join-Path $target 'dump') *> $null
        docker exec datacenter-mongo rm -rf /tmp/dump *> $null
    } else { $dumpOk = $false; Write-Host "  [警告] mongodump 失败，跳过数据库备份" }
}
if ($dumpOk) { Write-Host "  mongodump 完成" }

# 2. 打包 NFS 数据目录
$dataRoot = $env:DATACENTER_DATA_ROOT_DIR
if (-not $dataRoot) { $dataRoot = Join-Path $root '.nfsdata' }
Write-Host "[2/3] 打包数据目录 $dataRoot"
if (Test-Path $dataRoot) {
    Compress-Archive -Path $dataRoot -DestinationPath (Join-Path $target 'nfsdata.zip') -CompressionLevel Fastest
} else {
    Write-Host "  [警告] 数据目录不存在：$dataRoot"
}

# 3. 轮转：保留最近 $Retain 份
Write-Host "[3/3] 轮转：保留最近 $Retain 份"
Get-ChildItem $BackupDir -Directory -Filter 'backup_*' |
    Sort-Object Name -Descending |
    Select-Object -Skip $Retain |
    ForEach-Object { Remove-Item $_.FullName -Recurse -Force; Write-Host "  已清理 $($_.Name)" }

Write-Host "备份完成: $target"
