#!/usr/bin/env bash
# 契约: setup —— bootstrap 哨兵(空库) + 注册首个 admin + 建 key。
# 合同: doc-02 §4/§5, doc-03 §3.3 (bootstrap UUID(int=0) 哨兵)。

suite() {
  # 全新库状态
  http GET /auth/setup-status
  check_status "setup-status" 200
  check_json "needsSetup=true" '.needsSetup' 'true'
  g "setup-status"

  # bootstrap 哨兵: 空库时 ADMIN_API_KEY 经 require_admin 拿到 UUID(int=0) 哨兵 (全新部署可自助管理)
  # 注: require_auth (如 POST /api-keys) 在空库无用户可落 → 401, bootstrap 只覆盖 require_admin 面
  http GET '/requests?limit=1' "X-API-Key: $ADMIN_API_KEY"
  check_status "bootstrap: 空库 ADMIN_API_KEY 过 require_admin" 200
  g "bootstrap-requests-empty-db"

  # 注册首个 admin (唯一入口; 之后 registration closed)
  http POST /auth/register '{"name":"Contract Admin","email":"contract@example.com","password":"contract-pass-123"}'
  check_status "register first admin" 200
  check_json "register token_type" '.token_type' 'bearer'
  g "register"
  ACCESS_TOKEN=$(printf '%s' "$R_BODY" | jq -r .access_token)
  REFRESH_TOKEN=$(printf '%s' "$R_BODY" | jq -r .refresh_token)

  # login (与 register 独立桶)
  http POST /auth/login '{"email":"contract@example.com","password":"contract-pass-123"}'
  check_status "login" 200
  ACCESS_TOKEN=$(printf '%s' "$R_BODY" | jq -r .access_token)
  REFRESH_TOKEN=$(printf '%s' "$R_BODY" | jq -r .refresh_token)

  # 用户 API key (明文仅此一次)
  http POST /api-keys '{"label":"contract-user-key"}' "Authorization: Bearer $ACCESS_TOKEN"
  check_status "create user api key" 201
  check_json "key 前缀 m0sk_" '.key_prefix | startswith("m0sk_")' 'true'
  g "user-key-created"
  USER_API_KEY=$(printf '%s' "$R_BODY" | jq -r .key)
  USER_KEY_ID=$(printf '%s' "$R_BODY" | jq -r .id)

  # GET /api-keys: bootstrap key 归属哨兵 UUID(0) ≠ 当前用户 → 仅见自己的
  http GET /api-keys "Authorization: Bearer $ACCESS_TOKEN"
  check_status "list own api keys" 200
  check_json "仅见自身 key (len=1)" 'length' '1'
  g "api-keys-list"
}
