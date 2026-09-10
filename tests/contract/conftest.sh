#!/usr/bin/env bash
# 用法: run.sh source 本文件, 各套件脚本定义 suite() 函数, run.sh 顺序执行。
# 所有 golden 比对前先经 normalize() 抹平易变字段(id/时间戳/token/hash/分数/latency)。

set -u

SUITE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GOLDENS_DIR="$SUITE_DIR/goldens"
TMP_BODY="$(mktemp /tmp/ct_body.XXXXXX)"
TMP_HDR="$(mktemp /tmp/ct_hdr.XXXXXX)"
TMP_DIFF="$(mktemp /tmp/ct_diff.XXXXXX)"

BASE_URL="${CONTRACT_BASE_URL:-http://localhost:18888}"
DISABLED_URL="${CONTRACT_DISABLED_URL:-http://localhost:18889}"
STUB_URL="${STUB_BASE_URL:-http://localhost:8090}"
ADMIN_API_KEY="${CONTRACT_ADMIN_API_KEY:-contract-admin-key-0123456789}"

CAPTURE="${CAPTURE:-0}"
PASS=0
FAIL=0
FAILED=()

# ---------- HTTP ----------
# http METHOD PATH [BODY] [HEADER...]  -> R_STATUS / R_BODY / R_HDR_XREQ / R_HDR_WWA
http() {
  local method=$1 path=$2
  shift 2
  local body=""
  if [ $# -gt 0 ]; then
    body=$1
    shift
  fi
  local headers=("$@")
  # 无 body 调用易把 header 误传为 body —— 按前缀识别归位
  case "$body" in
    "Authorization: "*|"X-API-Key: "*)
      headers=("$body" ${headers[@]+"${headers[@]}"})
      body=""
      ;;
  esac
  local args=(-s -X "$method" "$BASE_URL$path" -o "$TMP_BODY" -D "$TMP_HDR" -w '%{http_code}')
  local h
  for h in ${headers[@]+"${headers[@]}"}; do args+=(-H "$h"); done
  if [ -n "$body" ]; then
    args+=(-H 'Content-Type: application/json' --data "$body")
  fi
  R_STATUS=$(curl "${args[@]}")
  R_BODY=$(cat "$TMP_BODY")
  R_HDR_XREQ=$(grep -i '^x-request-id:' "$TMP_HDR" | awk '{print $2}' | tr -d '\r')
  R_HDR_WWA=$(grep -i '^www-authenticate:' "$TMP_HDR" | cut -d' ' -f2- | tr -d '\r')
}

# 向打桩 server 注入故障模式: normal|auth|rate_limit|bad_request|unavailable
stub_mode() {
  curl -s -X POST "$STUB_URL/_control" -H 'Content-Type: application/json' -d "{\"mode\": \"$1\"}" > /dev/null
}

# ---------- 归一化 ----------
# 跨实现/跨运行可比: token→<JWT> key→<KEY> uuid→<UUID> 时间戳→<TS>; 键级: hash/latency/token/key/score
normalize() {
  jq -S '
  def jwt_re: "eyJ[A-Za-z0-9_-]+\\.[A-Za-z0-9_-]+\\.[A-Za-z0-9_-]*";
  def key_re: "m0sk_[A-Za-z0-9_-]{5,}";
  def uuid_re: "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}";
  def ts_re: "[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]+)?";
  walk(
    if type == "object" then
      with_entries(
        if .key == "hash" then .value = "<HASH>"
        elif .key == "latency_ms" then .value = "<LAT>"
        elif .key == "request_id" then .value = "<RID>"
        elif .key == "history_db_path" then .value = "<HIST>"
        elif .key == "host" then .value = "<HOST>"
        elif .key == "port" then .value = "<PORT>"
        elif (.key == "access_token" or .key == "refresh_token") then .value = "<JWT>"
        elif (.key == "key" and (.value|type)=="string" and (.value|startswith("m0sk_"))) then .value = "<KEY>"
        elif .key == "score" and (.value|type)=="number" then .value = ((.value * 10000 | round) / 10000)
        else . end
      )
    else . end
  )
  | walk(
      if type == "string" then
        gsub(jwt_re; "<JWT>")
        | gsub(key_re; "<KEY>")
        | gsub(uuid_re; "<UUID>")
        | gsub(ts_re; "<TS>")
      else . end
    )
  # 非整数浮点取 9 位小数: 跨语言 libm (math.exp) 的 ULP 级差异属实现细节
  | walk(if type == "number" and . != floor then ((. * 1000000000 | round) / 1000000000) else . end)
  ' 2>/dev/null || cat
}

