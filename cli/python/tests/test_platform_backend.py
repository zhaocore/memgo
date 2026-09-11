"""验证 platform_backend 的行为与兼容性。"""

from __future__ import annotations

from unittest.mock import patch

from memgo_cli.backend.platform import PlatformBackend
from memgo_cli.backend.types import BackendContext
from memgo_cli.config.models import PlatformConfig
from memgo_cli.runtime.state import caller_type, capture_notice


def _make_backend() -> PlatformBackend:
    # 密钥和地址仅用于构造客户端
    # 各测试替换请求方法，不发起真实网络请求
    return PlatformBackend(
        PlatformConfig(api_key="test-key", base_url="https://api.memgo.ai"),
        BackendContext(caller_type, capture_notice),
    )


class TestDeleteEntities:
    def test_multiple_entities_returns_all_results(self):
        backend = _make_backend()
        responses = {
            "/v2/entities/user/alice/": {"message": "user deleted"},
            "/v2/entities/agent/bob/": {"message": "agent deleted"},
        }
        with patch.object(backend, "_request") as mock_request:
            mock_request.side_effect = lambda method, path, **kw: responses[path]
            result = backend.delete_entities(
                user_id="alice", agent_id="bob", app_id=None, run_id=None
            )

        # 保留每个实体的响应，避免仅返回最后一项
        assert result == {
            "user": {"message": "user deleted"},
            "agent": {"message": "agent deleted"},
        }
        assert mock_request.call_count == 2

    def test_single_entity_keyed_by_type(self):
        backend = _make_backend()
        with patch.object(backend, "_request", return_value={"message": "user deleted"}):
            result = backend.delete_entities(
                user_id="alice", agent_id=None, app_id=None, run_id=None
            )
        assert result == {"user": {"message": "user deleted"}}

    def test_no_entities_raises(self):
        backend = _make_backend()
        import pytest

        with pytest.raises(ValueError):
            backend.delete_entities(user_id=None, agent_id=None, app_id=None, run_id=None)
