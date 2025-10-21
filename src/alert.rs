use crate::storage::Db;
use chrono::{Duration, Utc};

pub trait Sender: Send + Sync {
    /// send an alert; return Ok(()) on success
    fn send(&self, monitor_id: &str, message: &str) -> anyhow::Result<()>;
}

/// Dispatch pending alerts, honoring rate_limit_seconds per monitor if provided.
pub fn dispatch_pending_alerts(sender: &dyn Sender, db_path: &str, rate_limit_seconds: Option<u64>) -> anyhow::Result<usize> {
    let db = Db::open(db_path)?;
    let alerts = db.fetch_alerts(None)?;
    let mut dispatched = 0usize;
    for a in alerts.into_iter().rev() { // oldest first
        if a.sent {
            continue;
        }

        // check rate limit
        if let Some(limit) = rate_limit_seconds {
            if let Some(last) = db.get_last_sent_time(&a.monitor_id)? {
                let since = Utc::now() - last;
                if since < Duration::seconds(limit as i64) {
                    // skip due to rate limit
                    continue;
                }
            }
        }

        // attempt send
        if let Err(_e) = sender.send(&a.monitor_id, &a.message) {
            // keep alert unsent on failure
            continue;
        }
        db.mark_alert_sent(a.id, Utc::now())?;
        dispatched += 1;
    }
    Ok(dispatched)
}
