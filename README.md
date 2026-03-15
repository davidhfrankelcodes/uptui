# uptui

A local uptime monitor — like Uptime Kuma, but as a daemon on your machine with a terminal UI.

The daemon runs in the background and continuously checks your HTTP endpoints and TCP ports. When you open `uptui`, it connects to the already-running daemon and shows you what it has been tracking.

```
uptui                                          ● 3 up  ● 1 down

  STATUS  NAME                   TYPE   LATENCY   UPTIME   HISTORY
  ──────────────────────────────────────────────────────────────────
▶ ● UP    github.com             HTTP    45 ms    99.9%    ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓
  ● UP    api.myapp.com          HTTP   120 ms    98.5%    ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓
  ● DOWN  postgres:5432          TCP      -        95.0%   ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓
  ● UP    smtp.myapp.com:25      TCP     33 ms   100.0%    ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓
  ──────────────────────────────────────────────────────────────────
  a dd  e dit  d elete  p ause  s:name  f:all  ↑↓ nav  ↵ detail  r refresh  q uit
```

On terminals ≥ 100 columns wide, 7-day and 30-day uptime columns appear automatically.

## Docker

Run the daemon in a container with your config and history on the host:

```bash
# Start the daemon (detached)
docker compose up -d

# Edit config/monitors.toml to add monitors — daemon picks up changes in ≤ 5 s
# Or use the CLI via exec:
docker compose exec daemon uptui add https://github.com
docker compose exec daemon uptui status

# Run the TUI from your local machine (connects to the container via 127.0.0.1:29374)
uptui
```

The `config/` directory is bind-mounted so you can edit `monitors.toml` directly.
Check history is stored in the `uptui-data` named volume.

**To reach hosts on your machine from inside the container** use `host.docker.internal`
instead of `localhost` in your monitor targets (e.g. `host.docker.internal:5432`).

## Install

