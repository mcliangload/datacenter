# verify.ps1 — 验证服务与种子数据：登录 → 概览统计 → DQL 查询（含 parent/ancestor 关系限定）
# Usage: .\scripts\verify.ps1 [-Port 18080]
param([int]$Port = 18080)
$ErrorActionPreference = 'Stop'
$base = "http://127.0.0.1:$Port/api/v1"

function Req($method, $path, $body, $token) {
    $headers = @{}
    if ($token) { $headers["Authorization"] = "Bearer $token" }
    $params = @{ Method = $method; Uri = "$base$path"; Headers = $headers; ContentType = 'application/json' }
    if ($null -ne $body) { $params["Body"] = ($body | ConvertTo-Json -Depth 10 -Compress) }
    return Invoke-RestMethod @params
}

$login = Req 'POST' '/auth/login' @{ username = 'admin'; password = 'admin123' } $null
$token = $login.data.token
Write-Host "login ok: $($login.data.user.username) role=$($login.data.user.role)"

$ov = (Req 'GET' '/stats/overview' $null $token).data
Write-Host ("overview: collections={0} items={1}" -f $ov.collections, $ov.items)
Write-Host ("relations: total={0} parent_child={1} reference={2} call={3}" -f $ov.relations.total, $ov.relations.parent_child, $ov.relations.reference, $ov.relations.call)
Write-Host ("tasks: pending={0} running={1} success={2} failed={3}" -f $ov.tasks.pending, $ov.tasks.running, $ov.tasks.success, $ov.tasks.failed)

$queries = @(
    'collection = "model" AND status = "released" AND na = 1.35',
    'collection = "layout" AND drc_status = "clean" AND node = "7"',
    'collection = "layer" AND opc_treatment = "ilt" AND epe_violations = 0',
    'collection = "case" AND status = "passed" AND meef >= 3',
    'collection IN ("model","layout") AND node LIKE "1"'
)
foreach ($q in $queries) {
    $r = (Req 'POST' '/dql/query' @{ dql = $q; page = 1; page_size = 5 } $token).data
    Write-Host ("DQL [{0}] total={1}" -f $q, $r.total)
}

# 关系限定：取一个版图，查其子树（ancestor），再取一个图层查其父（parent）
$lay = (Req 'POST' '/dql/query' @{ dql = 'collection = "layout"'; page = 1; page_size = 1 } $token).data
$layId = $lay.items[0].id
$layName = $lay.items[0].tags.name
Write-Host "layout picked: $layName id=$layId"

$sub = (Req 'POST' '/dql/query' @{ dql = "collection = `"layer`" AND ancestor = `"$layId`""; page = 1; page_size = 1 } $token).data
Write-Host "DQL [ancestor = $layId] total=$($sub.total) (该版图图层数)"

$lyr = (Req 'POST' '/dql/query' @{ dql = "collection = `"layer`" AND ancestor = `"$layId`""; page = 1; page_size = 1 } $token).data
$lyrId = $lyr.items[0].id
$par = (Req 'POST' '/dql/query' @{ dql = "parent = `"$lyrId`""; page = 1; page_size = 1 } $token).data
if ($par.items.Count -gt 0) {
    Write-Host "DQL [parent = $lyrId] -> $($par.items[0].tags.name) (id=$($par.items[0].id))"
} else {
    Write-Host "DQL [parent = $lyrId] -> (无结果)"
}
Write-Host "verify done"
