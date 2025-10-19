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
