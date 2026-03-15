# Running the tests

## Prerequisites

Go 1.21 or later. If `go` is not on your PATH, the default install location on Windows is `C:\Program Files\Go\bin\go`.

Run once after cloning to fetch dependencies and write `go.sum`:

```bash
go mod tidy
```

---

## Run all tests

```bash
go test ./...
```

Typical output — all packages should show `ok`:

```
?       uptui/cmd/uptui         [no test files]
ok      uptui/internal/checker  6.0s
ok      uptui/internal/config   0.9s
ok      uptui/internal/daemon   0.9s
ok      uptui/internal/ipc      0.9s
?       uptui/internal/models   [no test files]
ok      uptui/internal/store    1.6s
ok      uptui/internal/tui      1.0s
```

---

## Useful flags

| Command | What it does |
|---------|--------------|
| `go test ./...` | Run all tests, summary output |
| `go test -v ./...` | Verbose — print each test name as it runs |
| `go test -run TestName ./...` | Run only tests whose name matches the pattern |
| `go test -count=1 ./...` | Disable result caching (always re-runs) |
| `go test -race ./...` | Enable the data-race detector |

---

## Run a single package

```bash
go test ./internal/tui/...
go test ./internal/config/...
go test ./internal/checker/...
go test ./internal/store/...
go test ./internal/daemon/...
go test ./internal/ipc/...
```

---

## Run a single test

```bash
go test -v -run TestDPrimesPendingDelete ./internal/tui/...
go test -v -run TestSaveSettingsRoundTrip ./internal/config/...
```

Pattern matching is a substring/regex match on the test function name, so `-run TestEdit` will match all tests whose names contain `TestEdit`.

---

## Package overview

| Package | Test file | Style | Notes |
|---------|-----------|-------|-------|
| `internal/checker` | `checker_test.go` | black-box (`package checker_test`) | Starts a real local HTTP/TCP server; needs network access on loopback |
| `internal/config` | `config_test.go` | black-box (`package config_test`) | Reads/writes temp files; covers `Load`/`Save` and `LoadSettings`/`SaveSettings` |
| `internal/daemon` | `daemon_test.go` | white-box (`package daemon`) | Tests unexported `calcUptime`; no network |
| `internal/ipc` | `ipc_test.go` | black-box (`package ipc_test`) | Starts a real IPC server on a random port; TOCTOU window is acceptable in tests |
| `internal/store` | `store_test.go` | black-box (`package store_test`) | Reads/writes temp files |
| `internal/tui` | `app_test.go`, `theme_test.go` | white-box (`package tui`) | No network, no real daemon; messages injected directly into `model.Update()` |

---

## What is and isn't tested

**Covered:**
- Config load/save round-trips, defaults, atomic writes, and settings (theme)
- History store: add, delete, rename, persistence, 500-result cap
- Checker: HTTP up/down, redirects, cancelled context, TCP up/down
- Daemon: uptime calculation over a 24-hour rolling window
- IPC: all actions (list, add, delete, pause, resume, edit, reload) over a real TCP socket
- TUI model: navigation, view switching, form validation, delete/edit confirmation flow, sparklines, all 7 themes

**Not covered:**
- `cmd/uptui` (`main.go`) — no test file; CLI behaviour is exercised manually
- End-to-end daemon + TUI integration — test the two processes talking to each other manually with `uptui daemon` in one terminal and `uptui` in another
