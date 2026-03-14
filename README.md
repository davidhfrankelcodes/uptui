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
  add  delete  pause/resume  ↑↓ navigate  ↵ detail  r refresh  quit
```

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

Flags for add:
  --name, -n NAME                 Display name  (default: TARGET)
  --type, -t http|tcp             Monitor type  (default: http)
  --interval, -i SECONDS          Check interval (default: 60, min: 10)
```

## TUI keybindings

### Dashboard

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `enter` | Open detail view |
| `a` | Add a new monitor |
| `d` | Delete selected monitor |
| `p` | Pause / resume selected monitor |
| `r` | Force refresh |
| `q` / `ctrl+c` | Quit |

### Detail view

| Key | Action |
|-----|--------|
| `esc` / `backspace` | Back to dashboard |
| `q` / `ctrl+c` | Quit |

### Add form

| Key | Action |
|-----|--------|
| `tab` / `↓` | Next field |
| `shift+tab` / `↑` | Previous field |
| `enter` | Next field / submit (on last field) |
| `esc` | Cancel |

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
| `db.json` | Monitor config + check history (last 500 results per monitor) |
| `daemon.pid` | PID of the running daemon (deleted on clean stop) |
| `daemon.log` | Daemon stdout/stderr when auto-started |

Data is written atomically (write to `.tmp`, then rename) so a crash mid-write cannot corrupt the database.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  uptui (TUI process)                                        │
│  bubbletea model ──── polls every 5s ────┐                  │
└──────────────────────────────────────────┼──────────────────┘
                                           │ JSON over TCP
                                           │ 127.0.0.1:29374
┌──────────────────────────────────────────┼──────────────────┐
│  uptui daemon                            │                  │
│  IPC server ◄────────────────────────────┘                  │
│      │                                                      │
│  per-monitor goroutines                                     │
│      │                                                      │
│  checker (HTTP/TCP) ──► ~/.uptui/db.json                    │
└─────────────────────────────────────────────────────────────┘
```

The daemon is the single writer of `db.json`. The TUI never touches the file directly — it only communicates via IPC.
