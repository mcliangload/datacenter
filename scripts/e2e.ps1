# 端到端冒烟测试：admin 建用户/集合 → 集合管理员配脚本/授权 → 操作工添加数据项（直接+刮削）→
# 验证标签入库、按标签查询、权限隔离、手动重刮。
# 用法：pwsh -File scripts/e2e.ps1 [-BaseUrl http://localhost:18080]
param(
    [string]$BaseUrl = "http://localhost:18080"
)

$ErrorActionPreference = "Stop"
$json = 'application/json'
$passed = 0
$failed = 0

function Check($name, $cond, $detail) {
    if ($cond) { $script:passed++; Write-Host "[PASS] $name" -ForegroundColor Green }
    else { $script:failed++; Write-Host "[FAIL] $name => $detail" -ForegroundColor Red }
}

function Req($method, $path, $body, $token) {
    $headers = @{}
    if ($token) { $headers["Authorization"] = "Bearer $token" }
    $params = @{ Method = $method; Uri = "$BaseUrl$path"; Headers = $headers }
    if ($null -ne $body) {
        $params.ContentType = $json
        # 显式 UTF-8 编码，避免 PS 5.1 默认编码破坏中文 JSON
        $params.Body = [System.Text.Encoding]::UTF8.GetBytes(($body | ConvertTo-Json -Depth 10))
    }
    return Invoke-RestMethod @params
}

Write-Host "== 1. 登录 admin =="
$admin = Req POST "/api/v1/auth/login" @{ username = "admin"; password = "admin123" } $null
$adminToken = $admin.data.token
Check "admin 登录" ($null -ne $adminToken) $admin

# 每次运行使用唯一名称，支持同一数据库重复执行
$suffix = Get-Date -Format "HHmmss"

Write-Host "== 2. 用户管理 =="
$op1Name = "op1_$suffix"; $op2Name = "op2_$suffix"
$op1 = Req POST "/api/v1/users" @{ username = $op1Name; password = "op1pass1"; role = "user" } $adminToken
Check "创建操作工 op1" ($op1.data.username -eq $op1Name) $op1
$op2 = Req POST "/api/v1/users" @{ username = $op2Name; password = "op2pass2"; role = "user" } $adminToken
Check "创建操作工 op2" ($op2.data.username -eq $op2Name) $op2

Write-Host "== 3. 创建集合（admin，指定初始集合管理员=admin）=="
$col = Req POST "/api/v1/collections" @{
    name        = "光刻模型库_$suffix"
    description = "计算光刻 model 数据"
    initial_admin_id = $admin.data.user.id
    tag_schema = @(
        @{ name = "model_name"; type = "string"; required = $true },
        @{ name = "version";   type = "string"; required = $true },
        @{ name = "age";       type = "int";    required = $false },
        @{ name = "stage";     type = "enum";   required = $true; enum_values = @("dev", "test", "prod") },
        @{ name = "config";    type = "object"; required = $false; fields = @(
            @{ name = "version"; type = "string"; required = $true },
            @{ name = "accuracy"; type = "float"; required = $false }
        ) }
    )
} $adminToken
$colId = $col.data.id
Check "创建集合" ($null -ne $colId) $col

Write-Host "== 4. 集合管理员配置刮削脚本并授权操作工 =="
$script = (Resolve-Path (Join-Path $PSScriptRoot "..\bin\scrape_demo.exe")).Path
$col2 = Req PUT "/api/v1/collections/$colId/script" @{ path = $script } $adminToken
Check "注册刮削脚本" ($col2.data.scrape_script.path -eq $script) $col2
$col3 = Req POST "/api/v1/collections/$colId/members" @{ user_id = $op1.data.id } $adminToken
Check "授权 op1 为操作工" ($col3.data.members.Count -eq 2) $col3

Write-Host "== 5. 权限隔离：非成员 op2 访问集合应 403 =="
$op2Login = Req POST "/api/v1/auth/login" @{ username = $op2Name; password = "op2pass2" } $null
$op2Token = $op2Login.data.token
$denied = $false
try { Req GET "/api/v1/collections/$colId" $null $op2Token } catch { $denied = $true }
Check "op2 非成员访问集合 403" $denied "op2 不应能访问"

