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
                target TEXT NOT NULL
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
                "INSERT OR REPLACE INTO monitors (id, name, target) VALUES (?1, ?2, ?3)",
                params![id, name, target],
            )?;
        Ok(())
    }

    pub fn list_monitors(&self) -> anyhow::Result<Vec<MonitorRecord>> {
        let mut stmt = self
            .conn
            .prepare("SELECT id, name, target FROM monitors ORDER BY id")?;
        let rows = stmt.query_map([], |r| {
            Ok(MonitorRecord {
                id: r.get(0)?,
                name: r.get(1)?,
                target: r.get(2)?,
            })
        })?;

        let mut out = Vec::new();
        for r in rows {
            out.push(r?);
        }
        Ok(out)
    }

    pub fn get_monitor(&self, id: &str) -> anyhow::Result<Option<MonitorRecord>> {
        let mut stmt = self.conn.prepare("SELECT id, name, target FROM monitors WHERE id = ?1")?;
        let mut rows = stmt.query_map([id], |r| {
            Ok(MonitorRecord {
                id: r.get(0)?,
                name: r.get(1)?,
                target: r.get(2)?,
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
