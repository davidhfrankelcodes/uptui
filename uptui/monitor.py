"""Monitor core: async HTTP checks."""
from __future__ import annotations

import time
import asyncio
import socket
import httpx


class MonitorManager:
    """Simple manager that runs HTTP checks for a list of monitors."""

    def __init__(self, monitors: list[dict] | None = None) -> None:
        self.monitors = monitors or []

    async def check_monitor(self, m: dict) -> dict:
        """Perform a check for the given monitor dict.

        Supported monitor types:
        - HTTP (default): provide `url`.
        - TCP: provide `type: tcp`, `host`, and `port`.
        """
        name = m.get("name") or m.get("url") or "unknown"
        mtype = (m.get("type") or "http").lower()

        if mtype == "tcp":
            host = m.get("host")
            port = m.get("port")
            timeout = float(m.get("timeout", 5.0))
            if not host or not port:
                return {"name": name, "status": "error: missing host/port", "latency_ms": "-"}

            start = time.perf_counter()
            try:
                coro = asyncio.open_connection(host, int(port))
                reader_writer = await asyncio.wait_for(coro, timeout=timeout)
                # close writer if available
                try:
                    writer = reader_writer[1]
                    writer.close()
                    # await writer.wait_closed() only available in Python 3.7+ for streams
                    if hasattr(writer, "wait_closed"):
                        await writer.wait_closed()
                except Exception:
                    pass
                latency = (time.perf_counter() - start) * 1000
                return {"name": name, "status": "up", "latency_ms": int(latency), "address": f"{host}:{port}"}
            except Exception as e:  # pragma: no cover - network errors
                return {"name": name, "status": f"error: {type(e).__name__}", "latency_ms": "-", "address": f"{host}:{port}"}

        # fallback/HTTP check
        url = m.get("url")
        if not url:
            return {"name": name, "status": "error: missing url", "latency_ms": "-", "address": ""}

        try:
            start = time.perf_counter()
            async with httpx.AsyncClient(timeout=10.0) as client:
                r = await client.get(url)
            latency = (time.perf_counter() - start) * 1000
            status = "up" if r.status_code < 400 else f"down ({r.status_code})"
            return {"name": name, "url": url, "status": status, "latency_ms": int(latency), "address": url}
        except Exception as e:  # pragma: no cover - network errors
            return {"name": name, "url": url, "status": f"error: {type(e).__name__}", "latency_ms": "-", "address": url}

    async def run_checks(self) -> list[dict]:
        tasks = [self.check_monitor(m) for m in self.monitors]
        return await asyncio.gather(*tasks)
