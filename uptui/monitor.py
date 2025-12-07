"""Monitor core: async HTTP checks."""
from __future__ import annotations

import time
import asyncio
import httpx


class MonitorManager:
    """Simple manager that runs HTTP checks for a list of monitors."""

    def __init__(self, monitors: list[dict] | None = None) -> None:
        self.monitors = monitors or []

    async def check_monitor(self, m: dict) -> dict:
        url = m.get("url")
        name = m.get("name", url)
        try:
            start = time.perf_counter()
            async with httpx.AsyncClient(timeout=10.0) as client:
                r = await client.get(url)
            latency = (time.perf_counter() - start) * 1000
            status = "up" if r.status_code < 400 else f"down ({r.status_code})"
            return {"name": name, "url": url, "status": status, "latency_ms": int(latency)}
        except Exception as e:  # pragma: no cover - network errors
            return {"name": name, "url": url, "status": f"error: {type(e).__name__}", "latency_ms": "-"}

    async def run_checks(self) -> list[dict]:
        tasks = [self.check_monitor(m) for m in self.monitors]
        return await asyncio.gather(*tasks)
