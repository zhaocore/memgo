#!/usr/bin/env bash
# 首次引导: setup-status → register 首个 admin → 建 API key → 打印凭据 (明文仅此一次)。
# 对齐上游 server/Makefile seed 流程。用法: API=http://localhost:8888 ./seed.sh
set -euo pipefail
API="${API:-http://localhost:8888}"
NAME="${NAME:-Admin}"
EMAIL="${EMAIL:-admin@example.com}"
PASSWORD="${PASSWORD:-memgo-admin-123}"

STATUS=$(curl -sf "$API/auth/setup-status")
NEEDS=$(printf '%s' "$STATUS" | jq -r .needsSetup)
if [ "$NEEDS" != "true" ]; then
  echo "已初始化 (needsSetup=false), 跳过注册。"
  exit 0
fi
TOKEN=$(curl -sf -X POST "$API/auth/register" -H 'Content-Type: application/json' \
  -d "{\"name\":\"$NAME\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" | jq -r .access_token)
KEY=$(curl -sf -X POST "$API/api-keys" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"label":"seed-key"}')
echo "seed 完成:"
echo "  email:    $EMAIL"
echo "  password: $PASSWORD"
echo "  api-key:  $(printf '%s' "$KEY" | jq -r .key)"
