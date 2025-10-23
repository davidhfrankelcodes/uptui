use anyhow::Context;
use chrono::{DateTime, Utc};
use rusqlite::{params, Connection};

#[derive(Debug)]
pub struct Db {
    conn: Connection,
}

#[derive(Debug)]
pub struct CheckResult {
    pub id: i64,
    pub monitor_id: String,
    pub success: bool,
    pub status_code: Option<u16>,
    pub timestamp: DateTime<Utc>,
}

#[derive(Debug)]
pub struct MonitorRecord {
    pub id: String,
    pub name: String,
    pub target: String,
    pub recipients: Option<String>,
}

#[derive(Debug)]
pub struct Alert {
    pub id: i64,
    pub monitor_id: String,
    pub message: String,
    pub created_at: DateTime<Utc>,
    pub sent: bool,
    pub sent_at: Option<DateTime<Utc>>,
}

impl Db {
    pub fn open(path: &str) -> anyhow::Result<Self> {
        let conn = Connection::open(path).context("open sqlite")?;
        let db = Self { conn };
        db.init()?;
        Ok(db)
    }

    fn init(&self) -> anyhow::Result<()> {
        self.conn.execute_batch(
            "BEGIN;
            CREATE TABLE IF NOT EXISTS monitors (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                target TEXT NOT NULL,
                recipients TEXT
            );
            CREATE TABLE IF NOT EXISTS results (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                monitor_id TEXT NOT NULL,
                success INTEGER NOT NULL,
                status_code INTEGER,
                timestamp TEXT NOT NULL
            );
            CREATE TABLE IF NOT EXISTS alerts (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                monitor_id TEXT NOT NULL,
                message TEXT NOT NULL,
                created_at TEXT NOT NULL,
                sent INTEGER NOT NULL DEFAULT 0,
                sent_at TEXT
            );
            COMMIT;",
        )?;
        Ok(())
    }

    pub fn insert_monitor(&self, id: &str, name: &str, target: &str) -> anyhow::Result<()> {
        self.conn
            .execute(
                "INSERT OR REPLACE INTO monitors (id, name, target, recipients) VALUES (?1, ?2, ?3, COALESCE((SELECT recipients FROM monitors WHERE id = ?1), NULL))",
                params![id, name, target],
            )?;
        Ok(())
    }

    pub fn set_monitor_recipients(&self, id: &str, recipients: Option<&str>) -> anyhow::Result<()> {
        self.conn.execute(
            "UPDATE monitors SET recipients = ?1 WHERE id = ?2",
            params![recipients, id],
        )?;
        Ok(())
    }

    pub fn list_monitors(&self) -> anyhow::Result<Vec<MonitorRecord>> {
        let mut stmt = self
            .conn
            .prepare("SELECT id, name, target, recipients FROM monitors ORDER BY id")?;
        let rows = stmt.query_map([], |r| {
            Ok(MonitorRecord {
                id: r.get(0)?,
                name: r.get(1)?,
                target: r.get(2)?,
                recipients: r.get(3)?,
            })
        })?;

        let mut out = Vec::new();
        for r in rows {
            out.push(r?);
        }
        Ok(out)
    }

    pub fn get_monitor(&self, id: &str) -> anyhow::Result<Option<MonitorRecord>> {
        let mut stmt = self.conn.prepare("SELECT id, name, target, recipients FROM monitors WHERE id = ?1")?;
        let mut rows = stmt.query_map([id], |r| {
            Ok(MonitorRecord {
                id: r.get(0)?,
                name: r.get(1)?,
                target: r.get(2)?,
                recipients: r.get(3)?,
            })
        })?;
        if let Some(r) = rows.next() {
            Ok(Some(r?))
        } else {
            Ok(None)
        }
    }

    pub fn delete_monitor(&self, id: &str) -> anyhow::Result<usize> {
        let n = self.conn.execute("DELETE FROM monitors WHERE id = ?1", params![id])?;
        Ok(n)
    }

