#!/usr/bin/env bash
# 契约: 请求日志 (doc-02 §7)。只记 api_key 类; 跳过清单生效; 限流边界 422。

suite() {
  local AH_UKEY="X-API-Key: $USER_API_KEY"
  local AH_AKEY="X-API-Key: $ADMIN_API_KEY"

  http GET '/requests?limit=200' "$AH_AKEY"
  check_status "requests 列表 (admin key)" 200
  check_json "仅 api_key 类" '[.[] | select(.auth_type != "api_key" and .auth_type != "admin_api_key")] | length' '0'
  check_json "无 JWT 会话记录" '[.[] | select(.auth_type == "bearer")] | length' '0'
  check_json "跳过清单: /docs" '[.[] | select(.path == "/docs")] | length' '0'
  check_json "跳过清单: /openapi.json" '[.[] | select(.path == "/openapi.json")] | length' '0'
  check_json "跳过清单: /api/health" '[.[] | select(.path == "/api/health")] | length' '0'
  check_json "跳过 /requests 前缀" '[.[] | select(.path | startswith("/requests"))] | length' '0'
  check_json "含 POST /memories" '[.[] | select(.method == "POST" and .path == "/memories")] | length > 0' 'true'
  check_json "429 请求也被记录 (auth_type=none 不应出现)" '[.[] | select(.auth_type == "none")] | length' '0'
  # 单条形状快照 (最近一条)
  R_BODY=$(printf '%s' "$R_BODY" | jq -c '.[0]')
  g "request-entry-shape"

  http GET '/requests?limit=1' "$AH_AKEY"
  check_status "limit=1" 200
  check_json "limit=1 生效" 'length' '1'

  http GET '/requests?limit=0' "$AH_AKEY"
  check_status "limit=0 → 422" 422
  g "422-limit-zero"

  http GET '/requests?limit=201' "$AH_AKEY"
  check_status "limit=201 → 422" 422

}
