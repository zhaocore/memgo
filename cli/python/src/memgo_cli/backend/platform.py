"""MemGo Platform HTTP 连接器。"""

from __future__ import annotations

from urllib.parse import quote

import httpx

from memgo_cli import __version__
from memgo_cli.backend.http import read_response
from memgo_cli.backend.json import (
    JsonObject,
    JsonValue,
    json_object,
    json_records,
    json_result,
    json_string,
)
from memgo_cli.backend.types import Backend, BackendContext
from memgo_cli.config.models import PlatformConfig


def _encode_path_segment(value: object) -> str:
    """编码完整路径段，防止 ID 中的斜杠改变请求路径。"""
    return quote(str(value), safe="")


class PlatformBackend(Backend):
    """通过 HTTP 访问 MemGo Platform 的连接器。"""

    def __init__(self, config: PlatformConfig, context: BackendContext) -> None:
        """保存连接依赖并创建 HTTP 客户端。"""
        self._context = context
        self.config = config
        self.base_url = config.base_url.rstrip("/")
        self._client = httpx.Client(
            base_url=self.base_url,
            headers={
                "Authorization": f"Token {config.api_key}",
                "Content-Type": "application/json",
                "X-MemGo-Source": "cli",
                "X-MemGo-Client-Language": "python",
                "X-MemGo-Client-Version": __version__,
            },
            timeout=30.0,
        )

    def close(self) -> None:
        """释放 HTTP 连接池。"""
        self._client.close()

    def _request(
        self, method: str, path: str, *, json: JsonObject | None, params: dict[str, str] | None
    ) -> JsonValue:
        """执行请求并校验响应，通知经注入函数交给调用层。"""
        response = self._client.request(
            method,
            path,
            json=json,
            params=params,
            headers={"X-MemGo-Caller-Type": self._context.caller_type()},
        )
        data, notice = read_response(response)
        self._context.notice(notice)
        return data

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
        payload: JsonObject = {}

        if messages:
            payload["messages"] = messages
        elif content:
            payload["messages"] = [{"role": "user", "content": content}]

        if user_id:
            payload["user_id"] = user_id
        if agent_id:
            payload["agent_id"] = agent_id
        if app_id:
            payload["app_id"] = app_id
        if run_id:
            payload["run_id"] = run_id
        if metadata:
            payload["metadata"] = metadata
        if immutable:
            payload["immutable"] = True
        if not infer:
            payload["infer"] = False
        if expires:
            payload["expiration_date"] = expires
        if custom_instructions:
            payload["custom_instructions"] = custom_instructions
        if agent_custom_instructions:
            payload["agent_custom_instructions"] = agent_custom_instructions
        if custom_categories:
            payload["custom_categories"] = custom_categories
        if structured_data_schema:
            payload["structured_data_schema"] = structured_data_schema
        if timestamp is not None:
            payload["timestamp"] = timestamp
        payload["source"] = "CLI"

        return json_result(self._request("POST", "/v3/memories/add/", json=payload, params=None))

    def _build_filters(
        self,
        *,
        user_id: str | None,
        agent_id: str | None,
        app_id: str | None,
        run_id: str | None,
        extra_filters: JsonObject | None,
    ) -> JsonObject | None:
        """构造 v3 筛选对象。

        显式 filters 优先；否则实体 ID、日期和分类条件按 AND 合并。
        """
        # 显式筛选结构优先，直接使用调用方提供的 filters
        if extra_filters and ("AND" in extra_filters or "OR" in extra_filters):
            return extra_filters

        # 实体 ID 组成 AND 条件
        and_conditions: list[JsonObject] = []
        if user_id:
            and_conditions.append({"user_id": user_id})
        if agent_id:
            and_conditions.append({"agent_id": agent_id})
        if app_id:
            and_conditions.append({"app_id": app_id})
        if run_id:
            and_conditions.append({"run_id": run_id})

        # 追加日期、分类等筛选条件
        if extra_filters:
            for k, v in extra_filters.items():
                and_conditions.append({k: v})

        if len(and_conditions) == 1:
            return and_conditions[0]
        elif and_conditions:
            return {"AND": and_conditions}
        else:
            return None

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
        payload: JsonObject = {"query": query, "top_k": top_k, "threshold": threshold}

        api_filters = self._build_filters(
            user_id=user_id,
            agent_id=agent_id,
            app_id=app_id,
            run_id=run_id,
            extra_filters=filters,
        )
        if api_filters:
            payload["filters"] = api_filters
        if rerank:
            payload["rerank"] = True
        if keyword:
            payload["keyword_search"] = True
        if fields:
            payload["fields"] = fields
        if show_expired:
            payload["show_expired"] = True
        if reference_date is not None:
            payload["reference_date"] = reference_date
        if latest_only:
            payload["latest_only"] = True
        payload["source"] = "CLI"

        result = self._request("POST", "/v3/memories/search/", json=payload, params=None)
        return json_records(result)

    def get(self, memory_id: str) -> JsonObject:
        """按 ID 获取单条记忆。"""
        return json_object(
            self._request(
                "GET",
                f"/v1/memories/{_encode_path_segment(memory_id)}/",
                params={"source": "CLI"},
                json=None,
            )
        )

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
        payload: JsonObject = {}
        params = {"page": str(page), "page_size": str(page_size)}

        # 实体 ID 和日期条件统一放入 filters
        extra: JsonObject = {}
        if category:
            extra["categories"] = {"contains": category}
        if after:
            extra["created_at"] = {**json_object(extra.get("created_at", {})), "gte": after}
        if before:
            extra["created_at"] = {**json_object(extra.get("created_at", {})), "lte": before}

        api_filters = self._build_filters(
            user_id=user_id,
            agent_id=agent_id,
            app_id=app_id,
            run_id=run_id,
            extra_filters=extra if extra else None,
        )
        if api_filters:
            payload["filters"] = api_filters
        if show_expired:
            payload["show_expired"] = True
        if latest_only:
            payload["latest_only"] = True
        payload["source"] = "CLI"

        result = self._request("POST", "/v3/memories/", json=payload, params=params)
        return json_records(result)

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
        payload: JsonObject = {}
        if content:
            payload["text"] = content
        if metadata:
            payload["metadata"] = metadata
        if expiration_date:
            payload["expiration_date"] = expiration_date
        if timestamp is not None:
            payload["timestamp"] = timestamp
        payload["source"] = "CLI"
        return json_object(
            self._request(
                "PUT", f"/v1/memories/{_encode_path_segment(memory_id)}/", json=payload, params=None
            )
        )

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
        if all:
            params: dict[str, str] = {"source": "CLI"}
            if user_id:
                params["user_id"] = user_id
            if agent_id:
                params["agent_id"] = agent_id
            if app_id:
                params["app_id"] = app_id
            if run_id:
                params["run_id"] = run_id
            return json_object(self._request("DELETE", "/v1/memories/", params=params, json=None))
        elif memory_id:
            params = {"source": "CLI"}
            if delete_linked:
                params["delete_linked"] = "true"
            return json_object(
                self._request(
                    "DELETE",
                    f"/v1/memories/{_encode_path_segment(memory_id)}/",
                    params=params,
                    json=None,
                )
            )
        else:
            raise ValueError("Either memory_id or --all is required")

    def delete_entities(
        self,
        *,
        user_id: str | None,
        agent_id: str | None,
        app_id: str | None,
        run_id: str | None,
    ) -> JsonObject:
        # 使用 v2 实体删除路径
        """删除指定实体及关联记忆，分别保留各实体的响应。"""
        type_map = {
            "user": user_id,
            "agent": agent_id,
            "app": app_id,
            "run": run_id,
        }
        entities = {t: v for t, v in type_map.items() if v}
        if not entities:
            raise ValueError("At least one entity ID is required for delete_entities.")
        # 分别请求每个实体的 v2 删除接口
        # 按实体类型保留各响应
        # 避免多实体删除只剩最后一个结果
        results: JsonObject = {}
        for entity_type, entity_id in entities.items():
            results[entity_type] = self._request(
                "DELETE",
                f"/v2/entities/{_encode_path_segment(entity_type)}/{_encode_path_segment(entity_id)}/",
                params={"source": "CLI"},
                json=None,
            )
        return results

    def ping(self, timeout: float | None) -> JsonObject:
        """探测服务，提供超时时覆盖客户端默认值。"""
        if timeout is not None:
            response = self._client.get("/v1/ping/", timeout=timeout)
            data, notice = read_response(response)
            self._context.notice(notice)
            return json_object(data)
        return json_object(self._request("GET", "/v1/ping/", json=None, params=None))

    def status(
        self,
        *,
        user_id: str | None,
        agent_id: str | None,
    ) -> JsonObject:
        """通过 ping 接口检查连接和鉴权状态。"""
        try:
            self.ping(timeout=None)
            return {"connected": True, "backend": "platform", "base_url": self.base_url}
        except Exception as e:
            return {"connected": False, "backend": "platform", "error": str(e)}

    def entities(self, entity_type: str) -> list[JsonObject]:
        """读取实体列表并按类型筛选。"""
        result = self._request("GET", "/v1/entities/", json=None, params=None)
        items = json_records(result)
        # API 返回所有类型，在客户端按类型筛选
        type_map = {"users": "user", "agents": "agent", "apps": "app", "runs": "run"}
        target_type = type_map.get(entity_type)
        if target_type:
            items = [
                e
                for e in items
                if json_string(e.get("type", ""), "entity.type").lower() == target_type
            ]
        return items

    def list_events(self) -> list[JsonObject]:
        """读取最近的后台事件。"""
        result = self._request("GET", "/v1/events/", json=None, params=None)
        return json_records(result)

    def get_event(self, event_id: str) -> JsonObject:
        """读取指定事件状态。"""
        return json_object(
            self._request(
                "GET", f"/v1/event/{_encode_path_segment(event_id)}/", json=None, params=None
            )
        )