Requires [Go 1.21+](https://go.dev/dl/).

```bash
git clone https://github.com/you/uptui
cd uptui
go install ./cmd/uptui
```

Or build a binary in place:

```bash
go build -o uptui ./cmd/uptui
```

## Quick start

```bash
# Add a monitor
uptui add https://github.com
uptui add --name "My API" https://api.myapp.com/health
uptui add --type tcp --name "Postgres" localhost:5432

# Open the TUI (daemon auto-starts on first launch)
uptui
```

The daemon starts automatically when you first run `uptui`. It keeps running in the background after you close the TUI.

## Commands

```
uptui                             Open the TUI (auto-starts daemon)
uptui daemon                      Run the daemon in the foreground
uptui stop                        Stop the background daemon
uptui status                      Print current status to stdout
uptui add TARGET [flags]          Add a monitor
uptui edit NAME [flags]           Edit an existing monitor
uptui theme                       Show current theme and available themes
uptui theme NAME                  Set TUI color theme

Flags for add:
  --name, -n NAME                 Display name  (default: TARGET)
  --type, -t http|tcp             Monitor type  (default: http)
  --interval, -i SECONDS          Check interval (default: 60, min: 10)

Flags for edit:
  --name NEWNAME                  Rename the monitor
  --target URL                    New target URL or host:port
  --type http|tcp                 Change monitor type
  --interval SECONDS              New check interval
  --timeout SECONDS               New timeout
```

## Config file

Monitors are stored in `~/.uptui/monitors.toml`. You can edit this file directly in any text editor — the daemon picks up changes within 5 seconds without restarting.

```toml
[[monitor]]
name     = "GitHub"
type     = "http"
target   = "https://github.com"

[[monitor]]
name     = "Postgres"
type     = "tcp"
target   = "localhost:5432"
interval = 30

[[monitor]]
name   = "Legacy API"
type   = "http"
target = "https://api.legacy.com/health"
active = false
```

**Field defaults** — omitted when at their defaults to keep the file clean:

| Field | Default | Notes |
|-------|---------|-------|
| `interval` | `60` | Seconds between checks (min 10) |
| `timeout` | `30` | Seconds before a check times out |
| `active` | `true` | Set `false` to pause without deleting |

The CLI (`uptui add`, `uptui edit`) and TUI (`a`, `e`) write back to this file automatically.

## TUI keybindings

### Dashboard

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `enter` | Open detail view |
| `a` | Add a new monitor |
| `e` | Edit selected monitor (opens pre-filled form) |
| `d` | Delete selected monitor (confirmation required) |
| `p` | Pause / resume selected monitor |
| `s` | Cycle sort order (name → status → uptime → name) |
| `f` | Cycle filter (all → down → problems → all) |
| `r` | Force refresh |
| `q` / `ctrl+c` | Quit |

The footer always shows the active sort and filter, e.g. `s:status  f:down`.

### Detail view

| Key | Action |
|-----|--------|
| `↑` / `k` | Scroll to newer checks |
| `↓` / `j` | Scroll to older checks |
| `esc` / `backspace` | Back to dashboard |
| `q` / `ctrl+c` | Quit |

### Add / Edit form

| Key | Action |
|-----|--------|
| `tab` / `↓` | Next field |
| `shift+tab` / `↑` | Previous field |
| `enter` | Next field / submit (on last field) |
| `esc` | Cancel |

When editing, submitting the form shows a confirmation prompt (`Save changes to "name"?`). Press `y` to confirm or any other key to cancel and return to the form.

## Themes

7 built-in color themes, selectable via CLI or by editing `~/.uptui/settings.toml`:

```bash
uptui theme dracula       # set theme for this machine
uptui theme               # show current theme and available names
```

Or edit `~/.uptui/settings.toml` directly:

```toml
theme = "nord"
```

| Name | Style |
|------|-------|
| `default` | ANSI bright terminal colors |
| `dracula` | Purple/cyan/green/pink on dark |
| `nord` | Arctic blues and snow on dark slate |
| `solarized` | Earthy green/cyan on dark teal |
| `monokai` | Vivid yellow/green/pink on near-black |
| `gruvbox` | Warm oranges/greens on dark brown |
| `monochrome` | Bold/dim only — works on any terminal |

When `settings.toml` is absent or theme is `"default"`, ANSI colors are used.

## Monitor types

| Type | Target format | Example |
|------|---------------|---------|
| `http` | Full URL | `https://example.com/health` |
| `tcp` | `host:port` | `localhost:5432` |

HTTP monitors follow redirects (up to 10) and report down for any 4xx/5xx response. TCP monitors report up as soon as the connection is established.

## Data

All data is stored in `~/.uptui/`:

| File | Contents |
|------|----------|
| `monitors.toml` | Monitor definitions — the canonical config, hand-editable |
| `settings.toml` | User preferences: `theme` (written by `uptui theme NAME`) |
| `history.json` | Check history (last 500 results per monitor, keyed by name) |
| `daemon.pid` | PID of the running daemon (deleted on clean stop) |
| `daemon.log` | Daemon stdout/stderr when auto-started |

Both `monitors.toml` and `history.json` are written atomically (write to `.tmp`, then rename) so a crash mid-write cannot corrupt them.

### Environment variable overrides

| Variable | Default | Purpose |
|----------|---------|---------|
| `UPTUI_DATA_DIR` | `~/.uptui` | Directory for `history.json`, `daemon.pid`, `daemon.log` |
| `UPTUI_CONFIG_DIR` | same as `UPTUI_DATA_DIR` | Directory for `monitors.toml`, `settings.toml` |
| `UPTUI_LISTEN_ADDR` | `127.0.0.1:29374` | Address the daemon IPC server binds to |

The Docker setup sets `UPTUI_LISTEN_ADDR=0.0.0.0:29374` so the published port is reachable from the host.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  uptui (TUI process)                                            │
│  bubbletea model ──── polls every 5s ────┐                      │
└──────────────────────────────────────────┼──────────────────────┘
                                           │ JSON over TCP
                                           │ 127.0.0.1:29374
┌──────────────────────────────────────────┼──────────────────────┐
│  uptui daemon                            │                      │
│  IPC server ◄────────────────────────────┘                      │
│      │  writes on every mutation                                │
│      ▼                                                          │
│  ~/.uptui/monitors.toml ◄──── user edits (picked up in ≤5s)    │
│      │  reconciler reads on start + mtime change               │
│      ▼                                                          │
│  per-monitor goroutines                                         │
│      │                                                          │
│  checker (HTTP/TCP) ──► ~/.uptui/history.json                   │
└─────────────────────────────────────────────────────────────────┘
```

The daemon is the **single writer** of both `monitors.toml` and `history.json`. All mutations (add, delete, edit, pause, resume) go through IPC; the daemon writes the config and then updates its runtime state.

External edits to `monitors.toml` are detected by polling the file's modification time every 5 seconds and trigger a reconcile — goroutines are started, stopped, or restarted to match the new desired state.
