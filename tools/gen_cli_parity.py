#!/usr/bin/env python3
"""内省 mem0 python CLI (typer→click) 生成命令×选项矩阵 golden。

用法: python3 tools/gen_cli_parity.py > tests/contract/cli_parity_golden.json
Go 侧 parity_test 对照: 不得有矩阵外命令/选项; 缺席项在测试内显式列明。
"""
import json
import sys
from pathlib import Path

REPO = Path("/Users/johnson/work/github/mem0")
sys.path.insert(0, str(REPO / "cli" / "python" / "src"))


def main() -> None:
    import click
    import typer.main

    from mem0_cli.app import app

    click_app = typer.main.get_command(app)
    assert isinstance(click_app, click.Group)

    def walk(group) -> dict:
        out = {}
        for name, cmd in sorted(group.commands.items()):
            entry: dict = {}
            args = []
            opts = []
            for p in cmd.params:
                if isinstance(p, click.Argument):
                    args.append(p.name)
                elif isinstance(p, click.Option):
                    opts.extend(p.opts)
            if args:
                entry["arguments"] = args
            if opts:
                entry["options"] = opts
            if isinstance(cmd, click.Group):
                entry["subcommands"] = walk(cmd)
            out[name] = entry
        return out

    print(json.dumps({"name": click_app.name, "commands": walk(click_app)},
                     ensure_ascii=False, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
