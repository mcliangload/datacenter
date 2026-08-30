# mongo-up.ps1 — 重建本地 MongoDB 容器并挂载命名数据卷（容器重建不丢数据）
# Usage: .\scripts\mongo-up.ps1
$PSNativeCommandUseErrorActionPreference = $false
$ErrorActionPreference = 'Stop'
$name = 'datacenter-mongo'
$vol  = 'datacenter-mongo-data'

# 移除旧容器（若存在）
$existing = docker ps -a --filter "name=^/${name}$" --format '{{.Names}}'
if ($existing) {
    docker rm -f $name | Out-Null
    Write-Host "[docker] removed old container: $name"
} else {
    Write-Host "[docker] no old container: $name"
}

# 创建命名数据卷（若不存在）
$vols = docker volume ls -q
if ($vols -notcontains $vol) {
    docker volume create $vol | Out-Null
    Write-Host "[docker] volume created: $vol"
} else {
    Write-Host "[docker] volume exists: $vol"
}

# 启动容器（mongo:7，映射 27017，挂载数据卷到 /data/db）
docker run -d --name $name -p 27017:27017 -v "${vol}:/data/db" mongo:7 | Out-Null
if ($LASTEXITCODE -ne 0) { throw "docker run failed" }
Write-Host "[docker] container started: $name (mongo:7, 127.0.0.1:27017, volume=$vol)"

# 等待 Mongo 就绪
for ($i = 0; $i -lt 30; $i++) {
    docker exec $name mongosh --quiet --eval 'db.runCommand({ping:1}).ok' *> $null
    if ($LASTEXITCODE -eq 0) { Write-Host "[docker] mongo ready (ping ok)"; exit 0 }
    Start-Sleep -Seconds 1
}
throw "mongo did not become ready in 30s"
