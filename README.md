## WARNING: This is a work in progress...

# uptui

uptui is a TUI for uptime checks and monitoring.

Getting started

- Build:
## uptui — a small uptime checks & alerting service (TUI + daemon)

This repository contains a Rust-based monitoring/alerting project with:

- a CLI (subcommands wired via `clap`)
- a long-running daemon for periodic checks and alert dispatch
- a terminal UI (TUI) for interactive inspection and management
- storage and data models, plus SMTP integration for alerts

Repository layout (key files)

- `Cargo.toml` — Rust project manifest
- `config/config.yaml.example` — example configuration
- `src/main.rs` — binary entry (wires config, logging, CLI/daemon)
- `src/lib.rs` — library surface and re-exports
- `src/cli.rs` — command-line interface and subcommands
- `src/daemon.rs` — daemon loop and scheduling logic
- `src/tui.rs` — terminal UI implementation
- `src/config.rs` — config parsing and validation
- `src/alert.rs` — alert generation and queueing logic
- `src/monitor.rs` — monitor/check definitions and logic
- `src/smtp.rs` — SMTP transport abstraction
- `src/smtp_lettre.rs` — lettre adapter used in tests/optional feature
- `src/storage.rs` — storage abstractions / concrete implementations
- `src/data.rs` — shared data models
- `tests/` — integration and unit tests (many test files present)

Build, run, and test

Build (development):

```powershell
cd C:\Users\dfran\Git\uptui
cargo build
```

Run the built binary or use `cargo run` with subcommands. Example CLI usage:

```powershell
# show help
cargo run -- --help

# initialize example config (CLI handles writing the example)
cargo run -- init

# run daemon
cargo run -- daemon

# launch TUI
cargo run -- tui
```

Run tests:

```powershell
cd C:\Users\dfran\Git\uptui
cargo test
```

Configuration

Copy and adapt the example config at `config/config.yaml.example`. The parsing and validation live in `src/config.rs`.

Key behavior and where to look in the code

- CLI: `src/cli.rs` (top-level options like `--db` and subcommands such as `init`, `daemon`, `tui`, `monitor`)
- Daemon: `src/daemon.rs` contains the loop and where checks are scheduled and alerts dispatched
- Alerting: `src/alert.rs` and `src/smtp.rs` define the alert queue and sender abstraction
- Storage: `src/storage.rs` implements persistence (sqlite/local backends)
- TUI: `src/tui.rs` provides the interactive interface

Tests

There are many integration tests under `tests/` covering CLI commands, daemon cycles, storage checks, and SMTP behavior. To run a single test binary:

```powershell
cd C:\Users\dfran\Git\uptui
cargo test --test cli_monitors
```

Expanded TODO (actionable, prioritized)

1) Documentation
	- Add `CONTRIBUTING.md` with development workflow, branch/PR conventions, how to run tests locally, and how to add migrations.
	- Expand `config/config.yaml.example` to list every available option and defaults, plus example SMTP and storage sections for local and production.
	- Add `CHANGELOG.md` and a release checklist.

2) CI / developer tooling
	- Add GitHub Actions to run `cargo fmt -- --check`, `cargo clippy -- -D warnings`, and `cargo test` on pushes and PRs.
	- Add Dependabot or similar to keep deps up to date.
	- Add a `Makefile` or task runners for common dev tasks (format, lint, test, run).

3) Testing improvements
	- Expand integration tests to cover SMTP transient failures and retry/backoff behavior (use a fake SMTP server in tests).
	- Add tests for storage migration and recovery scenarios (fixtures for corrupted DBs).
	- Add a small harness for deterministic TUI tests (where practical) or snapshot tests for TUI rendering.

4) Reliability & observability
	- Ensure the daemon supports graceful shutdown (SIGTERM/ctrl-c), flushes state, and persists pending work.
	- Add structured logging (include monitor IDs/trace IDs) and a metrics endpoint (Prometheus) from the daemon.
	- Add alert delivery metrics (success/failures, retries) and exporter integration.

5) Alert delivery and resumability
	- Add configurable retry/backoff and max retries for SMTP delivery (config-driven).
	- Implement alert deduplication to avoid repeated notifications for the same ongoing incident.
	- Add a sandbox transport for development to avoid sending real emails during dev runs.

6) Storage & migrations
	- Add migration tooling and versioning for persisted schemas (consider `sqlx` migrations or a lightweight embedded migration table).
	- Add data retention/rotation jobs and tests to ensure old results are pruned safely.

7) CLI & UX
	- Add `monitor results <id>` and `alert list` / `alert resend` CLI commands.
	- Improve CLI help and examples; consider man page generation.
	- Add TUI help overlay with keyboard shortcuts and a short tutorial.

8) Packaging & ops
	- Add example `systemd` service file and a Dockerfile + docker-compose for local testing.
	- Add packaging guidance (deb/rpm) and a release pipeline.

9) Minor / housekeeping
	- Add pre-commit hooks to run `cargo fmt` and basic linters.
	- Add a small README section explaining how to contribute and which areas need help.

How I verified this change

- I updated `README.md` to reflect the repository layout and the files present under `src/` and `tests/`.
- The README now contains accurate file references and actionable next steps.

Next steps I can take for you

- Add `CONTRIBUTING.md` and a minimal GitHub Actions workflow to run tests and linters.
- Expand `config/config.yaml.example` with all currently-parsed config keys from `src/config.rs`.
- Create a `docs/` folder with short how-tos (deploying with systemd, running locally, etc.).

If you'd like me to commit the README update now, I already updated it in the repository. I can also open a follow-up PR adding CI and contributing docs next — tell me which item from the expanded TODO list to pick and I'll proceed.
- [ ] Daemon loop: periodic scheduler, graceful shutdown, metrics endpoint.

- [ ] Systemd service example and packaging (deb/rpm) examples.

