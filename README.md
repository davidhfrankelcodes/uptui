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
# run the TUI (reads monitors from ./config.yaml by default)
python -m uptui.cli --config config.yaml
```

Press `r` inside the app to run a manual refresh of checks.

Next steps:

Configuration
 - Place your monitor definitions in `config.yaml` (or pass a path with `--config`).
 - Example `config.yaml` structure:

 ```yaml
 monitors:
   - name: MySite
     url: https://example.com
 ```

 Press `r` inside the app to run a manual refresh of checks. The app performs an initial automatic refresh when it starts and will refresh every 30 seconds.