    pub fn insert_result(
        &self,
        monitor_id: &str,
        success: bool,
        status_code: Option<u16>,
        timestamp: DateTime<Utc>,
    ) -> anyhow::Result<i64> {
        self.conn.execute(
            "INSERT INTO results (monitor_id, success, status_code, timestamp) VALUES (?1, ?2, ?3, ?4)",
            params![monitor_id, success as i32, status_code.map(|s| s as i32), timestamp.to_rfc3339()],
        )?;
        Ok(self.conn.last_insert_rowid())
    }

    pub fn insert_alert(&self, monitor_id: &str, message: &str, created_at: DateTime<Utc>) -> anyhow::Result<i64> {
        self.conn.execute(
            "INSERT INTO alerts (monitor_id, message, created_at, sent) VALUES (?1, ?2, ?3, 0)",
            params![monitor_id, message, created_at.to_rfc3339()],
        )?;
        Ok(self.conn.last_insert_rowid())
    }

    pub fn mark_alert_sent(&self, alert_id: i64, sent_at: DateTime<Utc>) -> anyhow::Result<()> {
        self.conn.execute(
            "UPDATE alerts SET sent = 1, sent_at = ?1 WHERE id = ?2",
            params![sent_at.to_rfc3339(), alert_id],
        )?;
        Ok(())
    }

    pub fn get_last_sent_time(&self, monitor_id: &str) -> anyhow::Result<Option<DateTime<Utc>>> {
        let mut stmt = self.conn.prepare(
            "SELECT sent_at FROM alerts WHERE monitor_id = ?1 AND sent = 1 AND sent_at IS NOT NULL ORDER BY sent_at DESC LIMIT 1",
        )?;
        let mut rows = stmt.query_map([monitor_id], |r| Ok::<_, rusqlite::Error>(r.get::<_, Option<String>>(0)?))?;
        if let Some(r) = rows.next() {
            let s: Option<String> = r?;
            if let Some(ts) = s {
                let dt = DateTime::parse_from_rfc3339(&ts).map(|d| d.with_timezone(&Utc)).unwrap();
                return Ok(Some(dt));
            }
        }
        Ok(None)
    }

    pub fn fetch_alerts(&self, monitor_id: Option<&str>) -> anyhow::Result<Vec<Alert>> {
        let mut out = Vec::new();
        if let Some(mid) = monitor_id {
            let mut stmt = self.conn.prepare("SELECT id, monitor_id, message, created_at, sent, sent_at FROM alerts WHERE monitor_id = ?1 ORDER BY id DESC")?;
            let rows = stmt.query_map([mid], |r| {
                let ts: String = r.get(3)?;
                let dt = DateTime::parse_from_rfc3339(&ts).map(|d| d.with_timezone(&Utc)).unwrap();
                Ok(Alert {
                    id: r.get(0)?,
                    monitor_id: r.get(1)?,
                    message: r.get(2)?,
                    created_at: dt,
                    sent: r.get::<_, i32>(4)? != 0,
                    sent_at: r.get::<_, Option<String>>(5)?.map(|s| DateTime::parse_from_rfc3339(&s).map(|d| d.with_timezone(&Utc)).unwrap()),
                })
            })?;
            for r in rows {
                out.push(r?);
            }
        } else {
            let mut stmt = self.conn.prepare("SELECT id, monitor_id, message, created_at, sent, sent_at FROM alerts ORDER BY id DESC")?;
            let rows = stmt.query_map([], |r| {
                let ts: String = r.get(3)?;
                let dt = DateTime::parse_from_rfc3339(&ts).map(|d| d.with_timezone(&Utc)).unwrap();
                Ok(Alert {
                    id: r.get(0)?,
                    monitor_id: r.get(1)?,
                    message: r.get(2)?,
                    created_at: dt,
                    sent: r.get::<_, i32>(4)? != 0,
                    sent_at: r.get::<_, Option<String>>(5)?.map(|s| DateTime::parse_from_rfc3339(&s).map(|d| d.with_timezone(&Utc)).unwrap()),
                })
            })?;
            for r in rows {
                out.push(r?);
            }
        }
        Ok(out)
    }

