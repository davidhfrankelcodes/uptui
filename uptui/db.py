"""Simple async SQLite helpers for future persistence."""
from __future__ import annotations

import os
import aiosqlite

DB_PATH = os.path.join(os.path.dirname(__file__), "uptui.db")


async def init_db(path: str = DB_PATH) -> None:
    async with aiosqlite.connect(path) as db:
        await db.execute(
            """CREATE TABLE IF NOT EXISTS monitors (
            id INTEGER PRIMARY KEY,
            name TEXT,
            url TEXT
        )"""
        )
        await db.commit()


async def save_monitors(monitors: list[dict], path: str = DB_PATH) -> None:
    async with aiosqlite.connect(path) as db:
        await db.execute("DELETE FROM monitors")
        for m in monitors:
            await db.execute("INSERT INTO monitors (name, url) VALUES (?, ?)", (m.get("name"), m.get("url")))
        await db.commit()
