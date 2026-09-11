from unittest.mock import MagicMock

import httpx

from memgo_cli.backend.platform import PlatformBackend
from memgo_cli.backend.types import BackendContext
from memgo_cli.runtime.state import caller_type, capture_notice


def _backend(sample_config):
    backend = PlatformBackend(sample_config.platform, BackendContext(caller_type, capture_notice))
    backend._client = MagicMock()
    backend._client.request.side_effect = lambda method, path, **kwargs: httpx.Response(
        200, json={"message": "ok"}, request=httpx.Request(method, "https://api.memgo.ai" + path)
    )
    return backend


def test_memory_id_path_segments_are_encoded(sample_config):
    backend = _backend(sample_config)

    backend.get("mem/a?b#c")
    backend.update(
        "mem/a?b#c", content="updated", metadata=None, expiration_date=None, timestamp=None
    )
    backend.delete(
        "mem/a?b#c",
        all=False,
        user_id=None,
        agent_id=None,
        app_id=None,
        run_id=None,
        delete_linked=False,
    )

    paths = [call.args[1] for call in backend._client.request.call_args_list]
    assert paths == [
        "/v1/memories/mem%2Fa%3Fb%23c/",
        "/v1/memories/mem%2Fa%3Fb%23c/",
        "/v1/memories/mem%2Fa%3Fb%23c/",
    ]


def test_entity_and_event_path_segments_are_encoded(sample_config):
    backend = _backend(sample_config)

    backend.delete_entities(user_id="org/team?active#frag", agent_id=None, app_id=None, run_id=None)
    backend.get_event("evt/a?b#c")

    paths = [call.args[1] for call in backend._client.request.call_args_list]
    assert paths == [
        "/v2/entities/user/org%2Fteam%3Factive%23frag/",
        "/v1/event/evt%2Fa%3Fb%23c/",
    ]
