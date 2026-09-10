#!/usr/bin/env bash
# 契约套件编排器 (P0)。
# 用法:
#   ./run.sh [--impl python|go] [--capture] [--only <suite>]
# 环境变量:
#   CONTRACT_BASE_URL      server 地址 (默认 http://localhost:18888, Python 容器)
#   CONTRACT_DISABLED_URL  AUTH_DISABLED=true 第二实例 (可选, 补齐矩阵 disabled 列; 默认 http://localhost:18889)
#   STUB_BASE_URL          openai 打桩 server (默认 http://localhost:8090, 未运行则自动拉起)
# 前提: server 以全新库启动 (GET /auth/setup-status 需 needsSetup=true); compose 契约栈见 compose.contract.yaml。
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"

IMPL="python"
CAPTURE_FLAG="${CAPTURE:-0}"
ONLY=""
while [ $# -gt 0 ]; do
  case "$1" in
    --impl) IMPL=$2; shift 2 ;;
    --capture) CAPTURE_FLAG=1; shift ;;
    --only) ONLY=$2; shift 2 ;;
    *) echo "未知参数: $1"; exit 2 ;;
  esac
done
export CAPTURE=$CAPTURE_FLAG
# shellcheck disable=SC1091
source "$HERE/conftest.sh"

# --- 前置检查 ---
STATUS=$(curl -s "$BASE_URL/auth/setup-status" 2>/dev/null || true)
if ! printf '%s' "$STATUS" | grep -q needsSetup; then
  echo "server 不可达: $BASE_URL —— 先起契约栈: podman compose -f tests/contract/compose.contract.yaml up -d --build"
  exit 1
fi
NEEDS=$(printf '%s' "$STATUS" | jq -r .needsSetup)
if [ "$NEEDS" != "true" ]; then
  echo "库非空 (needsSetup=false) —— 套件要求全新库:"
  echo "  podman compose -f tests/contract/compose.contract.yaml down -v && podman compose -f tests/contract/compose.contract.yaml up -d"
  exit 1
fi

# --- 打桩 server 自动拉起 ---
STUB_PID=""
if ! curl -s "$STUB_URL/_control" > /dev/null 2>&1; then
  echo ">> 启动 openai 打桩 server: $STUB_URL"
  (cd "$HERE/stub" && go build -o /tmp/ct_stub .) && /tmp/ct_stub > /tmp/ct_stub.log 2>&1 &
  STUB_PID=$!
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
    curl -s "$STUB_URL/_control" > /dev/null 2>&1 && break
    sleep 0.3
  done
  curl -s "$STUB_URL/_control" > /dev/null 2>&1 || { echo "打桩 server 启动失败, 见 /tmp/ct_stub.log"; exit 1; }
fi
trap '[ -n "$STUB_PID" ] && kill "$STUB_PID" 2>/dev/null' EXIT

if [ "$IMPL" = "go" ]; then
  echo ">> 目标实现: Go (CONTRACT_BASE_URL=$BASE_URL)"
else
  echo ">> 目标实现: Python (基线)"
fi
[ "$CAPTURE" = "1" ] && echo ">> 模式: 采集 golden"

SUITE_RC=0
for s in setup auth_matrix memories entities error_envelope auth_flow rate_limit requests configure openapi; do
  if [ -n "$ONLY" ] && [ "$ONLY" != "$s" ]; then continue; fi
  echo ""
  echo "== suite: $s =="
  # shellcheck disable=SC1090
  source "$HERE/$s.sh"
  SUITE=$s
  suite
  if ! report; then SUITE_RC=1; fi
done

echo ""
if [ "$SUITE_RC" = "0" ]; then
  echo "契约套件全绿 ($IMPL)"
else
  echo "契约套件存在失败 ($IMPL)"
fi
exit "$SUITE_RC"
