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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config_example() {
        let config = Config::example();
        
        // Test database config
        assert_eq!(config.db.path, "./uptui.db");
        assert_eq!(config.db.retention_days, Some(30));
        
        // Test SMTP config
        assert!(config.smtp.is_some());
        let smtp = config.smtp.unwrap();
        assert_eq!(smtp.server, "smtp.example.org");
        assert_eq!(smtp.port, 587);
        assert_eq!(smtp.username, None);
        assert_eq!(smtp.password, None);
        assert_eq!(smtp.from, "uptui@example.org");
        assert_eq!(smtp.rate_limit_seconds, Some(3600));
    }

    #[test]
    fn test_config_serialization() {
        let config = Config::example();
        
        // Test that the config can be serialized to YAML
        let yaml_str = serde_yaml::to_string(&config)
            .expect("Failed to serialize config to YAML");
        
        // Test that it can be deserialized back
        let deserialized: Config = serde_yaml::from_str(&yaml_str)
            .expect("Failed to deserialize config from YAML");
        
        // Verify the deserialized config matches the original
        assert_eq!(deserialized.db.path, config.db.path);
        assert_eq!(deserialized.db.retention_days, config.db.retention_days);
        
        assert!(deserialized.smtp.is_some());
        let original_smtp = config.smtp.as_ref().unwrap();
        let deserialized_smtp = deserialized.smtp.as_ref().unwrap();
        
        assert_eq!(deserialized_smtp.server, original_smtp.server);
        assert_eq!(deserialized_smtp.port, original_smtp.port);
        assert_eq!(deserialized_smtp.from, original_smtp.from);
        assert_eq!(deserialized_smtp.rate_limit_seconds, original_smtp.rate_limit_seconds);
    }

    #[test]
    fn test_smtp_config_optional() {
        let config_without_smtp = Config {
            db: StorageConfig {
                path: "./test.db".to_string(),
                retention_days: None,
            },
            smtp: None,
        };
        
        // Should be able to serialize/deserialize config without SMTP
        let yaml_str = serde_yaml::to_string(&config_without_smtp)
            .expect("Failed to serialize config without SMTP");
        
        let deserialized: Config = serde_yaml::from_str(&yaml_str)
            .expect("Failed to deserialize config without SMTP");
        
        assert_eq!(deserialized.db.path, "./test.db");
        assert_eq!(deserialized.db.retention_days, None);
        assert!(deserialized.smtp.is_none());
    }
}
