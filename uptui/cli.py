"""Command-line entry for uptui.

The CLI no longer embeds default monitors. Monitors are loaded from a
YAML config file supplied with `--config` or `./config.yaml` by default.
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path
from .app import UptuiApp

try:
    import yaml  # type: ignore
except Exception:  # pragma: no cover - environment may not have yaml installed
    yaml = None


def load_config(path: Path) -> dict:
    if not path.exists():
        return {}
    if yaml is None:
        print("PyYAML is not installed; cannot read YAML config.", file=sys.stderr)
        return {}
    try:
        data = yaml.safe_load(path.read_text(encoding="utf-8")) or {}
        if not isinstance(data, dict):
            return {}
        return data
    except Exception as e:
        print(f"Failed to read config {path}: {e}", file=sys.stderr)
        return {}


def main() -> None:
    parser = argparse.ArgumentParser(prog="uptui", description="Terminal uptime monitor")
    parser.add_argument("--config", "-c", default="config.yaml", help="Path to YAML config file")
    args = parser.parse_args()

    path = Path(args.config)
    cfg = load_config(path)
    monitors = cfg.get("monitors", []) or []

    app = UptuiApp(default_monitors=monitors)
    app.run()


if __name__ == "__main__":
    main()
