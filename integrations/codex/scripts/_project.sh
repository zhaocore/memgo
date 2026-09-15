#!/usr/bin/env bash
# Source this file. Sets MEMGO_AGENT_ID 和 MEMGO_BRANCH。
#
# 解析优先级(agent_id):
#   1. MEMGO_AGENT_ID 环境变量(显式覆盖)
#   2. ~/.memgo/project_map.json 按 $PWD 查找(需要 jq)
#   3. Git remote slug: 去掉协议前缀与 .git, / 和 : 换成 -
#      e.g. git@github.com:memgoai/memgo.git -> memgoai-memgo
#   4. 兜底: $PWD 的 basename
#
# branch 解析:
#   git branch --show-current, 失败时 "unknown"

_memgo_resolve_agent_id() {
  # 1. 显式覆盖
  if [ -n "${MEMGO_AGENT_ID:-}" ]; then
    printf '%s' "$MEMGO_AGENT_ID"
    return
  fi

  # 2. project_map.json 按 $PWD 查找
  _memgo_map="$HOME/.memgo/project_map.json"
  if [ -f "$_memgo_map" ] && command -v jq >/dev/null 2>&1; then
    _memgo_mapped=$(jq -r --arg cwd "$PWD" '.[$cwd] // empty' "$_memgo_map" 2>/dev/null)
    if [ -n "$_memgo_mapped" ]; then
      printf '%s' "$_memgo_mapped"
      return
    fi
  fi

  # 3. Git remote slug
  _memgo_remote_url=$(git remote get-url origin 2>/dev/null)
  if [ -n "$_memgo_remote_url" ]; then
    _memgo_slug="$_memgo_remote_url"
    # 去掉 .git 后缀
    _memgo_slug="${_memgo_slug%.git}"
    # 去掉协议前缀
    _memgo_slug="${_memgo_slug#https://}"
    _memgo_slug="${_memgo_slug#http://}"
    _memgo_slug="${_memgo_slug#ssh://}"
    _memgo_slug="${_memgo_slug#git://}"
    _memgo_slug="${_memgo_slug#git@}"
    # 把第一个冒号(SSH host:path 分隔符)换成 /
    _memgo_slug="${_memgo_slug/://}"
    # 只保留最后两段(owner/repo)
    _memgo_owner=$(printf '%s' "$_memgo_slug" | awk -F'/' '{print $(NF-1)}')
    _memgo_repo=$(printf '%s' "$_memgo_slug" | awk -F'/' '{print $NF}')
    _memgo_slug="${_memgo_owner}-${_memgo_repo}"
    # 剩余的 / 和 : 换成 -
    _memgo_slug="${_memgo_slug//\//-}"
    _memgo_slug="${_memgo_slug//:/-}"
    if [ -n "$_memgo_slug" ]; then
      printf '%s' "$_memgo_slug"
      _MEMGO_PERSIST_CWD="$PWD" _MEMGO_PERSIST_SLUG="$_memgo_slug" python3 -c "
import os, sys
sys.path.insert(0, '$(dirname "${BASH_SOURCE[0]:-$0}")')
from _project import save_project_mapping
save_project_mapping(os.environ['_MEMGO_PERSIST_CWD'], os.environ['_MEMGO_PERSIST_SLUG'])
" 2>/dev/null || true
      return
    fi
  fi

  # 4. 兜底: $PWD 的 basename
  printf '%s' "$(basename "$PWD")"
}

_memgo_resolve_branch() {
  git branch --show-current 2>/dev/null || printf 'unknown'
}

MEMGO_AGENT_ID="$(_memgo_resolve_agent_id)"
MEMGO_BRANCH="$(_memgo_resolve_branch)"
export MEMGO_AGENT_ID
export MEMGO_BRANCH
