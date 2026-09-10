#!/usr/bin/env bash
# 压测基线: add(infer=false)/search 顺序 N 次, 报 P50/P95 (秒)。真实 LLM 路径需自备 key 未纳入。
# 用法: TARGET=http://localhost:8888 KEY=<api-key> ./bench.sh [N]
set -euo pipefail
TARGET="${TARGET:-http://localhost:8888}"
KEY="${KEY:?KEY=<api-key>}"
N="${1:-50}"

BENCH_USER="bench-$(date +%s)"
run_probe() { # method path body → 秒
  local t
  t=$(curl -s -o /dev/null -w '%{time_total}' -X "$1" "$TARGET$2" \
    -H "X-API-Key: $KEY" -H 'Content-Type: application/json' ${3:+--data "$3"})
  echo "$t"
}

# 预热
run_probe POST /memories '{"messages":[{"role":"user","content":"warmup"}],"user_id":"'$BENCH_USER'","infer":false}' > /dev/null

# add 批次 (每条不同文本避免状态影响; infer=false 直存)
ADDTIMES=()
for i in $(seq 1 "$N"); do
  t=$(run_probe POST /memories '{"messages":[{"role":"user","content":"Bench note number '$i' about project alpha"}],"user_id":"'$BENCH_USER'","infer":false}')
  ADDTIMES+=("$t")
done

# search 批次
SEARTIMES=()
for i in $(seq 1 "$N"); do
  t=$(run_probe POST /search '{"query":"project alpha '$i'","filters":{"user_id":"'$BENCH_USER'"},"top_k":10}')
  SEARTIMES+=("$t")
done

pct() { # array → P50 P95
  local arr=("$@") sorted
  sorted=$(printf '%s\n' "${arr[@]}" | sort -n)
  local n=${#arr[@]}
  echo "$sorted" | awk -v n="$n" -v p50="$(echo "$sorted" | sed -n "$(( (n+1)/2 ))p")" -v p95="$(echo "$sorted" | sed -n "$(( (n*95+99)/100 ))p")" 'BEGIN{print p50, p95}'
}

read A50 A95 <<< "$(pct "${ADDTIMES[@]}")"
read S50 S95 <<< "$(pct "${SEARTIMES[@]}")"
printf 'target=%s n=%d\nadd    P50=%.4fs P95=%.4fs\nsearch P50=%.4fs P95=%.4fs\n' "$TARGET" "$N" "$A50" "$A95" "$S50" "$S95"
