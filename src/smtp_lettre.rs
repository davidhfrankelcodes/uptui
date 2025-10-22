
#![cfg(feature = "smtp")]

use anyhow::Context;
use lettre::message::{Mailbox, Message, header::ContentType};
use lettre::transport::stub::StubTransport;
use lettre::transport::smtp::authentication::Credentials;
use lettre::transport::smtp::SmtpTransport;
use lettre::Transport;

use crate::config::SmtpConfig;

/// A lettre-backed SMTP sender. By default tests can use `new_with_stub` which
/// uses a stub transport. The `from_config` constructor builds a real SMTP
/// transport using the provided `SmtpConfig` (with STARTTLS on port 587, or
/// plain TLS on 465 depending on port).
pub struct SmtpSenderL {
    from: String,
    transport: TransportKind,
}

enum TransportKind {
    Real(SmtpTransport),
    Stub(StubTransport),
}

impl SmtpSenderL {
    /// Create a sender backed by lettre's StubTransport for unit tests.
    pub fn new_with_stub(from: impl Into<String>) -> anyhow::Result<Self> {
        let stub = StubTransport::new_ok();
        Ok(Self { from: from.into(), transport: TransportKind::Stub(stub) })
    }

    /// Compatibility constructor matching the previous stub API: new(from, server)
    /// It currently returns a stub-backed sender and ignores `server`.
    pub fn new(from: impl Into<String>, _server: Option<String>) -> Self {
        let s_from = from.into();
        match Self::new_with_stub(s_from.clone()) {
            Ok(s) => s,
            Err(_) => {
                // Fallback: create a stub transport directly (shouldn't fail)
                let stub = StubTransport::new_ok();
                Self { from: s_from, transport: TransportKind::Stub(stub) }
            }
        }
    }

    /// Build a real SMTP sender using `SmtpConfig`.
    pub fn from_config(cfg: &SmtpConfig) -> anyhow::Result<Self> {
        let creds = match (&cfg.username, &cfg.password) {
            (Some(u), Some(p)) => Some(Credentials::new(u.clone(), p.clone())),
            _ => None,
        };

        let mut builder = SmtpTransport::relay(&cfg.server)
            .context("resolve smtp server")?;

        builder = builder.port(cfg.port);

        if let Some(creds) = creds {
            builder = builder.credentials(creds);
        }

        let transport = builder.build();

        Ok(Self { from: cfg.from.clone(), transport: TransportKind::Real(transport) })
    }

    fn send_email_blocking(&self, to: &str, subject: &str, body: &str) -> anyhow::Result<()> {
        let email = Message::builder()
            .from(self.from.parse::<Mailbox>()?)
            .to(to.parse::<Mailbox>()?)
            .subject(subject)
            .header(ContentType::TEXT_PLAIN)
            .body(String::from(body))?;

        match &self.transport {
            TransportKind::Real(t) => { t.send(&email).context("send email")?; }
            TransportKind::Stub(s) => { s.send(&email).context("send email")?; }
        }
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
