"""后端端口与调用依赖，不导入具体连接器。"""

from __future__ import annotations

from abc import ABC, abstractmethod
from collections.abc import Callable
from dataclasses import dataclass

from memgo_cli.backend.json import JsonObject


@dataclass(frozen=True)
class BackendContext:
    """由入口注入的调用方身份与通知接收函数。"""

    caller_type: Callable[[], str]
    notice: Callable[[str | None], None]


class Backend(ABC):
    """记忆后端的抽象端口。"""

    @abstractmethod
    def close(self) -> None:
        """关闭连接器持有的资源。"""
        ...

    @abstractmethod
    def ping(self, timeout: float | None) -> JsonObject:
        """检查服务连通性与凭据。"""
        ...

    @abstractmethod
    def add(
        self,
        content: str | None,
        messages: list[JsonObject] | None,
        *,
        user_id: str | None,
        agent_id: str | None,
        app_id: str | None,
        run_id: str | None,
        metadata: JsonObject | None,
        immutable: bool,
        infer: bool,
        expires: str | None,
        custom_instructions: str | None,
        agent_custom_instructions: str | None,
        custom_categories: list[JsonObject] | None,
        structured_data_schema: JsonObject | None,
        timestamp: int | None,
    ) -> JsonObject | list[JsonObject]:
        """解析内容、消息和文件选项并添加记忆。"""
        ...

    @abstractmethod
    def search(
        self,
        query: str,
        *,
        user_id: str | None,
        agent_id: str | None,
        app_id: str | None,
        run_id: str | None,
        top_k: int,
        threshold: float,
        rerank: bool,
        keyword: bool,
        filters: JsonObject | None,
        fields: list[str] | None,
        show_expired: bool,
        reference_date: str | None,
        latest_only: bool,
    ) -> list[JsonObject]:
        """解析检索选项并查询记忆。"""
        ...

    @abstractmethod
    def get(self, memory_id: str) -> JsonObject:
        """按 ID 获取单条记忆。"""
        ...

    @abstractmethod
    def list_memories(
        self,
        *,
        user_id: str | None,
        agent_id: str | None,
        app_id: str | None,
        run_id: str | None,
        page: int,
        page_size: int,
        category: str | None,
        after: str | None,
        before: str | None,
        show_expired: bool,
        latest_only: bool,
    ) -> list[JsonObject]:
        """读取指定范围的记忆列表。"""
        ...

    @abstractmethod
    def update(
        self,
        memory_id: str,
        content: str | None,
        metadata: JsonObject | None,
        *,
        expiration_date: str | None,
        timestamp: int | None,
    ) -> JsonObject:
        """解析更新选项并执行记忆更新。"""
        ...

    @abstractmethod
    def delete(
        self,
        memory_id: str | None,
        *,
        all: bool,
        user_id: str | None,
        agent_id: str | None,
        app_id: str | None,
        run_id: str | None,
        delete_linked: bool,
    ) -> JsonObject:
        """校验删除选项互斥关系并分发单条、范围或实体删除。"""
        ...

    @abstractmethod
    def delete_entities(
        self,
        *,
        user_id: str | None,
        agent_id: str | None,
        app_id: str | None,
        run_id: str | None,
    ) -> JsonObject:
        """删除指定实体及关联记忆，分别保留各实体的响应。"""
        ...

    @abstractmethod
    def status(
        self,
        *,
        user_id: str | None,
        agent_id: str | None,
    ) -> JsonObject:
        """通过 ping 接口检查连接和鉴权状态。"""
        ...

    @abstractmethod
    def entities(self, entity_type: str) -> list[JsonObject]:
        """读取实体列表并按类型筛选。"""
        ...

    @abstractmethod
    def list_events(self) -> list[JsonObject]:
        """读取最近的后台事件。"""
        ...

    @abstractmethod
    def get_event(self, event_id: str) -> JsonObject:
        """读取指定事件状态。"""
        ...