Write-Host "== 6. 操作工登录 =="
$op1Login = Req POST "/api/v1/auth/login" @{ username = $op1Name; password = "op1pass1" } $null
$op1Token = $op1Login.data.token
Check "op1 登录" ($null -ne $op1Token) $op1Login

Write-Host "== 7. 直接添加（auto_scrape=false，带手动标签）=="
$root = (Resolve-Path (Join-Path $PSScriptRoot "..\.nfsdata")).Path
$dir1 = Join-Path $root "opc_model_v1"
$item1 = Req POST "/api/v1/collections/$colId/items" @{
    path = $dir1; auto_scrape = $false
    tags = @{ model_name = "manual-model"; version = "0.9"; stage = "dev" }
} $op1Token
$item1Id = $item1.data.item.id
Check "直接添加数据项" ($null -ne $item1Id) $item1
Check "直接添加不触发刮削" ($item1.data.item.scrape_status -eq "none") $item1

Write-Host "== 8. 刮削添加（auto_scrape=true，等待 worker 处理）=="
$dir2 = Join-Path $root "opc_model_v2"
$item2 = Req POST "/api/v1/collections/$colId/items" @{ path = $dir2; auto_scrape = $true } $op1Token
$item2Id = $item2.data.item.id
Check "刮削添加创建任务" ($item2.data.task.status -eq "pending") $item2

$taskResult = $null
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep -Seconds 1
    $detail = Req GET "/api/v1/items/$item2Id" $null $op1Token
    if ($detail.data.scrape_status -eq "success") { $taskResult = $detail; break }
}
Check "刮削任务成功并写入标签" ($null -ne $taskResult) "scrape_status=$($detail.data.scrape_status)"
if ($taskResult) {
    Check "标签 model_name=demo-model" ($taskResult.data.tags.model_name -eq "demo-model") $taskResult.data.tags
    Check "标签 age=3 (int)" ($taskResult.data.tags.age -eq 3) $taskResult.data.tags
    Check "标签 config 嵌套对象" ($taskResult.data.tags.config.accuracy -eq 0.98) $taskResult.data.tags
    Check "标签来源=scrape" ($taskResult.data.tag_source -eq "scrape") $taskResult.data
}

Write-Host "== 9. 按标签查询 =="
$q1 = Req GET "/api/v1/collections/$colId/items?model_name=demo-model" $null $op1Token
Check "等值查询 model_name=demo-model" ($q1.data.total -eq 1) $q1
$q2 = Req GET "/api/v1/collections/$colId/items?age.gte=3" $null $op1Token
Check "范围查询 age.gte=3" ($q2.data.total -ge 1) $q2
$q3 = Req GET "/api/v1/collections/$colId/items?stage.in=test,prod" $null $op1Token
Check "in 查询 stage.in=test,prod" ($q3.data.total -ge 1) $q3
$q4 = Req GET "/api/v1/collections/$colId/items?config.exists=true" $null $op1Token
Check "exists 查询 config.exists=true" ($q4.data.total -ge 1) $q4

Write-Host "== 10. 操作工不可修改标签定义（403）=="
$forbidden = $false
try { Req PUT "/api/v1/collections/$colId/tags" @{ tags = @() } $op1Token } catch { $forbidden = $true }
Check "op1 改标签定义被拒" $forbidden "操作工不应能修改标签定义"

Write-Host "== 11. 修改数据项（改路径）=="
$dir3 = Join-Path $root "opc_model_v3"
$itemUpd = Req PATCH "/api/v1/items/$item1Id" @{ path = $dir3 } $op1Token
Check "修改数据项路径" ($itemUpd.data.path -eq $dir3) $itemUpd

