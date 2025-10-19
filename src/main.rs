use anyhow::Context;
use clap::Parser;

use uptui::{cli::Cli, config::Config};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt::init();

    let cli = Cli::parse();

    match &cli.command {
        Some(uptui::cli::Commands::Init { path }) => {
            let cfg = Config::example();
            let out = path.clone().unwrap_or_else(|| "config.yaml".to_string());
            let yaml = serde_yaml::to_string(&cfg).context("serializing example config")?;
            std::fs::write(&out, yaml).context("writing example config file")?;
            println!("Wrote example config to {}", out);
        }
        Some(uptui::cli::Commands::Daemon) => {
            let cfg = Config::example();
            uptui::daemon::run_daemon(&cfg).await?;
        }
        Some(uptui::cli::Commands::Tui) => {
            uptui::tui::run_tui()?;
        }
        Some(uptui::cli::Commands::Check { target: _ }) => {
            println!("one-shot check (not implemented)");
        }
        None => {
            println!("uptui: no command given, run --help for usage");
        }
    }

    Ok(())
}
