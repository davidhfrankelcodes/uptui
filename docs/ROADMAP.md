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

## ✓ v0.7 — themes

7 built-in color themes selectable via `uptui theme NAME` or `~/.uptui/settings.toml`: `default`, `dracula`, `nord`, `solarized`, `monokai`, `gruvbox`, `monochrome`. Theme preference is stored separately from monitor config so the daemon never needs to know about it.

## ✓ v0.7 — TUI improvements

- **Sort and filter** — `s` cycles sort (name/status/uptime); `f` cycles filter (all/down/problems). Active sort and filter shown in footer.
- **Uptime columns** — 7-day and 30-day uptime displayed in dashboard (terminals ≥ 100 cols) and detail view, alongside the 24-hour figure.
- **Scrollable check history** — detail view now pages through the full check log; `j`/`↓` older, `k`/`↑` newer, with count indicators.
- **Confirmation prompts** — `y/N` prompt before delete and before saving edits.
- **Dashboard viewport** — list scrolls to keep the cursor visible when monitors outnumber terminal rows.
- **Target format validation** — add/edit form validates HTTP targets require `http://` or `https://`, TCP targets require `host:port` with a valid port number (1–65535).

## ✓ Docker support

Run the daemon in a container with separate volumes for config and data:

- Multi-stage Dockerfile (`golang:1.21-alpine` builder, `alpine:3.19` runtime)
- `docker-compose.yaml` fully parameterized via `.env` (host address, port, volume paths, restart policy)
- `UPTUI_DATA_DIR`, `UPTUI_CONFIG_DIR`, `UPTUI_LISTEN_ADDR` env var overrides
- `ncurses-terminfo-base` added to the Alpine image so colors work in `docker exec` sessions

## ✓ IPC buffer limit removed

`bufio.Scanner` (64 KB limit) replaced with `json.NewDecoder` on both the client and server side. Large `list` responses (many monitors × 500 history entries) no longer cause `token too long` errors.

## ✓ `port` type alias

Monitors configured with `type = "port"` in a hand-edited `monitors.toml` are now treated as `tcp`. The alias is normalized to `tcp` on config load, in the checker, and in the TUI add/edit form.

---

## ✓ Custom HTTP accepted statuses

`accepted_statuses` field on HTTP monitors: comma-separated codes and ranges that count as "up" (e.g. `"200-299,401"`). Empty = any `< 400` is up. Configurable in `monitors.toml`, via `uptui add/edit --accepted-statuses`, and in the TUI add/edit form. Validation in `models.ParseAcceptedStatuses` is shared by the checker and the form.

---

## v0.2 — richer HTTP checks

The current HTTP monitor does a plain `GET` and checks the status code. Most real-world use cases need more.

- **Custom expected status code** — ✓ done (see above)
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
