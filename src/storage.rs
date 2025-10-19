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
                name TEXT NOT NULL
            );
            CREATE TABLE IF NOT EXISTS results (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                monitor_id TEXT NOT NULL,
                success INTEGER NOT NULL,
                status_code INTEGER,
                timestamp TEXT NOT NULL
            );
            COMMIT;",
        )?;
        Ok(())
    }

    pub fn insert_monitor(&self, id: &str, name: &str) -> anyhow::Result<()> {
        self.conn
            .execute(
                "INSERT OR REPLACE INTO monitors (id, name) VALUES (?1, ?2)",
                params![id, name],
            )?;
        Ok(())
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
