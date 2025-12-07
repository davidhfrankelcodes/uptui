"""Minimal Textual TUI for uptui."""
from __future__ import annotations

from textual.app import App, ComposeResult
from textual.widgets import Header, Footer, DataTable
from textual.binding import Binding
from textual import events
from .monitor import MonitorManager


class UptuiApp(App):
    """A tiny TUI that lists monitors and allows manual refresh (press `r`)."""

    CSS_PATH = None
    BINDINGS = [Binding("r", "refresh_checks", "Refresh checks")]

    def __init__(self, default_monitors: list[dict] | None = None, **kwargs) -> None:
        super().__init__(**kwargs)
        self.monitors = default_monitors or []
        self.manager = MonitorManager(self.monitors)
        self.table: DataTable | None = None

    def compose(self) -> ComposeResult:
        yield Header()
        yield Footer()
        yield DataTable(id="monitors_table")

    async def on_mount(self) -> None:
        self.table = self.query_one("#monitors_table", DataTable)
        # define columns and initial rows
        self.table.add_columns("Name", "Address", "Status", "Latency (ms)")
        for m in self.monitors:
            address = m.get("url") or (f"{m.get('host')}:{m.get('port')}" if m.get("host") and m.get("port") else "")
            self.table.add_row(m.get("name", ""), address, "unknown", "-")

        # perform an initial refresh so the UI isn't all 'unknown'
        await self.action_refresh_checks()

        # schedule periodic refreshes every 30 seconds
        self.set_interval(30, self.action_refresh_checks)

    async def action_refresh_checks(self) -> None:
        """Run asynchronous checks and update the table."""
        if not self.table:
            return
        results = await self.manager.run_checks()
        self.table.clear()
        for r in results:
            latency = r.get("latency_ms", "-")
            address = r.get("address", r.get("url", ""))
            self.table.add_row(r.get("name", ""), address, r.get("status", ""), str(latency))
