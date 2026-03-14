# uptui — notes for Claude

## What this is

`uptui` is a Go application with two roles:

1. **Daemon** (`uptui daemon`) — a long-running background process that checks HTTP endpoints and TCP ports on a configurable interval, persists results to `~/.uptui/history.json`, and writes monitor definitions to `~/.uptui/monitors.toml`.
2. **TUI** (`uptui`) — a [bubbletea](https://github.com/charmbracelet/bubbletea) terminal UI that connects to the daemon over a local TCP socket and displays live monitor status.

The two processes communicate via line-delimited JSON over `127.0.0.1:29374`.

## Package layout

```
cmd/uptui/main.go          CLI entry point (subcommands: daemon, stop, status, add, edit, theme)
internal/
  models/models.go         Shared types: Monitor, Result, MonitorStatus, Status
  config/config.go         TOML Load/Save for monitors.toml; Settings/LoadSettings/SaveSettings
  store/store.go           History-only JSON storage — the only place that writes history.json
  checker/checker.go       Stateless check functions: Check(ctx, Monitor) → Result
  daemon/daemon.go         Daemon: reconciler, watchConfig goroutine, implements ipc.Handler
  ipc/
    protocol.go            Request / Response types and Action constants
    server.go              TCP IPC server (daemon side); defines Handler interface
    client.go              TCP IPC client (TUI side)
  tui/
    theme.go               Theme struct, 7 built-in palettes, ParseTheme, ThemeNames, DefaultTheme
    styles.go              Styles struct + NewStyles(Theme) + StatusStyle method
    app.go                 Bubbletea Model — Init / Update / View + all helper functions
```

## How the pieces connect

```
monitors.toml (hand-editable)
     │  loaded on startup + on mtime change (5 s poll)
     ▼
main.go ──► runDaemon()
              store.New(~/.uptui)
              daemon.New(store, configPath).Run(ctx, ":29374")
                ├── reconcileLocked(desired) → starts/stops/restarts goroutines
                ├── goroutine: watchConfig → mtime poll → reconcileLocked
                ├── goroutine: runMonitor → checker.Check → store.AddResult
                └── ipc.Server.Listen → handles List/Add/Delete/Pause/Resume/Edit/Reload
                      └── every mutation writes back to monitors.toml

main.go ──► runTUI()
              ensureDaemon()           // starts daemon subprocess if not running
              config.LoadSettings(~/.uptui/settings.toml) → theme
              tui.NewModel(ipcClient, theme)
              bubbletea.NewProgram(model).Run()
                ├── Init: fetchData (List RPC) + schedTick
                ├── tickMsg every 5 s: fetchData again
                └── key events → updateDashboard / updateDetail / updateAdd
                      └── 'e' key: opens pre-filled add form in edit mode
```

## Build and test

```bash
go mod tidy                     # required first time (fetches BurntSushi/toml)
go build ./cmd/uptui            # build binary
go install ./cmd/uptui          # install to $GOPATH/bin
go test ./...                   # run all tests
go test -v ./...                # verbose
go test ./internal/store/...    # single package
```

## Key design decisions

**Config as source of truth**: `~/.uptui/monitors.toml` is the canonical record of monitors. The daemon is the **only writer** of that file. All mutations — add, delete, edit, pause, resume — go through IPC; the daemon writes the config file and then updates its runtime state. External edits are detected by polling the file's `mtime` every 5 seconds.

**Settings file**: `~/.uptui/settings.toml` holds user preferences (currently: `theme`). It is owned by the TUI/CLI, not the daemon. `config.LoadSettings` / `config.SaveSettings` handle it; `SaveSettings` omits the file entirely when theme is `"default"`. The daemon never reads or writes this file.

**Single writer for history**: Only the daemon writes `~/.uptui/history.json` (via `store.AddResult`). The TUI never touches either file; it communicates exclusively via IPC.

**Atomic writes**: Both `config.Save()` and `store.save()` write to a `.tmp` file then call `os.Rename`. Rename is atomic on all supported platforms, so a mid-write crash cannot leave a corrupt file.

**Monitor identity — Name, not ID**: `Monitor.ID int` does not exist. `Name string` is the stable unique key everywhere: daemon state map, history store, IPC request fields, CLI arguments. All IPC operations (delete, pause, resume, edit) address monitors by name.

**Reconciler**: On startup and on every detected config-file change, the daemon runs `reconcileLocked(desired)`:
```
for each monitor in desired (config file):
  not in state        → add + start goroutine
  in state, changed   → cancel old goroutine, replace, restart
  active changed      → pause or resume goroutine

for each name in state not in desired:
  cancel goroutine + remove from state
```
This is idempotent — safe to run repeatedly. Called from `Run()` (no lock, single-threaded startup) and from `watchConfig`/`Reload` (with `d.mu` write-locked).

**History cap**: Each monitor keeps at most 500 results in memory and on disk (`store.maxHistory = 500`). The uptime percentage only considers results within a rolling 24-hour window (`calcUptime`). History is keyed by monitor name (`map[string][]models.Result`).

**Context-per-monitor**: Each running monitor goroutine receives its own `context.CancelFunc` stored in `daemon.state[name].cancel`. `PauseMonitor` calls that cancel to stop the goroutine; `ResumeMonitor` starts a new goroutine with a fresh context.

**Auto-start**: When `runTUI()` is called, it checks `client.Ping()` (a plain TCP dial). If the daemon is not reachable it calls `exec.Command(os.Executable(), "daemon")` with stdout/stderr redirected to `~/.uptui/daemon.log` and calls `cmd.Start()` (no `Wait`). It then polls `Ping()` for up to 3 seconds.

**IPC protocol**: Each connection is request/response: the client sends one JSON line, the server responds with one JSON line, then the connection is closed. The server uses `bufio.Scanner` to read lines and `json.NewEncoder` to write responses.

## IPC actions

| Action   | Key request fields   | Effect |
|----------|----------------------|--------|
| `list`   | —                    | Return all MonitorStatus |
| `add`    | Monitor              | Add monitor; write config; start goroutine |
| `delete` | Name                 | Stop goroutine; delete from config |
| `pause`  | Name                 | Stop goroutine; set active=false in config |
| `resume` | Name                 | Start goroutine; set active=true in config |
| `edit`   | OldName + Monitor    | Update settings (handles rename); write config; reconcile |
| `reload` | —                    | Force re-read of monitors.toml and reconcile |

## Config file format

```toml
[[monitor]]
name     = "GitHub"
type     = "http"
target   = "https://github.com"
interval = 30

[[monitor]]
name   = "Postgres"
type   = "tcp"
target = "localhost:5432"
active = false    # written only when paused; omitted (defaults true) otherwise
```

`interval` and `timeout` are omitted when at their defaults (60 s / 30 s). A custom encoder in `config/config.go` (not reflection-based) produces this output; `github.com/BurntSushi/toml` is used only for decoding.

## Conventions

- All IPC actions are defined as `Action` constants in `ipc/protocol.go`.
- `models.Status` values are lowercase strings (`"up"`, `"down"`, `"pending"`, `"paused"`).
- `Monitor.Target` is the raw target string — a full URL for HTTP, `host:port` for TCP.
- `Monitor.Name` is the unique key — never use a numeric ID.
- All lipgloss styles are accessed via `m.styles` (`Styles` struct in `tui/styles.go`). Don't create ad-hoc styles in `app.go`; add new fields to `Styles` and `NewStyles` instead.
- Theme colors live in `tui/theme.go` (`Theme` struct). All 7 built-in palettes are defined there. `ParseTheme("")` returns `DefaultTheme()`.
- The bubbletea model uses value semantics throughout (no pointer receivers). `Update` returns a new model copy.

## Test notes

- `daemon/daemon_test.go` uses `package daemon` (white-box) to test the unexported `calcUptime`.
- `tui/app_test.go` uses `package tui` (white-box) to test unexported helpers and view constants. All `NewModel` calls pass `DefaultTheme()` as the second argument.
- `tui/theme_test.go` uses `package tui` (white-box) to test `ParseTheme`, `ThemeNames`, `NewStyles`, and `StatusStyle`.
- `config/config_test.go` uses `package config_test` (black-box) to test `Load`/`Save` round-trips, defaults, edge cases, and `LoadSettings`/`SaveSettings`.
- IPC tests start a real server on a randomly-assigned port. There is a TOCTOU window between grabbing the port and the server binding to it; this is acceptable in tests.
- TUI tests never connect to a real daemon. `dataMsg` values are injected directly into `model.Update()`.

## Adding a new monitor type

1. Add a constant to `models/models.go` (e.g. `Ping MonitorType = "ping"`).
2. Add a `checkPing` function in `checker/checker.go` and dispatch to it from `Check`.
3. Update the add-form validation in `tui/app.go` (`submitAdd`) to accept the new type string.
4. Add tests in `checker/checker_test.go`.

## Adding a new IPC action

1. Add an `Action` constant in `ipc/protocol.go`.
2. Add a method to the `Handler` interface in `ipc/server.go`.
3. Implement the method on `*daemon.Daemon` in `daemon/daemon.go`.
4. Add a dispatch case in `ipc/server.go` (`dispatch` function).
5. Add a client method in `ipc/client.go`.
6. Add a test case in `ipc/ipc_test.go`.
