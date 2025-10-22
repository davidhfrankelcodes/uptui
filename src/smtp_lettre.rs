
#![cfg(feature = "smtp")]

use anyhow::Context;
use lettre::message::{Mailbox, Message, header::ContentType};
use lettre::transport::stub::StubTransport;
use lettre::Transport;

/// A lettre-backed SMTP sender using the stub transport by default.
pub struct SmtpSenderL {
    from: String,
    transport: StubTransport,
}

impl SmtpSenderL {
    /// Create a new sender. The `server` argument is currently unused; kept for API
    /// compatibility with the stub sender.
    pub fn new(from: impl Into<String>, _server: Option<String>) -> Self {
        let transport = StubTransport::new_ok();
        Self { from: from.into(), transport }
    }

    fn send_email_blocking(&self, to: &str, subject: &str, body: &str) -> anyhow::Result<()> {
        let email = Message::builder()
            .from(self.from.parse::<Mailbox>()?)
            .to(to.parse::<Mailbox>()?)
            .subject(subject)
            .header(ContentType::TEXT_PLAIN)
            .body(String::from(body))?;

        self.transport.send(&email).context("send email")?;
        Ok(())
    }
}

impl crate::alert::Sender for SmtpSenderL {
    fn send(&self, monitor_id: &str, message: &str) -> anyhow::Result<()> {
        let to = format!("{}@localhost", monitor_id);
        let subject = format!("uptui alert: {}", monitor_id);
        self.send_email_blocking(&to, &subject, message)?;
        Ok(())
    }
}
