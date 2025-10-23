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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::storage::Db;
    use std::sync::{Arc, Mutex};
    use tempfile::NamedTempFile;

    // Mock sender for testing
    #[derive(Debug, Default)]
    struct MockSender {
        sent_messages: Arc<Mutex<Vec<(String, String)>>>,
        send_to_messages: Arc<Mutex<Vec<(String, String)>>>,
        should_fail: Arc<Mutex<bool>>,
    }

    impl MockSender {
        fn new() -> Self {
            Self::default()
        }

        fn set_should_fail(&self, fail: bool) {
            *self.should_fail.lock().unwrap() = fail;
        }

        fn get_sent_messages(&self) -> Vec<(String, String)> {
            self.sent_messages.lock().unwrap().clone()
        }

        fn get_send_to_messages(&self) -> Vec<(String, String)> {
            self.send_to_messages.lock().unwrap().clone()
        }
    }

    impl Sender for MockSender {
        fn send(&self, monitor_id: &str, message: &str) -> anyhow::Result<()> {
            if *self.should_fail.lock().unwrap() {
                return Err(anyhow::anyhow!("Mock send failure"));
            }
            self.sent_messages.lock().unwrap().push((monitor_id.to_string(), message.to_string()));
            Ok(())
        }

        fn send_to(&self, to: &str, message: &str) -> anyhow::Result<()> {
            if *self.should_fail.lock().unwrap() {
                return Err(anyhow::anyhow!("Mock send_to failure"));
            }
            self.send_to_messages.lock().unwrap().push((to.to_string(), message.to_string()));
            Ok(())
        }
    }

    fn create_test_db_with_alerts() -> (NamedTempFile, String, Db) {
        let temp_file = NamedTempFile::new().expect("Failed to create temp file");
        let db_path = temp_file.path().to_string_lossy().to_string();
        let db = Db::open(&db_path).expect("Failed to create test database");
        
        // Insert a monitor
        db.insert_monitor("test-monitor", "Test Monitor", "http://example.com")
            .expect("Failed to insert monitor");
        
        (temp_file, db_path, db)
    }

    #[test]
    fn test_dispatch_no_alerts() {
        let (_temp_file, db_path, _db) = create_test_db_with_alerts();
        let sender = MockSender::new();
        
        let dispatched = dispatch_pending_alerts(&sender, &db_path, None)
            .expect("Failed to dispatch alerts");
        
        assert_eq!(dispatched, 0);
        assert_eq!(sender.get_sent_messages().len(), 0);
    }

    #[test]
    fn test_dispatch_single_alert() {
        let (_temp_file, db_path, db) = create_test_db_with_alerts();
        let sender = MockSender::new();
        
        // Insert an alert
        let _alert_id = db.insert_alert("test-monitor", "Test alert message", Utc::now())
            .expect("Failed to insert alert");
        
        let dispatched = dispatch_pending_alerts(&sender, &db_path, None)
            .expect("Failed to dispatch alerts");
        
        assert_eq!(dispatched, 1);
        let sent_messages = sender.get_sent_messages();
        assert_eq!(sent_messages.len(), 1);
        assert_eq!(sent_messages[0].0, "test-monitor");
        assert_eq!(sent_messages[0].1, "Test alert message");
        
        // Verify alert is marked as sent
        let alerts = db.fetch_alerts(Some("test-monitor"))
            .expect("Failed to fetch alerts");
        assert_eq!(alerts.len(), 1);
        assert_eq!(alerts[0].sent, true);
    }

    #[test]
    fn test_dispatch_with_recipients() {
        let (_temp_file, db_path, db) = create_test_db_with_alerts();
        let sender = MockSender::new();
        
        // Set recipients for the monitor
        db.set_monitor_recipients("test-monitor", Some("admin@example.com,user@example.com"))
            .expect("Failed to set recipients");
        
        // Insert an alert
        db.insert_alert("test-monitor", "Test alert message", Utc::now())
            .expect("Failed to insert alert");
        
        let dispatched = dispatch_pending_alerts(&sender, &db_path, None)
            .expect("Failed to dispatch alerts");
        
        assert_eq!(dispatched, 1);
        
        // Should use send_to for recipients, not send
        let sent_messages = sender.get_sent_messages();
        let send_to_messages = sender.get_send_to_messages();
        
        assert_eq!(sent_messages.len(), 0);
        assert_eq!(send_to_messages.len(), 2);
        
        let recipients: Vec<String> = send_to_messages.iter().map(|(to, _)| to.clone()).collect();
        assert!(recipients.contains(&"admin@example.com".to_string()));
        assert!(recipients.contains(&"user@example.com".to_string()));
    }

    #[test]
    fn test_dispatch_rate_limiting() {
        let (_temp_file, db_path, db) = create_test_db_with_alerts();
        let sender = MockSender::new();
        
        let now = Utc::now();
        let recent_time = now - Duration::seconds(30); // 30 seconds ago
        
        // Insert and mark an alert as sent recently
        let alert1_id = db.insert_alert("test-monitor", "First alert", recent_time)
            .expect("Failed to insert first alert");
        db.mark_alert_sent(alert1_id, recent_time)
            .expect("Failed to mark first alert as sent");
        
        // Insert a new alert
        db.insert_alert("test-monitor", "Second alert", now)
            .expect("Failed to insert second alert");
        
        // Dispatch with 60-second rate limit (should be blocked)
        let dispatched = dispatch_pending_alerts(&sender, &db_path, Some(60))
            .expect("Failed to dispatch alerts");
        
        assert_eq!(dispatched, 0); // Should be rate limited
        assert_eq!(sender.get_sent_messages().len(), 0);
        
        // Dispatch with 10-second rate limit (should go through)
        let dispatched = dispatch_pending_alerts(&sender, &db_path, Some(10))
            .expect("Failed to dispatch alerts");
        
        assert_eq!(dispatched, 1);
        assert_eq!(sender.get_sent_messages().len(), 1);
    }

    #[test]
    fn test_dispatch_skips_already_sent() {
        let (_temp_file, db_path, db) = create_test_db_with_alerts();
        let sender = MockSender::new();
        
        // Insert and mark an alert as already sent
        let alert_id = db.insert_alert("test-monitor", "Already sent alert", Utc::now())
            .expect("Failed to insert alert");
        db.mark_alert_sent(alert_id, Utc::now())
            .expect("Failed to mark alert as sent");
        
        let dispatched = dispatch_pending_alerts(&sender, &db_path, None)
            .expect("Failed to dispatch alerts");
        
        assert_eq!(dispatched, 0);
        assert_eq!(sender.get_sent_messages().len(), 0);
    }

    #[test]
    fn test_dispatch_handles_send_failure() {
        let (_temp_file, db_path, db) = create_test_db_with_alerts();
        let sender = MockSender::new();
        
        // Make sender fail
        sender.set_should_fail(true);
        
        // Insert an alert
        db.insert_alert("test-monitor", "Test alert message", Utc::now())
            .expect("Failed to insert alert");
        
        let dispatched = dispatch_pending_alerts(&sender, &db_path, None)
            .expect("Failed to dispatch alerts");
        
        assert_eq!(dispatched, 0); // Should not count as dispatched due to failure
        
        // Verify alert is still not marked as sent
        let alerts = db.fetch_alerts(Some("test-monitor"))
            .expect("Failed to fetch alerts");
        assert_eq!(alerts.len(), 1);
        assert_eq!(alerts[0].sent, false);
    }

    #[test]
    fn test_dispatch_multiple_alerts_oldest_first() {
        let (_temp_file, db_path, db) = create_test_db_with_alerts();
        let sender = MockSender::new();
        
        let base_time = Utc::now();
        
        // Insert alerts in chronological order (first to last)
        db.insert_alert("test-monitor", "First alert", base_time)
            .expect("Failed to insert first alert");
        db.insert_alert("test-monitor", "Second alert", base_time + Duration::minutes(1))
            .expect("Failed to insert second alert");
        db.insert_alert("test-monitor", "Third alert", base_time + Duration::minutes(2))
            .expect("Failed to insert third alert");
        
        let dispatched = dispatch_pending_alerts(&sender, &db_path, None)
            .expect("Failed to dispatch alerts");
        
        assert_eq!(dispatched, 3);
        let sent_messages = sender.get_sent_messages();
        assert_eq!(sent_messages.len(), 3);
        
        // Should be dispatched in chronological order (oldest first by ID, which corresponds to insertion order)
        // Since fetch_alerts returns ORDER BY id DESC and we reverse it, the first inserted (lowest ID) comes first
        assert_eq!(sent_messages[0].1, "First alert");
        assert_eq!(sent_messages[1].1, "Second alert");
        assert_eq!(sent_messages[2].1, "Third alert");
    }
}
