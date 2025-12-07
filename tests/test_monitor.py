import pytest
import types
from uptui.monitor import MonitorManager
import asyncio


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


@pytest.mark.asyncio
async def test_tcp_check_success_and_failure(monkeypatch):
    # stub asyncio.open_connection to simulate success for port 22 and failure for other
    async def fake_open_connection_success(host, port):
        # return (reader, writer) tuple mimics
        class DummyWriter:
            def close(self):
                pass

            async def wait_closed(self):
                return None

        return (None, DummyWriter())

    async def fake_open_connection_fail(host, port):
        raise ConnectionRefusedError("refused")

    # First monitor: will 'succeed' (we'll patch for its host/port)
    # Second monitor: will fail
    mm = MonitorManager([
        {"name": "tcp-ok", "type": "tcp", "host": "127.0.0.1", "port": 22},
        {"name": "tcp-bad", "type": "tcp", "host": "127.0.0.1", "port": 65000},
    ])

    # monkeypatch open_connection: return success for port 22, fail otherwise
    async def open_conn_cond(host, port):
        if int(port) == 22:
            return await fake_open_connection_success(host, port)
        return await fake_open_connection_fail(host, port)

    monkeypatch.setattr(asyncio, "open_connection", open_conn_cond)

    res = await mm.run_checks()
    assert res[0]["status"] == "up"
    assert res[1]["status"].startswith("error:")
