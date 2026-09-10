#!/usr/bin/env python3
"""导出 mem0 prompts.py 的字符串常量为 JSON (供 Go parity_test 逐字节比对)。"""
import json
import re
import sys

def main() -> None:
    src = sys.argv[1]
    namespace: dict = {}
    with open(src, "r", encoding="utf-8") as f:
        exec(compile(f.read(), src, "exec"), namespace)
    consts = {k: v for k, v in namespace.items() if re.fullmatch(r"[A-Z][A-Z0-9_]*", k) and isinstance(v, str)}
    json.dump(consts, sys.stdout, ensure_ascii=False, sort_keys=True)

if __name__ == "__main__":
    main()