    pub fn recent_results(&self, monitor_id: &str) -> anyhow::Result<Vec<CheckResult>> {
        let mut stmt = self.conn.prepare(
            "SELECT id, monitor_id, success, status_code, timestamp FROM results WHERE monitor_id = ?1 ORDER BY id DESC LIMIT 100",
        )?;
        let rows = stmt.query_map([monitor_id], |r| {
            let ts: String = r.get(4)?;
            let dt = DateTime::parse_from_rfc3339(&ts).map(|d| d.with_timezone(&Utc)).unwrap();
            Ok(CheckResult {
                id: r.get(0)?,
                monitor_id: r.get(1)?,
                success: r.get::<_, i32>(2)? != 0,
                status_code: r.get::<_, Option<i32>>(3)?.map(|v| v as u16),
                timestamp: dt,
            })
        })?;

        let mut out = Vec::new();
        for r in rows {
            out.push(r?);
        }
        Ok(out)
    }

    pub fn rotate(&self, retention_days: u32) -> anyhow::Result<usize> {
        let cutoff = Utc::now() - chrono::Duration::days(retention_days as i64);
        let cutoff_s = cutoff.to_rfc3339();
        let n = self.conn.execute(
            "DELETE FROM results WHERE timestamp < ?1",
            params![cutoff_s],
        )?;
        Ok(n)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::{Duration, TimeZone};

    fn create_test_db() -> Db {
        Db::open(":memory:").expect("Failed to create test database")
    }

    #[test]
    fn test_db_initialization() {
        let _db = create_test_db();
        // If we get here without panicking, initialization worked
        assert!(true);
    }

    #[test]
    fn test_insert_and_get_monitor() {
        let db = create_test_db();
        
        // Insert a monitor
        db.insert_monitor("test-1", "Test Monitor", "http://example.com")
            .expect("Failed to insert monitor");
        
        // Retrieve the monitor
        let monitor = db.get_monitor("test-1")
            .expect("Failed to get monitor")
            .expect("Monitor not found");
        
        assert_eq!(monitor.id, "test-1");
        assert_eq!(monitor.name, "Test Monitor");
        assert_eq!(monitor.target, "http://example.com");
        assert_eq!(monitor.recipients, None);
    }

    #[test]
    fn test_get_nonexistent_monitor() {
        let db = create_test_db();
        
        let monitor = db.get_monitor("nonexistent")
            .expect("Failed to query monitor");
        
        assert!(monitor.is_none());
    }

    #[test]
    fn test_list_monitors() {
        let db = create_test_db();
        
        // Insert multiple monitors
        db.insert_monitor("test-1", "Test Monitor 1", "http://example1.com")
            .expect("Failed to insert monitor 1");
        db.insert_monitor("test-2", "Test Monitor 2", "http://example2.com")
            .expect("Failed to insert monitor 2");
        
        let monitors = db.list_monitors().expect("Failed to list monitors");
        
        assert_eq!(monitors.len(), 2);
        assert_eq!(monitors[0].id, "test-1"); // Should be ordered by id
        assert_eq!(monitors[1].id, "test-2");
    }

    #[test]
    fn test_update_monitor_preserves_recipients() {
        let db = create_test_db();
        
        // Insert a monitor
        db.insert_monitor("test-1", "Test Monitor", "http://example.com")
            .expect("Failed to insert monitor");
        
        // Set recipients
        db.set_monitor_recipients("test-1", Some("admin@example.com"))
            .expect("Failed to set recipients");
        
        // Update the monitor (should preserve recipients)
        db.insert_monitor("test-1", "Updated Monitor", "http://updated.com")
            .expect("Failed to update monitor");
        
        let monitor = db.get_monitor("test-1")
            .expect("Failed to get monitor")
            .expect("Monitor not found");
        
        assert_eq!(monitor.name, "Updated Monitor");
        assert_eq!(monitor.target, "http://updated.com");
        assert_eq!(monitor.recipients, Some("admin@example.com".to_string()));
    }

    #[test]
    fn test_set_monitor_recipients() {
        let db = create_test_db();
        
        // Insert a monitor
        db.insert_monitor("test-1", "Test Monitor", "http://example.com")
            .expect("Failed to insert monitor");
        
        // Set recipients
        db.set_monitor_recipients("test-1", Some("admin@example.com,user@example.com"))
            .expect("Failed to set recipients");
        
        let monitor = db.get_monitor("test-1")
            .expect("Failed to get monitor")
            .expect("Monitor not found");
        
        assert_eq!(monitor.recipients, Some("admin@example.com,user@example.com".to_string()));
        
        // Clear recipients
        db.set_monitor_recipients("test-1", None)
            .expect("Failed to clear recipients");
        
        let monitor = db.get_monitor("test-1")
            .expect("Failed to get monitor")
            .expect("Monitor not found");
        
        assert_eq!(monitor.recipients, None);
    }

    #[test]
    fn test_delete_monitor() {
        let db = create_test_db();
        
        // Insert a monitor
        db.insert_monitor("test-1", "Test Monitor", "http://example.com")
            .expect("Failed to insert monitor");
        
        // Verify it exists
        assert!(db.get_monitor("test-1").expect("Failed to get monitor").is_some());
        
        // Delete it
        let deleted_count = db.delete_monitor("test-1")
            .expect("Failed to delete monitor");
        
        assert_eq!(deleted_count, 1);
        
        // Verify it's gone
        assert!(db.get_monitor("test-1").expect("Failed to get monitor").is_none());
        
        // Delete non-existent monitor
        let deleted_count = db.delete_monitor("nonexistent")
            .expect("Failed to delete nonexistent monitor");
        
        assert_eq!(deleted_count, 0);
    }

    #[test]
    fn test_insert_result() {
        let db = create_test_db();
        let timestamp = Utc::now();
        
        let result_id = db.insert_result("test-monitor", true, Some(200), timestamp)
            .expect("Failed to insert result");
        
        assert!(result_id > 0);
        
        // Verify we can retrieve the result
        let results = db.recent_results("test-monitor")
            .expect("Failed to get recent results");
        
        assert_eq!(results.len(), 1);
        assert_eq!(results[0].monitor_id, "test-monitor");
        assert_eq!(results[0].success, true);
        assert_eq!(results[0].status_code, Some(200));
        // Note: We don't check exact timestamp equality due to potential precision differences
    }

    #[test]
    fn test_recent_results_limit() {
        let db = create_test_db();
        let base_time = Utc.with_ymd_and_hms(2023, 1, 1, 0, 0, 0).unwrap();
        
        // Insert more than 100 results
        for i in 0..150 {
            let timestamp = base_time + Duration::minutes(i);
            db.insert_result("test-monitor", i % 2 == 0, Some(200), timestamp)
                .expect("Failed to insert result");
        }
        
        let results = db.recent_results("test-monitor")
            .expect("Failed to get recent results");
        
        // Should be limited to 100 results, ordered by id DESC (most recent first)
        assert_eq!(results.len(), 100);
        
        // First result should be the most recent (id 150)
        assert!(results[0].id > results[1].id);
    }

    #[test]
    fn test_insert_and_fetch_alerts() {
        let db = create_test_db();
        let timestamp = Utc::now();
        
        let alert_id = db.insert_alert("test-monitor", "Test alert message", timestamp)
            .expect("Failed to insert alert");
        
        assert!(alert_id > 0);
        
        // Fetch alerts for specific monitor
        let alerts = db.fetch_alerts(Some("test-monitor"))
            .expect("Failed to fetch alerts");
        
        assert_eq!(alerts.len(), 1);
        assert_eq!(alerts[0].monitor_id, "test-monitor");
        assert_eq!(alerts[0].message, "Test alert message");
        assert_eq!(alerts[0].sent, false);
        assert_eq!(alerts[0].sent_at, None);
        
        // Fetch all alerts
        let all_alerts = db.fetch_alerts(None)
            .expect("Failed to fetch all alerts");
        
        assert_eq!(all_alerts.len(), 1);
    }

    #[test]
    fn test_mark_alert_sent() {
        let db = create_test_db();
        let created_at = Utc::now();
        let sent_at = created_at + Duration::minutes(5);
        
        let alert_id = db.insert_alert("test-monitor", "Test alert", created_at)
            .expect("Failed to insert alert");
        
        // Mark as sent
        db.mark_alert_sent(alert_id, sent_at)
            .expect("Failed to mark alert as sent");
        
        let alerts = db.fetch_alerts(Some("test-monitor"))
            .expect("Failed to fetch alerts");
        
        assert_eq!(alerts.len(), 1);
        assert_eq!(alerts[0].sent, true);
        assert!(alerts[0].sent_at.is_some());
    }

    #[test]
    fn test_get_last_sent_time() {
        let db = create_test_db();
        let base_time = Utc.with_ymd_and_hms(2023, 1, 1, 12, 0, 0).unwrap();
        
        // Initially no sent alerts
        let last_sent = db.get_last_sent_time("test-monitor")
            .expect("Failed to get last sent time");
        assert!(last_sent.is_none());
        
        // Insert and send multiple alerts
        let alert1_id = db.insert_alert("test-monitor", "Alert 1", base_time)
            .expect("Failed to insert alert 1");
        let alert2_id = db.insert_alert("test-monitor", "Alert 2", base_time + Duration::minutes(10))
            .expect("Failed to insert alert 2");
        
        db.mark_alert_sent(alert1_id, base_time + Duration::minutes(5))
            .expect("Failed to mark alert 1 as sent");
        db.mark_alert_sent(alert2_id, base_time + Duration::minutes(15))
            .expect("Failed to mark alert 2 as sent");
        
        // Should return the most recent sent time
        let last_sent = db.get_last_sent_time("test-monitor")
            .expect("Failed to get last sent time")
            .expect("Expected last sent time");
        
        // Should be approximately the second alert's sent time
        let expected = base_time + Duration::minutes(15);
        assert!((last_sent - expected).num_seconds().abs() < 1);
    }

    #[test]
    fn test_rotate_old_results() {
        let db = create_test_db();
        let now = Utc::now();
        let old_time = now - Duration::days(40);
        let recent_time = now - Duration::days(10);
        
        // Insert old and recent results
        db.insert_result("test-monitor", true, Some(200), old_time)
            .expect("Failed to insert old result");
        db.insert_result("test-monitor", true, Some(200), recent_time)
            .expect("Failed to insert recent result");
        
        // Rotate with 30-day retention
        let deleted_count = db.rotate(30)
            .expect("Failed to rotate data");
        
        assert_eq!(deleted_count, 1); // Only the old result should be deleted
        
        let remaining_results = db.recent_results("test-monitor")
            .expect("Failed to get remaining results");
        
        assert_eq!(remaining_results.len(), 1);
        // The remaining result should be the recent one
        assert!((remaining_results[0].timestamp - recent_time).num_seconds().abs() < 1);
    }

    #[test]
    fn test_rotate_no_old_results() {
        let db = create_test_db();
        let now = Utc::now();
        let recent_time = now - Duration::days(10);
        
        // Insert only recent results
        db.insert_result("test-monitor", true, Some(200), recent_time)
            .expect("Failed to insert recent result");
        
        // Rotate with 30-day retention
        let deleted_count = db.rotate(30)
            .expect("Failed to rotate data");
        
        assert_eq!(deleted_count, 0); // No results should be deleted
        
        let remaining_results = db.recent_results("test-monitor")
            .expect("Failed to get remaining results");
        
        assert_eq!(remaining_results.len(), 1);
    }
}
