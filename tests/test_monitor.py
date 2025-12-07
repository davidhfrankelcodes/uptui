import pytest
import types
from uptui.monitor import MonitorManager


class AsyncClientStub:
    """A very small AsyncClient stub for testing."""

    def __init__(self, *args, **kwargs):
        pass

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc, tb):
        return False

    async def get(self, url):
        # return an object with status_code depending on url
        class R:
            def __init__(self, status_code):
                self.status_code = status_code

        if "503" in url:
            return R(503)
        if "timeout" in url:
            raise Exception("connect timeout")
        return R(200)


@pytest.mark.asyncio
async def test_check_monitor_success(monkeypatch):
    mm = MonitorManager([{"name": "ok", "url": "https://example.com"}])

    monkeypatch.setattr("httpx.AsyncClient", AsyncClientStub)

    res = await mm.run_checks()
    assert isinstance(res, list)
    assert res[0]["status"] == "up"
    assert isinstance(res[0]["latency_ms"], int)


@pytest.mark.asyncio
async def test_check_monitor_down_and_error(monkeypatch):
    mm = MonitorManager([
        {"name": "srv503", "url": "https://httpstat.us/503"},
        {"name": "err", "url": "https://timeout.example/timeout"},
    ])

    monkeypatch.setattr("httpx.AsyncClient", AsyncClientStub)

    res = await mm.run_checks()
    assert res[0]["status"].startswith("down") or "503" in res[0]["status"]
    # error should be caught and reported as error: ExceptionName
    assert res[1]["status"].startswith("error:")
