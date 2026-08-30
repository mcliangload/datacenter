# seed.ps1 — 灌入演示数据：model/case/layout/layer 四集合 + 真实文件树 + 关联关系
# Usage: .\scripts\seed.ps1 [-Models 3000] [-Layouts 2000] [-Layers 0] [-Cases 3000] [-Workers 16] [-NoFiles]
param(
    [int]$Models = 3000,
    [int]$Layouts = 2000,
    [int]$Layers = 0,
    [int]$Cases = 3000,
    [int]$Workers = 16,
    [switch]$NoFiles
)
$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
. (Join-Path $PSScriptRoot 'goenv.ps1')
$env:DATACENTER_DATABASE_URI = 'mongodb://localhost:27017'
$env:DATACENTER_DATABASE_NAME = 'datacenter'
$env:DATACENTER_DATA_ROOT_DIR = (Join-Path $root '.nfsdata')
Set-Location $root

$seedArgs = @(
    '-config', (Join-Path $root 'config\config.yaml'),
    '-models', "$Models", '-layouts', "$Layouts",
    '-layers', "$Layers", '-cases', "$Cases", '-workers', "$Workers"
)
if ($NoFiles) { $seedArgs += '-no-files' }
go run ./cmd/seeder @seedArgs
