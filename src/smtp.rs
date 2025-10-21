use crate::alert::Sender;
use tracing::info;

/// A stub SMTP sender that logs sends. This is easy to test.
pub struct SmtpSender {
    pub from: String,
    pub server: Option<String>,
}

impl SmtpSender {
    pub fn new(from: impl Into<String>, server: Option<String>) -> Self {
        Self { from: from.into(), server }
    }
}

impl Sender for SmtpSender {
    fn send(&self, monitor_id: &str, message: &str) -> anyhow::Result<()> {
        // In a production implementation, use `lettre` to send via SMTP here.
        // For now, log the send so tests can provide a TestSender for assertions.
        if let Some(srv) = &self.server {
            info!("sending alert for {} via {}: {}", monitor_id, srv, message);
        } else {
            info!("sending alert for {}: {}", monitor_id, message);
        }
        Ok(())
    }
}
