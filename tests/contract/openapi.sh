#!/usr/bin/env bash
# 契约: /openapi.json 快照 (FastAPI 自动产物 = 合同一部分; R3: Go 侧静态 golden 直接 serve)。
# 不走 normalize(): openapi 内容对固定版本完全确定。

suite() {
  http GET /openapi.json
  check_status "openapi.json" 200
  local f="$GOLDENS_DIR/openapi__doc.json"
  if [ "$CAPTURE" = "1" ]; then
    mkdir -p "$GOLDENS_DIR"
    printf '%s' "$R_BODY" | jq -S . > "$f"
    pass "golden: openapi (采集)"
  else
    printf '%s' "$R_BODY" | jq -S . > "$TMP_DIFF"
    if diff -u "$f" "$TMP_DIFF" > "$TMP_DIFF.d"; then
      pass "golden: openapi"
    else
      fail "golden: openapi (diff)"
      diff -u "$f" "$TMP_DIFF" | sed 's/^/    /' | head -30
    fi
  fi
}
