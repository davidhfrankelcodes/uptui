# Roadmap

## ✓ v0.1 — core daemon + TUI

Initial release: HTTP and TCP monitors, bubbletea dashboard and detail views, add/delete/pause/resume via IPC.

## ✓ v0.5 — edit monitor

Edit a monitor's name, target, type, interval, or timeout without deleting and re-adding. Accessible via `uptui edit NAME` on the CLI and the `e` key in the TUI (opens a pre-filled form).

## ✓ v0.6 — config file ↔ CLI bidirectionality

`~/.uptui/monitors.toml` is the canonical record of monitors. Fully interchangeable workflows:

- Edit `monitors.toml` by hand → daemon picks it up within 5 seconds
- Use `uptui add` / `uptui edit` / `uptui delete` → file stays in sync
- Use TUI `a` / `e` / `d` → file stays in sync

Key implementation details:
- `monitors.toml` uses TOML `[[monitor]]` array-of-tables format
- Daemon polls file mtime every 5s; on change, runs a reconciler that diffs desired vs running state
- `history.json` replaces the old `db.json`; keyed by monitor name (stable across restarts)
- Monitor identity uses `name` (string) as primary key — `ID int` removed
- New IPC actions: `edit` (update settings + optional rename), `reload` (force re-read)

---

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

## ✓ v0.7 — themes (partial)

7 built-in color themes selectable via `uptui theme NAME` or `~/.uptui/settings.toml`: `default`, `dracula`, `nord`, `solarized`, `monokai`, `gruvbox`, `monochrome`. Theme preference is stored separately from monitor config so the daemon never needs to know about it.

## v0.7 — remaining TUI improvements

- **Sort and filter** — sort by name, status, or uptime; filter to show only down monitors
- **Uptime columns** — add 7-day and 30-day uptime alongside the current 24-hour figure
- **Log view** — scrollable full check history within the detail view
- **Confirmation prompt** — "Delete monitor X? [y/N]" before destructive actions

## v0.8 — maintenance windows

- Schedule a monitor to be paused during a recurring window (e.g. every Sunday 02:00–04:00)
- Show "maintenance" status in the TUI during the window
- Suppress notifications while in maintenance

## v1.0 — stability and polish

- Replace the flat JSON history store with an embedded SQLite database (`modernc.org/sqlite`) for efficient history queries at scale
- `uptui export` — write a static HTML status page
- Shell completions (`uptui completion bash|zsh|fish`)
- Proper structured logging in the daemon (`log/slog`)
- Published binaries via GitHub Releases (cross-compiled for Linux, macOS, Windows)
