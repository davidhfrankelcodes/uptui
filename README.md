## WARNING: This is a work in progress...

# uptui

uptui is a TUI for uptime checks and monitoring.

Getting started

- Build:

```bash
cargo build
```

- Example config: `config/config.yaml.example`

- CLI:

```bash
# show help
cargo run -- --help

# init example config
cargo run -- init

# run daemon (placeholder)
cargo run -- daemon

# launch TUI (placeholder)
cargo run -- tui
```

CLI: top-level help

For quick reference, here's the current top-level CLI help (what you'll see from the built binary):

```text
david@vbox-zorin  ~/Git/uptui ⎇ main 
$ ./target/debug/uptui --help
uptui CLI

Usage: uptui [OPTIONS] [COMMAND]

Commands:
	init     Initialize configuration
	daemon   Run the daemon
	tui      Launch TUI
	check    Run a one-shot check
	monitor  Manage monitors
	help     Print this message or the help of the given subcommand(s)

Options:
			--db <DB>  Path to the database file
	-h, --help     Print help
	-V, --version  Print version
 david@vbox-zorin  ~/Git/uptui ⎇ main 
$ 
```

What this scaffold contains

- basic CLI wiring using clap
- example Config struct and example YAML
- placeholders for daemon, tui, monitor, and data rotation modules

Next steps

- add persistent storage (sqlite/sqlx)
- implement check runners and scheduler
- implement alert queue and SMTP sending with rate-limiting
- implement TUI with crossterm/tui-rs

Work in progress

- SMTP alerting: added `src/alert.rs` and `src/smtp.rs` with a pluggable `Sender` trait and a stub `SmtpSender`.

Running alerting tests:

```bash
cargo test --test alerting -- --nocapture
```

Recent changes

- CLI monitor management: added a global `--db` option and `monitor` subcommands (`add`, `list`, `remove`).

	Examples:

	```bash
	# add a monitor into a temporary DB path
	cargo run -- --db ./uptui.db monitor add m1 "My monitor" http://example.local

	# list monitors
	cargo run -- --db ./uptui.db monitor list

	# remove monitor
	cargo run -- --db ./uptui.db monitor remove m1
	```

- Tests added to cover the new features (run them individually):

	```bash
	cargo test --test cli_monitors -- --nocapture
	cargo test --test smtp_stub -- --nocapture
	cargo test --test daemon_cycle -- --nocapture
	cargo test --test storage_checks -- --nocapture
	```

Next up

- Implement a real SMTP sender (e.g. `lettre`) behind a feature flag and add integration tests against a fake SMTP server.
- Convert check runners to async worker pool and add scheduling for periodic checks.

Roadmap & Checklist

Below is a living checklist of tasks and milestones to enhance uptui. Pick an item, mark it done in your editor or here, and I can implement it.

Core
- [ ] Storage: migrate to a robust schema and add migrations (consider `sqlx`/`barrel` or simple migration scripts).
- [x] Basic storage (sqlite) implemented (monitors, results, alerts).
- [ ] Data rotation: background retention job and configurable retention per DB.

Checks & Scheduler
- [x] One-shot HTTP check runner (blocking) implemented.
- [ ] Async check runners + worker pool (Tokio) with configurable concurrency.
- [ ] Scheduling: interval-based scheduler for monitors (per-monitor interval).
- [ ] Check types: add TCP connect and ICMP (or TCP fallback) checks.

Alerting
- [x] Alert queue and DB-backed alerts table implemented.
- [x] Pluggable alert `Sender` trait and SMTP stub implemented.
- [ ] Real SMTP sender using `lettre` behind a feature flag.
- [ ] Rate-limiting and cooldown per-monitor (configurable via YAML). Already supported in dispatch logic; wire to config.
- [ ] Alert deduplication: avoid repeated notifications for same ongoing incident.
- [ ] Alert delivery confirmation and retry/backoff for transient SMTP failures.

CLI & TUI
- [x] CLI basics + `monitor add/list/remove` implemented.
- [ ] CLI: `monitor results <id>`, `alert list`, `alert resend` commands.
- [ ] TUI: interactive monitor list, results view, alert management (use `tui` + `crossterm`).

Daemon & Ops
- [x] `run_one_cycle` to run checks and enqueue alerts implemented.
- [ ] Daemon loop: periodic scheduler, graceful shutdown, metrics endpoint.
- [ ] Systemd service example and packaging (deb/rpm) examples.

Testing & CI
- [x] Unit + integration tests for storage, checks, CLI exist.
- [ ] Add CI (GitHub Actions) to run cargo test on push and PRs.
- [ ] Add fuzz/integration tests for HTTP/TCP check types.
- [ ] Add tests for SMTP `lettre` integration using a fake SMTP server or test harness.

Docs & UX
- [ ] Expand README with configuration reference and examples.
- [ ] Add example `config.yaml` for common setups (local, docker, production).
- [ ] Add CHANGELOG and CONTRIBUTING guidelines.

Security & Ops notes
- Use environment variables or an encrypted secrets store for SMTP credentials (do NOT commit secrets).
- Consider rate limits and queuing to avoid SMTP provider rate throttling.

Suggested short-term milestones (2-week sprints)
- Sprint 1: Implement async check runners + scheduler; add `monitor results` CLI command. (High priority)
- Sprint 2: Implement `lettre` SMTP sender behind feature flag + tests; wire alert dispatch into daemon loop. (High priority)
- Sprint 3: Build a minimal TUI view and add CI + packaging. (Medium priority)

How to run tests

```bash
# run all tests
cargo test -- --nocapture

# run a single integration test
cargo test --test cli_monitors -- --nocapture
```

If you want, I can start right away on any checklist item and add corresponding tests and documentation. Reply with the task number or name and I'll mark it in the todo list and begin. 


