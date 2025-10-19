use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct SmtpConfig {
    pub server: String,
    pub port: u16,
    pub username: Option<String>,
    pub password: Option<String>,
    pub from: String,
    /// minimum seconds between emails to same target
    pub rate_limit_seconds: Option<u64>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct StorageConfig {
    pub path: String,
    /// retention in days for historical results
    pub retention_days: Option<u32>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Config {
    pub db: StorageConfig,
    pub smtp: Option<SmtpConfig>,
}

impl Config {
    pub fn example() -> Self {
        Self {
            db: StorageConfig {
                path: "./uptui.db".to_string(),
                retention_days: Some(30),
            },
            smtp: Some(SmtpConfig {
                server: "smtp.example.org".to_string(),
                port: 587,
                username: None,
                password: None,
                from: "uptui@example.org".to_string(),
                rate_limit_seconds: Some(3600),
            }),
        }
    }
}
