#!/usr/bin/env bash
# 契约: /auth/* 业务流 (doc-02 §4)。require_auth 回退链 (admin key → 首用户)、
# refresh 一次性 (T11)、改密码。注: 409 email 冲突需第二用户, registration closed 不可触发 → Go 单测覆盖。

suite() {
  local AH_UKEY="X-API-Key: $USER_API_KEY"
  local AH_AKEY="X-API-Key: $ADMIN_API_KEY"
  local AH_JWT="Authorization: Bearer $ACCESS_TOKEN"

  # ---------- /auth/me 三凭据同用户 (require_auth 回退: admin key 落首用户) ----------
  http GET /auth/me "$AH_JWT"
  check_status "me: bearer" 200
  check_json "me role" '.role' 'admin'
  check_json "me email" '.email' 'contract@example.com'
  g "me-bearer"

  http GET /auth/me "$AH_UKEY"
  check_status "me: user key" 200
  check_json "me: user key 同用户" '.email' 'contract@example.com'

  http GET /auth/me "$AH_AKEY"
  check_status "me: admin key (落首用户)" 200
  check_json "me: admin key 同用户" '.email' 'contract@example.com'
  g "me-admin-key-fallback"

  # ---------- PATCH /auth/me ----------
  http PATCH /auth/me '{"name":"Renamed Admin"}' "$AH_JWT"
  check_status "PATCH name" 200
  http GET /auth/me "$AH_JWT"
  check_json "name 已改" '.name' 'Renamed Admin'
  http PATCH /auth/me '{"name":"Contract Admin"}' "$AH_JWT"
  check_status "PATCH name 复原" 200

  # ---------- refresh 一次性 (T11: jti 条件 UPDATE) ----------
  http POST /auth/refresh "{\"refresh_token\": \"$REFRESH_TOKEN\"}"
  check_status "refresh 首次" 200
  check_json "refresh token_type" '.token_type' 'bearer'
  g "refresh-first"
  NEW_REFRESH=$(printf '%s' "$R_BODY" | jq -r .refresh_token)
  # 重放旧 token → 恰一失败
  http POST /auth/refresh "{\"refresh_token\": \"$REFRESH_TOKEN\"}"
  check_status "refresh 重放 → 401" 401
  check_json "重放文案" '.detail' 'Refresh token is no longer valid.'
  g "refresh-replay"
  REFRESH_TOKEN=$NEW_REFRESH

  # ---------- 改密码 ----------
  http POST /auth/change-password '{"current_password":"wrong-pass","new_password":"another-pass-123"}' "$AH_JWT"
  check_status "改密码: 当前密码错 → 401" 401
  g "change-password-wrong"
  http POST /auth/change-password '{"current_password":"contract-pass-123","new_password":"short"}' "$AH_JWT"
  check_status "改密码: 新密码过短 → 400" 400
  g "change-password-short"
  http POST /auth/change-password '{"current_password":"contract-pass-123","new_password":"contract-pass-456"}' "$AH_JWT"
  check_status "改密码成功" 200
  check_json "改密码文案" '.message' 'Password updated.'
  g "change-password-ok"
  # 新密码可登录
  http POST /auth/login '{"email":"contract@example.com","password":"contract-pass-456"}'
  check_status "新密码登录" 200
  ACCESS_TOKEN=$(printf '%s' "$R_BODY" | jq -r .access_token)
  REFRESH_TOKEN=$(printf '%s' "$R_BODY" | jq -r .refresh_token)
  # 复原
  http POST /auth/change-password '{"current_password":"contract-pass-456","new_password":"contract-pass-123"}' "Authorization: Bearer $ACCESS_TOKEN"
  check_status "复原密码" 200
  http POST /auth/login '{"email":"contract@example.com","password":"contract-pass-123"}'
  check_status "复原后登录" 200
  ACCESS_TOKEN=$(printf '%s' "$R_BODY" | jq -r .access_token)
  REFRESH_TOKEN=$(printf '%s' "$R_BODY" | jq -r .refresh_token)
}
