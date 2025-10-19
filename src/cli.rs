use clap::{Parser, Subcommand};

/// uptui CLI
#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
pub struct Cli {
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
}
