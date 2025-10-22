use crate::config::Config;
use crate::storage::Db;
use chrono::Utc;
use crate::alert;
use std::sync::Arc;
use tokio::signal;
use tokio::time::{interval, Duration};

pub async fn run_daemon(_cfg: &Config) -> anyhow::Result<()> {
    tracing::info!("daemon starting");

    // build a sender from config if available
    #[cfg(feature = "smtp")]
    let sender: Arc<dyn alert::Sender> = {
        if let Some(s) = &_cfg.smtp {
            match crate::smtp_lettre::SmtpSenderL::from_config(s) {
                Ok(sndr) => Arc::new(sndr),
                Err(e) => {
                    tracing::error!(error = %e, "failed to build lettre smtp sender; falling back to stub");
                    // local fallback stub sender to ensure we have a Sender implementation
                    struct LocalStub;
                    impl crate::alert::Sender for LocalStub {
                        fn send(&self, monitor_id: &str, message: &str) -> anyhow::Result<()> {
                            tracing::info!(monitor = monitor_id, message = message, "local stub send");
                            Ok(())
                        }
                    }
                    Arc::new(LocalStub)
                }
            }
        } else {
            match crate::smtp_lettre::SmtpSenderL::new_with_stub("uptui") {
                Ok(s) => Arc::new(s),
                Err(_) => {
                    struct LocalStub;
                    impl crate::alert::Sender for LocalStub {
                        fn send(&self, monitor_id: &str, message: &str) -> anyhow::Result<()> {
                            tracing::info!(monitor = monitor_id, message = message, "local stub send");
                            Ok(())
                        }
                    }
                    Arc::new(LocalStub)
                }
            }
        }
    };

    #[cfg(not(feature = "smtp"))]
    let sender: Arc<dyn alert::Sender> = if let Some(s) = &_cfg.smtp {
        Arc::new(crate::smtp::SmtpSender::new(s.from.clone(), Some(s.server.clone())))
    } else {
        Arc::new(crate::smtp::SmtpSender::new("uptui".to_string(), None))
    };

    let db_path = &_cfg.db.path;

    let mut tick = interval(Duration::from_secs(10));

    loop {
        tokio::select! {
            _ = tick.tick() => {
                tracing::info!("daemon: running cycle and dispatch");
                let _ = run_cycle_and_dispatch(db_path, sender.as_ref(), _cfg.smtp.as_ref().and_then(|s| s.rate_limit_seconds));
            }
            _ = signal::ctrl_c() => {
                tracing::info!("daemon: received shutdown");
                break;
            }
        }
    }

    Ok(())
}

/// Run a single http check and store result in the database at `db_path`.
pub fn run_check_once(db_path: &str, monitor_id: &str, url: &str) -> anyhow::Result<i64> {
    let db = Db::open(db_path)?;
    db.insert_monitor(monitor_id, monitor_id, url)?;

    let res = reqwest::blocking::get(url);
    let now = Utc::now();

    match res {
        Ok(r) => {
            let status = r.status().as_u16();
            let success = r.status().is_success();
            let id = db.insert_result(monitor_id, success, Some(status), now)?;
            Ok(id)
        }
        Err(_e) => {
            let id = db.insert_result(monitor_id, false, None, now)?;
            Ok(id)
        }
    }
}

/// Run one scheduler cycle: list monitors, run checks, store results and create alerts on failures.
pub fn run_one_cycle(db_path: &str) -> anyhow::Result<()> {
    let db = Db::open(db_path)?;
    let monitors = db.list_monitors()?;

    for m in monitors {
        let res = reqwest::blocking::get(&m.target);
        let now = Utc::now();
        match res {
            Ok(r) => {
                let status = r.status().as_u16();
                let success = r.status().is_success();
                let _id = db.insert_result(&m.id, success, Some(status), now)?;
                if !success {
                    let msg = format!("monitor {} returned status {}", m.id, status);
                    db.insert_alert(&m.id, &msg, now)?;
                }
            }
            Err(_e) => {
                let _id = db.insert_result(&m.id, false, None, now)?;
                let msg = format!("monitor {} failed to reach target", m.id);
                db.insert_alert(&m.id, &msg, now)?;
            }
        }
    }

    Ok(())
}

/// Run one cycle (checks) and then dispatch pending alerts using the provided sender and optional rate limit.
pub fn run_cycle_and_dispatch(db_path: &str, sender: &dyn alert::Sender, rate_limit_seconds: Option<u64>) -> anyhow::Result<usize> {
    // run checks
    run_one_cycle(db_path)?;

    // dispatch alerts
    let dispatched = alert::dispatch_pending_alerts(sender, db_path, rate_limit_seconds)?;
    Ok(dispatched)
}
