use crate::config::Config;

pub async fn run_daemon(_cfg: &Config) -> anyhow::Result<()> {
    // placeholder: real daemon would schedule checks, enqueue alerts, and manage rotation
    tracing::info!("daemon started (placeholder)");
    Ok(())
}
