#!/usr/bin/env bash
# 契约: 限流 429 (T13)。slowapi 桶: register 5/min, login 10/min, refresh 20/min (按远端 IP)。
# 依赖: setup/register 已占 register 桶 1 格; 套件需在 1 分钟内跑完本段。
# 注: minute 边界穿越可能引入偶发 (重跑即可)。

suite() {
  # ---------- register: 已注册过 → 额度内为 403, 超额 429; 此处 5 连发应全 429 ----------
  LAST429=""
  OK429=0
  for i in 1 2 3 4 5; do
    http POST /auth/register '{"name":"x","email":"x@x.dev","password":"12345678"}'
    if [ "$R_STATUS" = "429" ]; then OK429=$((OK429+1)); LAST429=$R_BODY; fi
  done
  # 桶 5/min: setup 已耗 1 格 → 恰 1 个 429, 其余 403 (registration closed)
  if [ "$OK429" -ge 1 ]; then pass "register 429 出现 ($OK429 次)"; else fail "register 429 未出现"; fi
  R_BODY="$LAST429"
  g_raw "429-register-body"

  # ---------- login: 10/min ----------
  OK429=0
  for i in $(seq 1 10); do
    http POST /auth/login '{"email":"contract@example.com","password":"contract-pass-123"}'
    if [ "$R_STATUS" = "429" ]; then OK429=$((OK429+1)); LAST429=$R_BODY; fi
  done
  if [ "$OK429" -ge 1 ]; then pass "login 429 出现 ($OK429 次)"; else fail "login 429 未出现"; fi
  R_BODY="$LAST429"
  g_raw "429-login-body"

  # ---------- refresh: 20/min ----------
  OK429=0
  for i in $(seq 1 20); do
    http POST /auth/refresh '{"refresh_token":"invalid.token.value"}'
    if [ "$R_STATUS" = "429" ]; then OK429=$((OK429+1)); LAST429=$R_BODY; fi
  done
  if [ "$OK429" -ge 1 ]; then pass "refresh 429 出现 ($OK429 次)"; else fail "refresh 429 未出现"; fi
  R_BODY="$LAST429"
  g_raw "429-refresh-body"
}