# ---------- 断言 ----------
pass() { PASS=$((PASS+1)); printf '  ok    %s\n' "$1"; }
fail() { FAIL=$((FAIL+1)); FAILED+=("$1"); printf '  FAIL  %s\n' "$1"; }

check_eq() { # check_eq <name> <expected> <actual>
  if [ "$2" = "$3" ]; then pass "$1"; else fail "$1 (期望[$2] 实际[$3])"; fi
}
check_status() { check_eq "$1: status" "$2" "$R_STATUS"; }
check_json() { # check_json <name> <jq表达式> <期望值>
  local actual
  actual=$(printf '%s' "$R_BODY" | jq -r "$2" 2>&1)
  check_eq "$1" "$3" "$actual"
}
check_header() { # check_header <name> <期望头值> — 断言最近一次响应的 WWW-Authenticate
  check_eq "$1" "$2" "$R_HDR_WWA"
}
check_contains() { # check_contains <name> <子串>
  if printf '%s' "$R_BODY" | grep -qF -- "$2"; then pass "$1"; else fail "$1 (body 未含[$2])"; fi
}

# JSON golden 采集/比对: g <name>
g() {
  local name=$1
  local norm
  norm=$(printf '%s' "$R_BODY" | normalize)
  local f="$GOLDENS_DIR/${SUITE}__${name}.json"
  if [ "$CAPTURE" = "1" ]; then
    mkdir -p "$GOLDENS_DIR"
    printf '%s\n' "$norm" | jq -S . > "$f" 2>/dev/null || printf '%s\n' "$norm" > "$f"
    pass "golden: $name (采集)"
  else
    if [ ! -f "$f" ]; then
      fail "golden: $name (缺失: $f)"
      return
    fi
    printf '%s\n' "$norm" | jq -S . > "$TMP_DIFF" 2>/dev/null
    if diff -u "$f" "$TMP_DIFF" > "$TMP_DIFF.d"; then
      pass "golden: $name"
    else
      fail "golden: $name (diff 见输出)"
      diff -u "$f" "$TMP_DIFF" | sed 's/^/    /' | head -30
    fi
  fi
}

# 非 JSON golden (纯文本 429 体等): g_raw <name>
g_raw() {
  local name=$1
  local f="$GOLDENS_DIR/${SUITE}__${name}.json"
  if [ "$CAPTURE" = "1" ]; then
    mkdir -p "$GOLDENS_DIR"
    printf '%s\n' "$R_BODY" > "$f"
    pass "golden_raw: $name (采集)"
  else
    if [ ! -f "$f" ]; then
      fail "golden_raw: $name (缺失: $f)"
      return
    fi
    if diff -u "$f" <(printf '%s\n' "$R_BODY") > "$TMP_DIFF.d"; then
      pass "golden_raw: $name"
    else
      fail "golden_raw: $name (diff)"
      diff -u "$f" <(printf '%s\n' "$R_BODY") | sed 's/^/    /' | head -30
    fi
  fi
}

report() {
  echo "[$SUITE] $PASS ok / $FAIL fail"
  if [ "$FAIL" -gt 0 ]; then return 1; fi
}

check_absent() { # check_absent <name> <子串>  — body 不应包含
  if printf '%s' "$R_BODY" | grep -qF -- "$2"; then fail "$1 (body 不应含[$2])"; else pass "$1"; fi
}
