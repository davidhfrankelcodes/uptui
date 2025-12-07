# uptui

Uptui is a terminal UI (TUI) service monitor inspired by uptime-kuma but running in the terminal.

This repository contains a minimal scaffold to start building a monitor TUI using Textual and async HTTP checks.

Quickstart (Windows PowerShell):

```powershell
# create venv
python -m venv .venv
.\.venv\Scripts\Activate.ps1
# install deps
pip install -r requirements.txt
# run the TUI
python -m uptui.cli --example
```

Press `r` inside the app to run a manual refresh of checks.

Next steps:
- add persistence, scheduler, alerts, and more monitor types.
