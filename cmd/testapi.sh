#!/bin/bash

# 登录获取token
echo "正在登录获取token..."
token=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"liangminchuan"}' \
  | jq -r '.token')

if [ -z "$token" ]; then
  echo "获取token失败"
  exit 1
fi

echo "获取token成功: $token"

# 测试查询业务数据
echo "\n测试查询业务数据..."
curl -s -X GET "http://localhost:8080/api/business/module/test_upload?page=1&pageSize=5" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $token" \
  | jq .

# 测试查询所有数据
echo "\n测试查询所有数据..."
curl -s -X GET "http://localhost:8080/api/business/module/test_upload" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $token" \
  | jq .
