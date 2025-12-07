"""Command-line entry for uptui."""
from __future__ import annotations

import argparse
from .app import UptuiApp


def main() -> None:
    parser = argparse.ArgumentParser(prog="uptui", description="Terminal uptime monitor")
    parser.add_argument("--example", action="store_true", help="Run with example monitor")
    args = parser.parse_args()

    monitors = []
    if args.example:
        monitors = [
            {"name": "Google", "url": "https://www.google.com"},
            {"name": "GitHub", "url": "https://github.com"},
            {"name": "HTTPStat 200", "url": "https://httpstat.us/200"},
            {"name": "HTTPStat 503", "url": "https://httpstat.us/503"},
        ]
    else:
        monitors = [{"name": "GitHub", "url": "https://github.com"}]

    app = UptuiApp(default_monitors=monitors)
    app.run()


if __name__ == "__main__":
    main()
