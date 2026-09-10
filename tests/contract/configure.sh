#!/usr/bin/env bash
# 契约: /configure (doc-02 §3)。深合并非替换 (T7) + redact + provider 白名单。
# 放套件末段: POST /configure 会重建 Memory 实例并持久化 overrides。

suite() {
  local AH_UKEY="X-API-Key: $USER_API_KEY"
  local AH_AKEY="X-API-Key: $ADMIN_API_KEY"

  # ---------- GET: redact (敏感键恒 "[redacted]") ----------
  http GET /configure "$AH_UKEY"
  check_status "GET /configure" 200
  check_json "version" '.version' 'v1.1'
  check_json "llm.api_key 已脱敏" '.llm.config.api_key' '[redacted]'
  check_json "vector_store.password 已脱敏" '.vector_store.config.password' '[redacted]'
  g "configure-get"

  http GET /configure/providers "$AH_UKEY"
  check_status "GET providers" 200
  g "configure-providers"

  # ---------- POST: 深合并 ----------
  http POST /configure '{"llm":{"config":{"model":"stub-llm-model"}}}' "$AH_AKEY"
  check_status "POST configure" 200
  check_json "POST 文案" '.message' 'Configuration set successfully'
  g "configure-post"

  http GET /configure "$AH_AKEY"
  check_status "GET after POST" 200
  check_json "model 已合并" '.llm.config.model' 'stub-llm-model'
  check_json "兄弟键 temperature 保留 (深合并证据)" '.llm.config.temperature' '0.2'
  check_json "embedder 未受影响" '.embedder.config.model' 'text-embedding-3-small'
  g "configure-after-merge"

  # ---------- provider 白名单 ----------
  http POST /configure '{"llm":{"provider":"mistral"}}' "$AH_AKEY"
  check_status "非内置 LLM provider → 400" 400
  check_contains "detail 指 BUNDLED_LLM_PROVIDERS" 'BUNDLED_LLM_PROVIDERS'
  g "400-llm-provider"

  http POST /configure '{"embedder":{"provider":"cohere"}}' "$AH_AKEY"
  check_status "非内置 embedder provider → 400" 400
  check_contains "detail 指 BUNDLED_EMBEDDER_PROVIDERS" 'BUNDLED_EMBEDDER_PROVIDERS'
  g "400-embedder-provider"

  # ---------- 鉴权 ----------
}
