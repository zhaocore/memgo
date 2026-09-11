"""用 ContextVar 隔离命令状态，避免同进程多次调用相互污染。"""

from __future__ import annotations

from collections.abc import Callable
from contextlib import ExitStack
from contextvars import ContextVar, Token
from dataclasses import dataclass, replace

from rich.console import Console


@dataclass(frozen=True)
class InvocationState:
    """单次调用的输出、身份和资源上下文。"""

    agent_mode: bool = False
    current_command: str = ""
    pending_notice: str = ""
    validated_email: str | None = None
    resources: ExitStack | None = None


# 默认值是冻结模型且不持有资源；所有更新均替换对象。
_state: ContextVar[InvocationState] = ContextVar("memgo_invocation", default=InvocationState())  # noqa: B039


def start_invocation(agent_mode: bool) -> Token[InvocationState]:
    """建立独立上下文，返回用于恢复外层状态的令牌。"""
    return _state.set(InvocationState(agent_mode=agent_mode, resources=ExitStack()))


def finish_invocation(token: Token[InvocationState]) -> None:
    """关闭连接并展示待处理通知，最后恢复外层调用状态。"""
    state = _state.get()
    try:
        if state.resources is not None:
            state.resources.close()
        if state.pending_notice and not state.agent_mode:
            Console(stderr=True).print(f"\n[yellow]🔔 {state.pending_notice}[/yellow]\n")
    finally:
        _state.reset(token)


def register_cleanup(action: Callable[[], None]) -> None:
    """为本次调用登记资源关闭操作。"""
    resources = _state.get().resources
    if resources is None:
        raise RuntimeError("CLI invocation context is missing")
    resources.callback(action)


def is_agent_mode() -> bool:
    """读取本次调用的代理输出开关。"""
    return _state.get().agent_mode


def set_agent_mode(val: bool) -> None:
    """替换本次调用的代理输出开关。"""
    _state.set(replace(_state.get(), agent_mode=val))


def get_current_command() -> str:
    """读取当前命令名。"""
    return _state.get().current_command


def set_current_command(name: str) -> None:
    """记录当前命令名供错误信封使用。"""
    _state.set(replace(_state.get(), current_command=name))


def capture_notice(notice: str | None) -> None:
    """缓存本次调用的最新非空平台通知。"""
    if notice:
        _state.set(replace(_state.get(), pending_notice=notice))


def take_notice() -> str:
    """取出平台通知并清除缓存。"""
    state = _state.get()
    _state.set(replace(state, pending_notice=""))
    return state.pending_notice


def validated_email() -> str | None:
    """读取当前调用已验证的邮箱。"""
    return _state.get().validated_email


def set_validated_email(email: str) -> None:
    """保存当前调用已验证的邮箱。"""
    _state.set(replace(_state.get(), validated_email=email))


def caller_type() -> str:
    """提供后端请求使用的调用方类型。"""
    return "agent" if is_agent_mode() else "user"
