"""验证 platform_backend_options 的行为与兼容性。"""

from __future__ import annotations

from unittest.mock import patch

from memgo_cli.backend.platform import PlatformBackend
from memgo_cli.backend.types import BackendContext
from memgo_cli.config.models import PlatformConfig
from memgo_cli.runtime.state import caller_type, capture_notice


def _make_backend() -> PlatformBackend:
    return PlatformBackend(
        PlatformConfig(api_key="test-key", base_url="https://api.memgo.ai"),
        BackendContext(caller_type, capture_notice),
    )


class TestAddOptions:
    def test_new_fields_and_existing_fields_land_in_payload_together(self):
        backend = _make_backend()
        with patch.object(backend, "_request", return_value={"results": []}) as mock_request:
            backend.add(
                content="hello",
                user_id="alice",
                metadata={"source": "test"},
                expires="2099-01-01",
                custom_instructions="Extract only preferences.",
                agent_custom_instructions="Extract only tool outcomes.",
                custom_categories=[{"prefs": "user preferences"}],
                structured_data_schema={"type": "object"},
                timestamp=1700000000,
                messages=None,
                agent_id=None,
                app_id=None,
                run_id=None,
                immutable=False,
                infer=True,
            )
        payload = mock_request.call_args.kwargs["json"]
        assert payload["custom_instructions"] == "Extract only preferences."
        assert payload["agent_custom_instructions"] == "Extract only tool outcomes."
        assert payload["custom_categories"] == [{"prefs": "user preferences"}]
        assert payload["structured_data_schema"] == {"type": "object"}
        assert payload["timestamp"] == 1700000000
        assert payload["metadata"] == {"source": "test"}
        assert payload["expiration_date"] == "2099-01-01"

    def test_omitted_fields_are_absent_from_payload(self):
        backend = _make_backend()
        with patch.object(backend, "_request", return_value={"results": []}) as mock_request:
            backend.add(
                content="hello",
                user_id="alice",
                messages=None,
                agent_id=None,
                app_id=None,
                run_id=None,
                metadata=None,
                immutable=False,
                infer=True,
                expires=None,
                custom_instructions=None,
                agent_custom_instructions=None,
                custom_categories=None,
                structured_data_schema=None,
                timestamp=None,
            )
        payload = mock_request.call_args.kwargs["json"]
        assert "custom_instructions" not in payload
        assert "agent_custom_instructions" not in payload
        assert "custom_categories" not in payload
        assert "structured_data_schema" not in payload
        assert "timestamp" not in payload


class TestSearchOptions:
    def test_show_expired_reference_date_latest_only_reach_payload(self):
        backend = _make_backend()
        with patch.object(backend, "_request", return_value=[]) as mock_request:
            backend.search(
                "query",
                show_expired=True,
                reference_date="2024-01-01",
                latest_only=True,
                user_id=None,
                agent_id=None,
                app_id=None,
                run_id=None,
                top_k=10,
                threshold=0.3,
                rerank=False,
                keyword=False,
                filters=None,
                fields=None,
            )
        payload = mock_request.call_args.kwargs["json"]
        assert payload["show_expired"] is True
        assert payload["reference_date"] == "2024-01-01"
        assert payload["latest_only"] is True

    def test_keyword_and_fields_reach_payload(self):
        backend = _make_backend()
        with patch.object(backend, "_request", return_value=[]) as mock_request:
            backend.search(
                "query",
                keyword=True,
                fields=["memory", "score"],
                user_id=None,
                agent_id=None,
                app_id=None,
                run_id=None,
                top_k=10,
                threshold=0.3,
                rerank=False,
                filters=None,
                show_expired=False,
                reference_date=None,
                latest_only=False,
            )
        payload = mock_request.call_args.kwargs["json"]
        assert payload["keyword_search"] is True
        assert payload["fields"] == ["memory", "score"]

    def test_keyword_and_fields_omitted_are_absent_from_payload(self):
        backend = _make_backend()
        with patch.object(backend, "_request", return_value=[]) as mock_request:
            backend.search(
                "query",
                user_id=None,
                agent_id=None,
                app_id=None,
                run_id=None,
                top_k=10,
                threshold=0.3,
                rerank=False,
                keyword=False,
                filters=None,
                fields=None,
                show_expired=False,
                reference_date=None,
                latest_only=False,
            )
        payload = mock_request.call_args.kwargs["json"]
        assert "keyword_search" not in payload
        assert "fields" not in payload


class TestListOptions:
    def test_show_expired_and_latest_only_are_top_level_not_in_filters(self):
        backend = _make_backend()
        with patch.object(backend, "_request", return_value=[]) as mock_request:
            backend.list_memories(
                user_id="alice",
                show_expired=True,
                latest_only=True,
                agent_id=None,
                app_id=None,
                run_id=None,
                page=1,
                page_size=100,
                category=None,
                after=None,
                before=None,
            )
        payload = mock_request.call_args.kwargs["json"]
        assert payload["show_expired"] is True
        assert payload["latest_only"] is True
        assert "show_expired" not in payload.get("filters", {})
        assert "latest_only" not in payload.get("filters", {})


class TestUpdateOptions:
    def test_expires_and_timestamp_reach_payload(self):
        backend = _make_backend()
        with patch.object(backend, "_request", return_value={}) as mock_request:
            backend.update(
                "mem-123",
                expiration_date="2099-01-01",
                timestamp=1700000000,
                content=None,
                metadata=None,
            )
        payload = mock_request.call_args.kwargs["json"]
        assert payload["expiration_date"] == "2099-01-01"
        assert payload["timestamp"] == 1700000000


class TestDeleteOptions:
    def test_delete_linked_is_a_query_param_not_json_body(self):
        backend = _make_backend()
        with patch.object(backend, "_request", return_value={}) as mock_request:
            backend.delete(
                memory_id="mem-123",
                delete_linked=True,
                all=False,
                user_id=None,
                agent_id=None,
                app_id=None,
                run_id=None,
            )
        call = mock_request.call_args
        assert call.kwargs["params"]["delete_linked"] == "true"
        assert call.kwargs.get("json") is None