Write-Host "== 12. 手动重刮 =="
$task2 = Req POST "/api/v1/items/$item1Id/scrape" $null $op1Token
Check "手动触发重刮" ($task2.data.status -eq "pending") $task2
$manualScraped = $null
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep -Seconds 1
    $detail = Req GET "/api/v1/items/$item1Id" $null $op1Token
    if ($detail.data.scrape_status -eq "success") { $manualScraped = $detail; break }
}
Check "手动重刮成功" ($null -ne $manualScraped) "scrape_status=$($detail.data.scrape_status)"
if ($manualScraped) {
    Check "手动标签与刮削标签合并(mixed)" ($manualScraped.data.tag_source -eq "mixed") $manualScraped.data
    # 手动标签始终优先：冲突键保留手动值，刮削仅补充手动未产出的标签
    Check "手动优先：model_name 保留 manual-model" ($manualScraped.data.tags.model_name -eq "manual-model") $manualScraped.data.tags
    Check "手动优先：version 保留 0.9" ($manualScraped.data.tags.version -eq "0.9") $manualScraped.data.tags
    Check "手动优先：stage 保留 dev" ($manualScraped.data.tags.stage -eq "dev") $manualScraped.data.tags
    Check "刮削补充：age=3（手动未产出）" ($manualScraped.data.tags.age -eq 3) $manualScraped.data.tags
    Check "刮削补充：config 嵌套对象" ($manualScraped.data.tags.config.accuracy -eq 0.98) $manualScraped.data.tags
    $tasks = Req GET "/api/v1/items/$item1Id/scrape-tasks" $null $op1Token
    Check "刮削历史 1 条（直接添加无自动任务 + 1 次手动重刮）" ($tasks.data.total -eq 1) $tasks

    # 第二次重刮（mixed 状态）：manual_tags 持久化，手动标签仍不被覆盖
    $task3 = Req POST "/api/v1/items/$item1Id/scrape" $null $op1Token
    Check "mixed 状态再次重刮触发" ($task3.data.status -eq "pending") $task3
    $mixedScraped = $null
    for ($i = 0; $i -lt 30; $i++) {
        Start-Sleep -Seconds 1
        $detail = Req GET "/api/v1/items/$item1Id" $null $op1Token
        if ($detail.data.scrape_status -eq "success") { $mixedScraped = $detail; break }
    }
    Check "mixed 状态再次重刮成功" ($null -ne $mixedScraped) "scrape_status=$($detail.data.scrape_status)"
    if ($mixedScraped) {
        Check "再次重刮后手动标签仍优先" ($mixedScraped.data.tags.model_name -eq "manual-model" -and $mixedScraped.data.tags.version -eq "0.9" -and $mixedScraped.data.tags.stage -eq "dev") $mixedScraped.data.tags
    }
}

Write-Host "== 13. 删除数据项（仅元数据）=="
$del = Req DELETE "/api/v1/items/$item2Id" $null $op1Token
Check "删除数据项" ($del.code -eq 0) $del
$notFound = $false
try { Req GET "/api/v1/items/$item2Id" $null $op1Token } catch { $notFound = $true }
Check "删除后查询 404" $notFound "删除后不应再查到"

Write-Host "== 14. 并发刮削（20 个数据项同时入队）=="
$concurrentIds = @()
for ($i = 0; $i -lt 20; $i++) {
    $d = Join-Path $root "batch_$i"
    New-Item -ItemType Directory -Force -Path $d | Out-Null
    $it = Req POST "/api/v1/collections/$colId/items" @{ path = $d; auto_scrape = $true } $op1Token
    $concurrentIds += $it.data.item.id
}
$okCount = 0
for ($i = 0; $i -lt 40; $i++) {
    Start-Sleep -Seconds 1
    $okCount = 0
    foreach ($id in $concurrentIds) {
        $d = Req GET "/api/v1/items/$id" $null $op1Token
        if ($d.data.scrape_status -eq "success") { $okCount++ }
    }
    if ($okCount -eq $concurrentIds.Count) { break }
}
Check "并发刮削全部成功（$($concurrentIds.Count) 个）" ($okCount -eq $concurrentIds.Count) "ok=$okCount/20"

Write-Host ""
Write-Host "======== 结果: PASS=$passed FAIL=$failed ========" -ForegroundColor Cyan
if ($failed -gt 0) { exit 1 }
