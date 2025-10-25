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
            let mut cfg = Config::example();
            // Override database path from CLI if provided
            if let Some(db_path) = &cli.db {
                cfg.db.path = db_path.clone();
            }
            uptui::daemon::run_daemon(&cfg).await?;
        }
        Some(uptui::cli::Commands::Tui) => {
            uptui::tui::run_tui()?;
        }
        Some(uptui::cli::Commands::Check { target: _ }) => {
            println!("one-shot check (not implemented)");
        }
        Some(uptui::cli::Commands::Monitor { sub }) => {
            let db_path = cli.db.clone().unwrap_or_else(|| "uptui.db".to_string());
            let db = uptui::storage::Db::open(&db_path).context("open db")?;
            match sub {
                uptui::cli::MonitorCmd::Add { id, name, target, recipients } => {
                    db.insert_monitor(id, name, target)?;
                    if let Some(r) = recipients {
                        // validate, normalize and dedupe addresses
                        let re = regex::Regex::new(r"(?i)^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$").expect("regex");
                        let mut seen = std::collections::HashSet::new();
                        let mut parts = Vec::new();
                        for part in r.split(',') {
                            let p = part.trim().to_lowercase();
                            if p.is_empty() { continue; }
                            if re.is_match(&p) && seen.insert(p.clone()) {
                                parts.push(p);
                            }
                        }
                        if !parts.is_empty() {
                            let joined = parts.join(",");
                            db.set_monitor_recipients(id, Some(joined.as_str()))?;
                        } else {
                            println!("no valid recipients provided; skipping recipients");
                        }
                    }
                    println!("monitor {} added", id);
                }
                uptui::cli::MonitorCmd::List => {
                    let mons = db.list_monitors()?;
                    for m in mons {
                        let recs = m.recipients.clone().unwrap_or_else(|| "-".to_string());
                        println!("{}\t{}\t{}\t{}", m.id, m.name, m.target, recs);
                    }
                }
                uptui::cli::MonitorCmd::Remove { id } => {
                    let n = db.delete_monitor(id)?;
                    if n > 0 {
                        println!("deleted monitor {}", id);
                    } else {
                        println!("monitor {} not found", id);
                    }
                }
                uptui::cli::MonitorCmd::SetRecipients { id, recipients } => {
                    // validate recipients similarly to add
                    let re = regex::Regex::new(r"(?i)^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$").expect("regex");
                    let mut seen = std::collections::HashSet::new();
                    let mut parts = Vec::new();
                    for part in recipients.split(',') {
                        let p = part.trim().to_lowercase();
                        if p.is_empty() { continue; }
                        if re.is_match(&p) && seen.insert(p.clone()) {
                            parts.push(p);
                        }
                    }
                    if parts.is_empty() {
                        println!("no valid recipients provided; not updated");
                    } else {
                        db.set_monitor_recipients(id, Some(parts.join(",").as_str()))?;
                        println!("recipients updated for {}", id);
                    }
                }
                uptui::cli::MonitorCmd::Results { id, limit } => {
                    // Verify monitor exists
                    let monitor = db.get_monitor(id)?;
                    if monitor.is_none() {
                        println!("monitor {} not found", id);
                        return Ok(());
                    }
                    let monitor = monitor.unwrap();
                    
                    // Get recent results (limited to max 100 in storage.rs)
                    let results = db.recent_results(id)?;
                    let display_limit = (*limit).min(100).min(results.len());
                    
                    if results.is_empty() {
                        println!("no results found for monitor {}", id);
                        return Ok(());
                    }
                    
                    println!("Recent results for monitor '{}' ({}):", monitor.name, monitor.target);
                    println!("ID\tSuccess\tStatus\tTimestamp");
                    println!("--\t-------\t------\t---------");
                    
                    for result in results.iter().take(display_limit) {
                        let status_str = match result.status_code {
                            Some(code) => code.to_string(),
                            None => "-".to_string(),
                        };
                        let success_str = if result.success { "✓" } else { "✗" };
                        println!("{}\t{}\t{}\t{}", 
                            result.id, 
                            success_str, 
                            status_str, 
                            result.timestamp.format("%Y-%m-%d %H:%M:%S UTC")
                        );
                    }
                    
                    if results.len() > display_limit {
                        println!("... ({} more results available)", results.len() - display_limit);
                    }
                }
            }
        }
        None => {
            println!("uptui: no command given, run --help for usage");
        }
    }

    Ok(())
}
