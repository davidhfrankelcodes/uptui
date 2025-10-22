use crate::alert::Sender;
use tracing::info;

/// A stub SMTP sender that logs sends. This is easy to test.
#[cfg(feature = "smtp")]
pub use crate::smtp_lettre::SmtpSenderL as SmtpSender;

#[cfg(not(feature = "smtp"))]
mod stub {
    use anyhow::Result;
    use tracing::info;

    pub struct SmtpSender {
        pub from: String,
        pub server: Option<String>,
    }

    impl SmtpSender {
        pub fn new(from: impl Into<String>, server: Option<String>) -> Self {
            Self { from: from.into(), server }
        }
    }

    impl crate::alert::Sender for SmtpSender {
        fn send(&self, monitor_id: &str, message: &str) -> Result<()> {
            if let Some(srv) = &self.server {
                info!(monitor_id = monitor_id, server = srv, message = message, "sending alert via stub");
            } else {
                info!(monitor_id = monitor_id, message = message, "sending alert via stub");
            }
            Ok(())
        }
    }
}

#[cfg(not(feature = "smtp"))]
pub use stub::SmtpSender;
