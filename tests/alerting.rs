use std::sync::{Arc, Mutex};

struct TestSender {
    sent: Arc<Mutex<Vec<(String, String)>>>,
}

impl TestSender {
    fn new() -> Self {
        Self { sent: Arc::new(Mutex::new(Vec::new())) }
    }
}

impl uptui::alert::Sender for TestSender {
    fn send(&self, monitor_id: &str, message: &str) -> anyhow::Result<()> {
        let mut s = self.sent.lock().unwrap();
        s.push((monitor_id.to_string(), message.to_string()));
        Ok(())
    }
}

#[test]
fn dispatch_respects_rate_limit() {
    let tmp = tempfile::NamedTempFile::new().expect("tmpfile");
    let path = tmp.path().to_str().unwrap().to_string();

    let db = uptui::storage::Db::open(&path).expect("open db");
    // insert monitor and two alerts
    db.insert_monitor("m1", "m1", "http://localhost").expect("insert");
    let now = chrono::Utc::now();
    let _a1 = db.insert_alert("m1", "first", now).expect("insert alert");
    let _a2 = db.insert_alert("m1", "second", now).expect("insert alert");

    let sender = TestSender::new();
    // first dispatch with rate_limit = 3600 should send only one alert (oldest first but both unsent; rate limit based on last sent will allow first)
    let d = uptui::alert::dispatch_pending_alerts(&sender, &path, Some(3600)).expect("dispatch");
    assert_eq!(d, 1);

    // second dispatch should send 0 due to rate limit
    let d2 = uptui::alert::dispatch_pending_alerts(&sender, &path, Some(3600)).expect("dispatch");
    assert_eq!(d2, 0);

    // dispatch with no rate limit should send remaining alerts
    let d3 = uptui::alert::dispatch_pending_alerts(&sender, &path, None).expect("dispatch");
    assert!(d3 >= 1);
}
