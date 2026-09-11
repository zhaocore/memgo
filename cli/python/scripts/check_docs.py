"""校验中文注释和文档摘要，补充 Ruff 尚不支持的中文规则。"""

from __future__ import annotations

import ast
import io
import re
import tokenize
from pathlib import Path

CHINESE = re.compile(r"[\u4e00-\u9fff]")
DIRECTIVE = re.compile(r"#\s*(?:noqa\b|type:\s*ignore\b|!|.*coding[:=])")


def check_file(path: Path) -> list[str]:
    """检查真实注释令牌和 docstring，不把用户文案当作注释。"""
    source = path.read_text(encoding="utf-8")
    failures: list[str] = []
    for token in tokenize.generate_tokens(io.StringIO(source).readline):
        if token.type != tokenize.COMMENT or DIRECTIVE.match(token.string):
            continue
        if re.search(r"[A-Za-z]", token.string) and not CHINESE.search(token.string):
            failures.append(f"{path}:{token.start[0]}: 注释必须使用中文")
    for node in ast.walk(ast.parse(source)):
        if not isinstance(node, (ast.Module, ast.ClassDef, ast.FunctionDef, ast.AsyncFunctionDef)):
            continue
        doc = ast.get_docstring(node)
        if doc is None:
            continue
        summary = doc.splitlines()[0]
        if not CHINESE.search(summary):
            failures.append(f"{path}:{node.body[0].lineno}: 文档摘要必须使用中文")
        if not summary.endswith(("。", "？", "！", ".", "?", "!")):
            failures.append(f"{path}:{node.body[0].lineno}: 文档摘要缺少句末标点")
    return failures


def main() -> None:
    """检查维护的源码、测试和脚本，发现违规时返回非零状态。"""
    root = Path(__file__).resolve().parents[1]
    paths = sorted(path for folder in ("src", "tests", "scripts") for path in (root / folder).rglob("*.py"))
    failures = [message for path in paths for message in check_file(path)]
    if failures:
        print("\n".join(failures))
        raise SystemExit(1)
    print(f"中文注释与文档检查通过：{len(paths)} 个文件")


if __name__ == "__main__":
    main()
