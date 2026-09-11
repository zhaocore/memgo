"""终端密钥输入及回显控制。"""

from __future__ import annotations

import sys

from rich.console import Console

console = Console()
err_console = Console(stderr=True)


def _prompt_secret(label: str) -> str:
    """读取密钥，以星号代替输入字符并在退出时恢复终端设置。"""
    chars: list[str] = []

    if sys.platform == "win32":
        import msvcrt

        sys.stdout.write(label)
        sys.stdout.flush()

        while True:
            ch = msvcrt.getwch()
            if ch in ("\r", "\n"):
                sys.stdout.write("\n")
                sys.stdout.flush()
                break
            if ch == "\x03":
                raise KeyboardInterrupt
            if ch in ("\x08", "\x7f"):  # 退格
                if chars:
                    chars.pop()
                    sys.stdout.write("\b \b")
                    sys.stdout.flush()
            else:
                chars.append(ch)
                sys.stdout.write("*")
                sys.stdout.flush()
    else:
        import termios
        import tty

        fd = sys.stdin.fileno()
        old_settings = termios.tcgetattr(fd)
        try:
            tty.setraw(fd)
            sys.stdout.write(label)
            sys.stdout.flush()
            while True:
                ch = sys.stdin.read(1)
                if not ch or ch == "\x04":
                    raise EOFError("Secret input ended before confirmation")
                if ch in ("\r", "\n"):
                    sys.stdout.write("\r\n")
                    sys.stdout.flush()
                    break
                if ch == "\x03":
                    raise KeyboardInterrupt
                if ch in ("\x7f", "\x08"):  # 退格或删除
                    if chars:
                        chars.pop()
                        sys.stdout.write("\b \b")
                        sys.stdout.flush()
                elif ch == "\x15":  # Ctrl+U 清空当前输入
                    sys.stdout.write("\b \b" * len(chars))
                    sys.stdout.flush()
                    chars = []
                elif ch >= " ":  # 忽略其他控制字符
                    chars.append(ch)
                    sys.stdout.write("*")
                    sys.stdout.flush()
        finally:
            termios.tcsetattr(fd, termios.TCSADRAIN, old_settings)

    return "".join(chars)
