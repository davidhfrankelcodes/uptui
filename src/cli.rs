use clap::{Parser, Subcommand};

/// uptui CLI
#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
pub struct Cli {
    /// Path to the database file
    #[arg(long, global = true)]
    pub db: Option<String>,

    #[command(subcommand)]
    pub command: Option<Commands>,
}

#[derive(Subcommand, Debug)]
pub enum Commands {
    /// Initialize configuration
    Init {
        #[arg(short, long)]
        path: Option<String>,
    },
    /// Run the daemon
    Daemon,
    /// Launch TUI
    Tui,
    /// Run a one-shot check
    Check {
        #[arg(short, long)]
        target: Option<String>,
    },
    /// Manage monitors
    Monitor {
        #[command(subcommand)]
        sub: MonitorCmd,
    },
}

#[derive(Subcommand, Debug)]
pub enum MonitorCmd {
    /// Add or update a monitor
    Add {
        id: String,
        name: String,
        target: String,
    },
    /// List monitors
    List,
    /// Remove a monitor
    Remove {
        id: String,
    },
}
