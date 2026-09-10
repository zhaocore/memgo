#!/usr/bin/env bash
# 契约: 错误信封 (doc-02 §1.3)。502 code 分类 (stub 故障注入)、404 映射、422 pydantic 格式、401 头。
# 注: provider_timeout 需客户端超时 (openai 默认 600s), 黑盒不可测 —— Go 分类器单测覆盖。

suite() {
  local AH_UKEY="X-API-Key: $USER_API_KEY"

  # ---------- 404: ValueError "not found" 映射 ----------
  http PUT '/memories/22222222-3333-4444-5555-666666666666' '{"text":"x"}' "$AH_UKEY"
  check_status "PUT 不存在 → 404" 404
  check_absent "404 body 无 code 键" '"code"'
  check_contains "detail 含 not found" 'not found'
  g "404-put-missing"

  # ---------- 502 code 分类 (异常链归因, stub 逐模式注入) ----------
  for m in auth rate_limit bad_request unavailable; do
    stub_mode "$m"
    http POST /memories '{"messages":[{"role":"user","content":"trigger LLM"}],"user_id":"stub-user","infer":true}' "$AH_UKEY"
    stub_mode normal
    case $m in
      auth)        EXP=provider_auth_failed ;;
      rate_limit)  EXP=provider_rate_limited ;;
      bad_request) EXP=provider_bad_request ;;
      unavailable) EXP=provider_unavailable ;;
    esac
    check_status "502 mode=$m" 502
    check_json "code 映射 mode=$m" '.code' "$EXP"
    check_json "request_id 8hex (mode=$m)" '.request_id | test("^[0-9a-f]{8}$")' 'true'
    check_eq "X-Request-ID 头 (mode=$m)" 'yes' "$(printf '%s' "$R_HDR_XREQ" | grep -qE '^[0-9a-f]{8}$' && echo yes || echo no)"
    g "502-$m"
  done

  # ---------- 401 + WWW-Authenticate ----------
  http GET '/memories?user_id=x'
  check_status "无凭据 401" 401
  check_header "WWW-Authenticate: Bearer" "Bearer"

  # ---------- 422 (pydantic 默认格式, 红线 T2) ----------
  http POST /memories '{}' "$AH_UKEY"
  check_status "422 缺 messages" 422
  check_json "422 detail[0].type" '.detail[0].type' 'missing'
  check_json "422 detail[0].loc[0]" '.detail[0].loc[0]' 'body'
  check_json "422 detail[0].loc[1]" '.detail[0].loc[1]' 'messages'
  check_json "422 detail[0].msg" '.detail[0].msg' 'Field required'
  g "422-missing-messages"

  http POST /memories '{"messages":[{"role":"user"}]}' "$AH_UKEY"
  check_status "422 缺 content" 422
  check_json "422 嵌套 loc (索引+字段)" '.detail[0].loc[3]' 'content'
  g "422-missing-content"

  http POST /search '{}' "$AH_UKEY"
  check_status "422 search 缺 query" 422
  check_json "422 search loc" '.detail[0].loc[1]' 'query'
  g "422-missing-query"

  # ---------- 400 ----------
  http POST /memories '{"messages":[{"role":"user","content":"x"}]}' "$AH_UKEY"
  check_status "无实体 id → 400" 400
  check_json "400 文案" '.detail' 'At least one identifier (user_id, agent_id, run_id) is required.'
  g "400-no-identifier"

  # ---------- X-Request-ID ----------
  http GET /auth/setup-status
  check_eq "X-Request-ID 8hex" 'yes' "$(printf '%s' "$R_HDR_XREQ" | grep -qE '^[0-9a-f]{8}$' && echo yes || echo no)"
}
