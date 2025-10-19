use crate::config::Config;
use crate::storage::Db;
use chrono::Utc;

pub async fn run_daemon(_cfg: &Config) -> anyhow::Result<()> {
    // placeholder: real daemon would schedule checks, enqueue alerts, and manage rotation
    tracing::info!("daemon started (placeholder)");
    Ok(())
}

/// Run a single http check and store result in the database at `db_path`.
pub fn run_check_once(db_path: &str, monitor_id: &str, url: &str) -> anyhow::Result<i64> {
    let db = Db::open(db_path)?;
    db.insert_monitor(monitor_id, monitor_id)?;

    let res = reqwest::blocking::get(url);
    let now = Utc::now();

    match res {
        Ok(r) => {
            let status = r.status().as_u16();
            let success = r.status().is_success();
            let id = db.insert_result(monitor_id, success, Some(status), now)?;
            Ok(id)
        }
        Err(_e) => {
            let id = db.insert_result(monitor_id, false, None, now)?;
            Ok(id)
        }
    }
}
