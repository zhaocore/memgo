#!/usr/bin/env bash
# Source this file. Sets MEMGO_API_KEY, MEMGO_RESOLVED_USER_ID, 以及设置项。
#
# API key 解析(第一个非空生效):
#   1. MEMGO_API_KEY 环境变量(显式 / shell profile)
#   2. 从 shell profile 文件抽取(~/.zshrc, ~/.bashrc 等)
#      某些桌面端不继承 shell 环境变量, 此兜底覆盖常见 export 写法。
#
# Settings: ~/.memgo/settings.json(用户可编辑, 缺失时回退默认值)

_SCRIPT_DIR="$( cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd )"

# 从 shell profile 文件抽取 MEMGO_API_KEY(不 source 整个 profile, 避免副作用)
if [ -z "${MEMGO_API_KEY:-}" ]; then
  for _profile in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.zprofile" "$HOME/.bash_profile" "$HOME/.profile"; do
    if [ -f "$_profile" ]; then
      _extracted=$(grep -E '^\s*(export\s+)?MEMGO_API_KEY=' "$_profile" 2>/dev/null \
        | tail -1 \
        | sed 's/^[^=]*=//' \
        | sed "s/^[\"']//;s/[\"']$//" \
        | sed 's/#.*//' \
        | tr -d '[:space:]')
      # 只接受字面值, 跳过变量引用(如 ${OTHER_VAR})
      if [ -n "$_extracted" ] && [ "${_extracted#\$}" = "$_extracted" ]; then
        MEMGO_API_KEY="$_extracted"
        export MEMGO_API_KEY
        break
      fi
    fi
  done
fi

_memgo_resolve_user_id() {
  if [ -n "${MEMGO_USER_ID:-}" ]; then
    printf '%s' "$MEMGO_USER_ID"
    return
  fi
  printf '%s' "${USER:-default}"
}

MEMGO_RESOLVED_USER_ID="$(_memgo_resolve_user_id)"
export MEMGO_RESOLVED_USER_ID

_MEMGO_IDENTITY_ANNOTATION=""
if [ -n "${MEMGO_USER_ID:-}" ] && [ "$MEMGO_USER_ID" != "${USER:-default}" ]; then
  _MEMGO_IDENTITY_ANNOTATION=" (override; default: ${USER:-default})"
fi
export _MEMGO_IDENTITY_ANNOTATION

# Load settings from ~/.memgo/settings.json
if command -v python3 >/dev/null 2>&1; then
  _SETTINGS_JSON=$(PYTHONPATH="$_SCRIPT_DIR" python3 -c "from load_settings import load_settings; import json; print(json.dumps(load_settings()))" 2>/dev/null || echo "{}")
  MEMGO_AUTO_SAVE=$(echo "$_SETTINGS_JSON" | python3 -c "import sys,json; print(str(json.load(sys.stdin).get('auto_save',True)).lower())" 2>/dev/null || echo "true")
  MEMGO_AUTO_SEARCH=$(echo "$_SETTINGS_JSON" | python3 -c "import sys,json; print(str(json.load(sys.stdin).get('auto_search',True)).lower())" 2>/dev/null || echo "true")
  MEMGO_SEARCH_LIMIT=$(echo "$_SETTINGS_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin).get('search_limit',10))" 2>/dev/null || echo "10")
  MEMGO_DEBUG=$(echo "$_SETTINGS_JSON" | python3 -c "import sys,json; print(str(json.load(sys.stdin).get('debug',False)).lower())" 2>/dev/null || echo "false")
else
  MEMGO_AUTO_SAVE="true"
  MEMGO_AUTO_SEARCH="true"
  MEMGO_SEARCH_LIMIT="10"
  MEMGO_DEBUG="false"
fi
export MEMGO_AUTO_SAVE MEMGO_AUTO_SEARCH MEMGO_SEARCH_LIMIT MEMGO_DEBUG

# 同时解析仓库身份(agent_id)与分支
. "$_SCRIPT_DIR/_project.sh"
