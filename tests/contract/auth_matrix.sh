#!/usr/bin/env bash
# 契约: 鉴权矩阵逐格 (doc-02 §8)。列: Bearer JWT(admin) / X-API-Key(用户) / ADMIN_API_KEY / 无凭据。
# AUTH_DISABLED 列需第二实例 (CONTRACT_DISABLED_URL, AUTH_DISABLED=true, 共享同库)。

suite() {
  local AH_JWT="Authorization: Bearer $ACCESS_TOKEN"
  local AH_UKEY="X-API-Key: $USER_API_KEY"
  local AH_AKEY="X-API-Key: $ADMIN_API_KEY"

  # ---- 公开端点: 无凭据放行 ----
  http GET /auth/setup-status
  check_status "公开: setup-status 无凭据" 200

  # ---- verify_auth 行: /memories 带实体过滤 ----
  http GET "/memories?user_id=matrix" "$AH_JWT";    check_status "verify_auth: bearer" 200
  http GET "/memories?user_id=matrix" "$AH_UKEY";   check_status "verify_auth: user key" 200
  http GET "/memories?user_id=matrix" "$AH_AKEY";   check_status "verify_auth: admin key" 200
  http GET "/memories?user_id=matrix";              check_status "verify_auth: 无凭据" 401
  check_header "verify_auth: 401 带 WWW-Authenticate" "Bearer"

  # ---- verify_auth 行: POST /memories ----
  http POST /memories '{"messages":[{"role":"user","content":"x"}],"user_id":"matrix-user","infer":false}' "$AH_JWT"
  check_status "verify_auth POST: bearer" 200
  http POST /memories '{"messages":[{"role":"user","content":"x"}],"user_id":"matrix-user","infer":false}'
  check_status "verify_auth POST: 无凭据" 401

  # ---- GET /memories 无 id (模式 B, admin-only) ----
  http GET "/memories?top_k=1000" "$AH_JWT";   check_status "模式B: bearer admin" 200
  http GET "/memories?top_k=1000" "$AH_UKEY";  check_status "模式B: user key (admin 角色) → 200" 200
  http GET "/memories?top_k=1000" "$AH_AKEY";  check_status "模式B: admin key" 200
  http GET "/memories?top_k=1000";             check_status "模式B: 无凭据" 401

  # ---- require_admin 行: DELETE /memories (无害用户), GET /requests ----
  http DELETE "/memories?user_id=matrix-none" "$AH_JWT";  check_status "require_admin: bearer admin" 200
  http DELETE "/memories?user_id=matrix-none" "$AH_UKEY"; check_status "require_admin: user key (admin 角色) → 200" 200
  http DELETE "/memories?user_id=matrix-none" "$AH_AKEY"; check_status "require_admin: admin key" 200
  http DELETE "/memories?user_id=matrix-none";            check_status "require_admin: 无凭据" 401
  http GET "/requests?limit=10" "$AH_JWT";   check_status "requests: bearer admin" 200
  http GET "/requests?limit=10" "$AH_UKEY";  check_status "requests: user key (admin 角色) → 200" 200
  http GET "/requests?limit=10" "$AH_AKEY";  check_status "requests: admin key" 200
  http GET "/requests?limit=10";             check_status "requests: 无凭据" 401

  # ---- require_auth 行: /auth/me /api-keys /generate-instructions ----
  http GET /auth/me "$AH_JWT";   check_status "require_auth: bearer" 200
  http GET /auth/me "$AH_UKEY";  check_status "require_auth: user key" 200
  http GET /auth/me "$AH_AKEY";  check_status "require_auth: admin key 落首用户" 200
  http GET /auth/me;             check_status "require_auth: 无凭据" 401
  http GET /api-keys "$AH_JWT";  check_status "api-keys: bearer" 200
  http GET /api-keys "$AH_AKEY"; check_status "api-keys: admin key 落首用户" 200
  http GET /api-keys;            check_status "api-keys: 无凭据" 401
  http POST /generate-instructions '{"use_case":"contract matrix"}' "$AH_JWT"
  check_status "generate-instructions: bearer" 200
  http POST /generate-instructions '{"use_case":"contract matrix"}'
  check_status "generate-instructions: 无凭据" 401

  # ---- GET /entities (verify_auth) ----
  http GET /entities "$AH_JWT";  check_status "entities: bearer" 200
  http GET /entities "$AH_UKEY"; check_status "entities: user key" 200
  http GET /entities "$AH_AKEY"; check_status "entities: admin key" 200
  http GET /entities;            check_status "entities: 无凭据" 401

  # ---- require_admin 行: POST /reset (数据未写入, 无害) ----
  http POST /reset "" "$AH_JWT";  check_status "reset: bearer admin" 200
  http POST /reset "" "$AH_UKEY"; check_status "reset: user key (admin 角色) → 200" 200
  http POST /reset "" "$AH_AKEY"; check_status "reset: admin key" 200
  http POST /reset "";            check_status "reset: 无凭据" 401

  # ---- 公开: GET / → /docs, /openapi.json ----
  http GET /
  check_eq "GET / 重定向" "307" "$R_STATUS"
  http GET /openapi.json
  check_status "openapi.json 公开" 200

  # ---- AUTH_DISABLED 列 (需第二实例) ----
  if [ -n "$DISABLED_URL" ]; then
    echo "  -- AUTH_DISABLED 列: $DISABLED_URL --"
    local SAVE_URL="$BASE_URL"
    BASE_URL="$DISABLED_URL"
    http GET "/memories?user_id=dx";         check_status "disabled: verify_auth 放行" 200
    http GET "/memories?top_k=1000";         check_status "disabled: 模式B 放行" 200
    http DELETE "/memories?user_id=dx";      check_status "disabled: require_admin 放行" 200
    http GET /auth/me;                       check_status "disabled: require_auth 落首用户" 200
    http GET "/requests?limit=5";            check_status "disabled: require_admin 放行(requests)" 200
    http GET /entities;                      check_status "disabled: entities 放行" 200
    BASE_URL="$SAVE_URL"
  else
    echo "  -- 跳过 AUTH_DISABLED 列 (CONTRACT_DISABLED_URL 未设) --"
  fi
}
