#!/usr/bin/env bash
# OSS backend CLI 冒烟 (P3 验收): memgo CLI 打 MemGo server 往返。
# HOME 隔离: 配置读写落到临时目录, 不污染真实 ~/.memgo。
# 前提: Go server 运行于 CONTRACT_BASE_URL (默认 http://localhost:18888), stub 已起。
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
BASE_URL="${CONTRACT_BASE_URL:-http://localhost:18888}"
ADMIN_KEY="${CONTRACT_ADMIN_API_KEY:-contract-admin-key-0123456789}"

SMOKE_HOME="$(mktemp -d /tmp/memgo_smoke.XXXXXX)"
trap 'rm -rf "$SMOKE_HOME"' EXIT

MEMGO_BIN="${MEMGO_BIN:-/tmp/memgo-bin}"
if [ ! -x "$MEMGO_BIN" ]; then
  echo "构建 memgo CLI..."
  (cd "$HERE/../.." && go build -o /tmp/memgo-bin ./cli/go/cmd/memgo)
fi

export HOME="$SMOKE_HOME"
export MEMGO_BASE_URL="$BASE_URL"
export MEMGO_API_KEY="$ADMIN_KEY"
PASS=0; FAIL=0
ok() { PASS=$((PASS+1)); echo "  ok    $1"; }
bad() { FAIL=$((FAIL+1)); echo "  FAIL  $1"; }
run() { "$MEMGO_BIN" "$@" 2>&1; }

OUT=$(run status)
printf '%s' "$OUT" | grep -q "Connected to oss backend" && ok "status 连接" || bad "status: $OUT"

OUT=$(run add "Alice likes hiking in the mountains." -u smoke-alice --no-infer -o json)
printf '%s' "$OUT" | grep -q '"event":"ADD"' && ok "add (infer=false)" || bad "add: $OUT"

OUT=$(run search "hiking" -u smoke-alice -o json)
printf '%s' "$OUT" | grep -q '"memory":"Alice likes hiking in the mountains."' && ok "search 命中" || bad "search: $OUT"

MID=$(printf '%s' "$OUT" | jq -r '.[0].id')
[ -n "$MID" ] && [ "$MID" != "null" ] && ok "取得 memory id" || bad "id 提取: $OUT"

OUT=$(run get "$MID" -o json)
printf '%s' "$OUT" | grep -q '"memory":"Alice likes hiking in the mountains."' && ok "get 往返" || bad "get: $OUT"

OUT=$(run update "$MID" "Alice loves trail running now." -o json)
printf '%s' "$OUT" | grep -q "updated successfully" && ok "update" || bad "update: $OUT"

OUT=$(run entity list users -o json)
printf '%s' "$OUT" | grep -q '"id":"smoke-alice"' && ok "entity list" || bad "entity list: $OUT"

OUT=$(run entity delete -u smoke-alice --force -o json)
printf '%s' "$OUT" | grep -q "Entity deleted" && ok "entity delete (级联)" || bad "entity delete: $OUT"

OUT=$(run list -u smoke-alice -o json)
[ "$OUT" = "[]" ] && ok "级联删除后为空" || bad "list after delete: $OUT"

OUT=$(run event list)
printf '%s' "$OUT" | grep -q "not supported on the OSS backend" && ok "event 显式不支持" || bad "event: $OUT"

echo "[cli-oss-smoke] $PASS ok / $FAIL fail"
[ "$FAIL" -eq 0 ]
