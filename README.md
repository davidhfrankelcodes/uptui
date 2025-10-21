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


