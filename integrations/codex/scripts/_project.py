"""Resolve MemGo agent_id(仓库身份)和 branch。

OSS 面没有 app_id, 仓库范围用 agent_id 表达(见 integrations/README.md 实体范围)。

解析优先级(agent_id):
  1. MEMGO_AGENT_ID 环境变量(显式覆盖)
  2. ~/.memgo/project_map.json 按 cwd 查找
  2b. ~/.memgo/project_map.json 按 remote hash 查找(目录移动/改名自愈)
  3. Git remote slug: git@github.com:owner/repo.git -> owner-repo
  4. 兜底: cwd 的 basename
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import subprocess


def resolve_agent_id(cwd: str | None = None) -> str:
    if cwd is None:
        cwd = os.getcwd()

    # 1. 显式覆盖
    explicit = os.environ.get("MEMGO_AGENT_ID", "").strip()
    if explicit:
        return explicit

    # 2. project_map.json 查找
    map_path = os.path.expanduser("~/.memgo/project_map.json")
    if os.path.isfile(map_path):
        try:
            with open(map_path) as f:
                project_map = json.load(f)
            mapped = project_map.get(cwd, "").strip()
            if mapped:
                return mapped
            # 2b. remote hash 兜底(文件夹被移动/改名时自愈)
            remote_key = _remote_hash_key(cwd)
            if remote_key:
                mapped = project_map.get(remote_key, "").strip()
                if mapped:
                    # 自愈: 写入新 cwd 的 key, 后续查找更快
                    project_map[cwd] = mapped
                    try:
                        with open(map_path, "w") as f:
                            json.dump(project_map, f, indent=2)
                    except OSError:
                        pass
                    return mapped
        except (OSError, json.JSONDecodeError, AttributeError):
            pass

    # 3. Git remote slug
    try:
        result = subprocess.run(
            ["git", "remote", "get-url", "origin"],
            capture_output=True,
            text=True,
            check=True,
            cwd=cwd,
        )
        remote_url = result.stdout.strip()
        if remote_url:
            slug = _remote_url_to_slug(remote_url)
            if slug:
                return slug
    except (subprocess.CalledProcessError, OSError):
        pass

    # 4. 兜底: cwd 的 basename
    return os.path.basename(cwd) or "unknown"


def resolve_branch(cwd: str | None = None) -> str:
    if cwd is None:
        cwd = os.getcwd()
    try:
        result = subprocess.run(
            ["git", "branch", "--show-current"],
            capture_output=True,
            text=True,
            check=True,
            cwd=cwd,
        )
        branch = result.stdout.strip()
        return branch if branch else "unknown"
    except (subprocess.CalledProcessError, OSError):
        return "unknown"


def save_project_mapping(cwd: str, agent_id: str) -> None:
    """把 cwd -> agent_id(以及 remote hash key -> agent_id)写入 ~/.memgo/project_map.json。"""
    memgo_dir = os.path.expanduser("~/.memgo")
    os.makedirs(memgo_dir, exist_ok=True)
    map_path = os.path.join(memgo_dir, "project_map.json")
    project_map: dict[str, str] = {}
    if os.path.isfile(map_path):
        try:
            with open(map_path) as f:
                project_map = json.load(f)
        except (OSError, json.JSONDecodeError):
            project_map = {}
    project_map[cwd] = agent_id
    # 同时写入 remote hash key, 让映射在文件夹移动/改名后仍有效
    remote_key = _remote_hash_key(cwd)
    if remote_key:
        project_map[remote_key] = agent_id
    with open(map_path, "w") as f:
        json.dump(project_map, f, indent=2)


def _remote_hash_key(cwd: str | None = None) -> str:
    """基于 git remote URL 返回稳定 key, 形如 ``remote:<sha256(url)[:16]>``。"""
    if cwd is None:
        cwd = os.getcwd()
    try:
        result = subprocess.run(
            ["git", "config", "--get", "remote.origin.url"],
            capture_output=True,
            text=True,
            check=True,
            cwd=cwd,
        )
        url = result.stdout.strip()
        if not url:
            return ""
        digest = hashlib.sha256(url.encode()).hexdigest()[:16]
        return f"remote:{digest}"
    except (subprocess.CalledProcessError, OSError):
        return ""


def _remote_url_to_slug(url: str) -> str:
    """把 git remote URL 转成确定性 slug。

    处理:
      - HTTPS:  https://github.com/owner/repo.git
      - SSH:    git@github.com:owner/repo.git
      - SSH:    git@github.com-alias:owner/repo.git  (自定义 host 别名)
      - ssh://: ssh://git@github.com/owner/repo.git
      - git://: git://github.com/owner/repo.git
    """
    slug = url.strip()
    # 去掉 .git 后缀
    if slug.endswith(".git"):
        slug = slug[:-4]
    # 去掉协议前缀
    for prefix in ("https://", "http://", "ssh://", "git://"):
        if slug.startswith(prefix):
            slug = slug[len(prefix):]
            break
    else:
        # git@ 风格(无协议前缀)
        slug = re.sub(r"^git@", "", slug)
    # 把第一个冒号(SSH host:path 分隔符)换成 /
    slug = slug.replace(":", "/", 1)
    # 按 / 拆分, 取最后两段(owner, repo)
    parts = [p for p in slug.split("/") if p]
    if len(parts) >= 2:
        owner, repo = parts[-2], parts[-1]
        slug = f"{owner}-{repo}"
    elif parts:
        slug = parts[-1]
    else:
        return ""
    # 剩余的 / 和 : 换成 -
    slug = slug.replace("/", "-").replace(":", "-")
    return slug
