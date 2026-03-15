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
| `internal/models` | `models_test.go` | black-box (`package models_test`) | Tests `ParseAcceptedStatuses` — empty, single, range, mixed, whitespace, invalid, inverted range |
| `internal/checker` | `checker_test.go` | black-box (`package checker_test`) | Starts a real local HTTP/TCP server on loopback; includes `port` type alias and `accepted_statuses` tests |
| `internal/config` | `config_test.go` | black-box (`package config_test`) | Reads/writes temp files; covers `Load`/`Save`, `LoadSettings`/`SaveSettings`, `port`→`tcp` normalization, and `accepted_statuses` round-trip |
| `internal/daemon` | `daemon_test.go` | white-box (`package daemon`) | Tests unexported `calcUptime` for 24 h, 7 d, and 30 d windows; no network |
| `internal/ipc` | `ipc_test.go` | black-box (`package ipc_test`) | Starts a real IPC server on a random port; includes large-payload regression (50 monitors × 500 history entries) |
| `internal/store` | `store_test.go` | black-box (`package store_test`) | Reads/writes temp files |
| `internal/tui` | `app_test.go`, `theme_test.go` | white-box (`package tui`) | No network, no real daemon; messages injected directly into `model.Update()`; includes `accepted_statuses` form validation |

---

## What is and isn't tested

**Covered:**
- Models: `ParseAcceptedStatuses` — format parsing, validation, edge cases
- Config load/save round-trips, defaults, atomic writes, settings (theme), `port`→`tcp` normalization, `accepted_statuses` round-trip
- History store: add, delete, rename, persistence, 500-result cap
- Checker: HTTP up/down/redirects/cancelled context, TCP up/down, `port` type alias, `accepted_statuses` (single code, ranges, multi, default behavior)
- Daemon: uptime calculation over 24 h, 7 d, and 30 d rolling windows
- IPC: all actions (list, add, delete, pause, resume, edit, reload) over a real TCP socket; large-payload responses (>64 KB)
- TUI model: navigation, view switching, form validation (HTTP protocol prefix, TCP `host:port` format, `port` type alias, `accepted_statuses` format + TCP ignore), delete/edit confirmation flow, sparklines, all 7 themes, sort/filter cycling, `visibleMonitors` filtering, detail-view scroll (j/k, clamp, reset), scroll indicators, dashboard viewport scrolling (cursor stays visible in long lists)

**Not covered:**
- `cmd/uptui` (`main.go`) — no test file; CLI behaviour is exercised manually
- End-to-end daemon + TUI integration — test the two processes talking to each other manually with `uptui daemon` in one terminal and `uptui` in another
