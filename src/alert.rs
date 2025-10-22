use crate::storage::Db;
use chrono::{Duration, Utc};

pub trait Sender: Send + Sync {
    /// send an alert; return Ok(()) on success
    fn send(&self, monitor_id: &str, message: &str) -> anyhow::Result<()>;

    /// send an alert to a specific recipient (email address). Default falls back to `send`.
    fn send_to(&self, to: &str, message: &str) -> anyhow::Result<()> {
        // default behaviour delegates to `send` using the recipient as monitor id
        self.send(to, message)
    }
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

        // attempt send: prefer per-monitor recipients when available
        let mut sent_ok = false;
        if let Ok(Some(m)) = db.get_monitor(&a.monitor_id) {
            if let Some(recs) = m.recipients {
                // comma-separated list of recipients
                let parts = recs.split(',').map(|s| s.trim()).filter(|s| !s.is_empty()).collect::<Vec<_>>();
                for r in parts {
                    if sender.send_to(r, &a.message).is_ok() {
                        sent_ok = true;
                    }
                }
            }
        }

        if !sent_ok {
            if let Err(_e) = sender.send(&a.monitor_id, &a.message) {
                // keep alert unsent on failure
                continue;
            }
        }
        db.mark_alert_sent(a.id, Utc::now())?;
        dispatched += 1;
    }
    Ok(dispatched)
}
