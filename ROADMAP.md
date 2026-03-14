# Roadmap

## v0.2 — richer HTTP checks

The current HTTP monitor does a plain `GET` and checks the status code. Most real-world use cases need more.

- **Custom expected status code** — configure what counts as "up" (e.g. `301`, `401`)
- **Keyword match** — mark down if the response body doesn't contain a string
- **Custom request method and headers** — useful for authenticated endpoints
- **TLS certificate expiry** — warn N days before a cert expires, independent of HTTP status

## v0.3 — more monitor types

- **Ping (ICMP)** — check host reachability without opening a port
- **DNS** — resolve a hostname and optionally assert the returned IP
- **Heartbeat** — uptui exposes a URL; an external job POSTs to it on a schedule and uptui alerts if it stops

## v0.4 — notifications

Alerts when a monitor transitions from up → down or down → up.

- **Desktop notification** — native OS notification via `notify-send` / `osascript` / Windows toast
- **Webhook** — HTTP POST to a configurable URL with a JSON payload
- **Run command** — execute an arbitrary shell command on state change
- Configurable: alert only on down, only on recovery, or both
- Cooldown period to avoid repeat alerts during a sustained outage

## v0.5 — TUI improvements

- **Edit monitor** — change name, target, interval, or timeout without delete + re-add
- **Sort and filter** — sort by name, status, or uptime; filter to show only down monitors
- **Uptime columns** — add 7-day and 30-day uptime alongside the current 24-hour figure
- **Log view** — scrollable full check history within the detail view
- **Confirmation prompt** — "Delete monitor X? [y/N]" before destructive actions

## v0.6 — configuration file

A `~/.uptui/config.toml` (or YAML) for monitors and global settings, so the full setup can be version-controlled and reproduced on a new machine.

```toml
[settings]
interval        = 60
timeout         = 30
notification    = "webhook"
webhook_url     = "https://hooks.example.com/abc"

[[monitor]]
name     = "Production API"
type     = "http"
target   = "https://api.example.com/health"
interval = 30

[[monitor]]
name   = "Postgres"
type   = "tcp"
target = "db.internal:5432"
```

The daemon watches the file for changes and applies them without a restart.

## v0.7 — maintenance windows

- Schedule a monitor to be paused during a recurring window (e.g. every Sunday 02:00–04:00)
- Show "maintenance" status in the TUI during the window
- Suppress notifications while in maintenance

## v1.0 — stability and polish

- Replace the flat JSON store with an embedded SQLite database (`modernc.org/sqlite`) for efficient history queries at scale
- `uptui export` — write a static HTML status page
- `uptui import` — load monitors from a config file or JSON export
- Shell completions (`uptui completion bash|zsh|fish`)
- Proper structured logging in the daemon (`log/slog`)
- Published binaries via GitHub Releases (cross-compiled for Linux, macOS, Windows)
