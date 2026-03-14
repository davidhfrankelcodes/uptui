# uptui — notes for Claude

## What this is

`uptui` is a Go application with two roles:

1. **Daemon** (`uptui daemon`) — a long-running background process that checks HTTP endpoints and TCP ports on a configurable interval and persists results to a local JSON file.
2. **TUI** (`uptui`) — a [bubbletea](https://github.com/charmbracelet/bubbletea) terminal UI that connects to the daemon over a local TCP socket and displays live monitor status.

The two processes communicate via line-delimited JSON over `127.0.0.1:29374`.

## Package layout

```
cmd/uptui/main.go          CLI entry point (subcommands: daemon, stop, status, add)
internal/
  models/models.go         Shared types: Monitor, Result, MonitorStatus, Status
  store/store.go           JSON file storage — the only place that writes ~/.uptui/db.json
  checker/checker.go       Stateless check functions: Check(ctx, Monitor) → Result
  daemon/daemon.go         Daemon: goroutine per monitor, implements ipc.Handler
  ipc/
    protocol.go            Request / Response types and Action constants
    server.go              TCP IPC server (daemon side); defines Handler interface
    client.go              TCP IPC client (TUI side)
  tui/
    styles.go              Lipgloss colour constants and style variables
    app.go                 Bubbletea Model — Init / Update / View + all helper functions
```

## How the pieces connect

```
main.go ──► runDaemon()
              store.New(~/.uptui)
              daemon.New(store).Run(ctx, ":29374")
                ├── goroutine: runMonitor → checker.Check → store.AddResult
                └── ipc.Server.Listen → handles List / Add / Delete / Pause / Resume

main.go ──► runTUI()
              ensureDaemon()           // starts daemon subprocess if not running
              tui.NewModel(ipcClient)
              bubbletea.NewProgram(model).Run()
                ├── Init: fetchData (List RPC) + schedTick
                ├── tickMsg every 5 s: fetchData again
                └── key events → updateDashboard / updateDetail / updateAdd
```

## Build and test

```bash
go build ./cmd/uptui        # build binary
go install ./cmd/uptui      # install to $GOPATH/bin
go test ./...               # run all tests
go test -v ./...            # verbose
go test ./internal/store/...    # single package
```

## Key design decisions

**Single writer**: Only the daemon reads/writes `~/.uptui/db.json`. The TUI never touches the file; it goes through IPC. This avoids concurrent-write corruption.

**Atomic writes**: `store.save()` writes to `db.json.tmp` then calls `os.Rename`. Rename is atomic on all supported platforms, so a mid-write crash cannot leave a corrupt file.

**History cap**: Each monitor keeps at most 500 results in memory and on disk (`store.maxHistory = 500`). The uptime percentage only considers results within a rolling 24-hour window (`calcUptime`).

**Context-per-monitor**: Each running monitor goroutine receives its own `context.CancelFunc` stored in `daemon.monitorState.cancel`. `PauseMonitor` calls that cancel to stop the goroutine; `ResumeMonitor` starts a new goroutine with a fresh context.

**Auto-start**: When `runTUI()` is called, it checks `client.Ping()` (a plain TCP dial). If the daemon is not reachable it calls `exec.Command(os.Executable(), "daemon")` with stdout/stderr redirected to `~/.uptui/daemon.log` and calls `cmd.Start()` (no `Wait`). It then polls `Ping()` for up to 3 seconds.

**IPC protocol**: Each connection is request/response: the client sends one JSON line, the server responds with one JSON line, then the connection is closed. The server uses `bufio.Scanner` to read lines and `json.NewEncoder` to write responses.

## Conventions

- All IPC actions are defined as `Action` constants in `ipc/protocol.go`.
- `models.Status` values are lowercase strings (`"up"`, `"down"`, `"pending"`, `"paused"`).
- `Monitor.Target` is the raw target string — a full URL for HTTP, `host:port` for TCP.
- Lipgloss styles all live in `tui/styles.go`. Don't create ad-hoc styles in `app.go`.
- The bubbletea model uses value semantics throughout (no pointer receivers). `Update` returns a new model copy.

## Test notes

- `daemon/daemon_test.go` uses `package daemon` (white-box) to test the unexported `calcUptime`.
- `tui/app_test.go` uses `package tui` (white-box) to test unexported helpers and view constants.
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
