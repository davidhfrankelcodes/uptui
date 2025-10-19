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
    db.insert_monitor(monitor_id, monitor_id, url)?;

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

/// Run one scheduler cycle: list monitors, run checks, store results and create alerts on failures.
pub fn run_one_cycle(db_path: &str) -> anyhow::Result<()> {
    let db = Db::open(db_path)?;
    let monitors = db.list_monitors()?;

    for m in monitors {
        let res = reqwest::blocking::get(&m.target);
        let now = Utc::now();
        match res {
            Ok(r) => {
                let status = r.status().as_u16();
                let success = r.status().is_success();
                let _id = db.insert_result(&m.id, success, Some(status), now)?;
                if !success {
                    let msg = format!("monitor {} returned status {}", m.id, status);
                    db.insert_alert(&m.id, &msg, now)?;
                }
            }
            Err(_e) => {
                let _id = db.insert_result(&m.id, false, None, now)?;
                let msg = format!("monitor {} failed to reach target", m.id);
                db.insert_alert(&m.id, &msg, now)?;
            }
        }
    }

    Ok(())
}
