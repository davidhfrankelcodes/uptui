use std::thread;
use std::time::Duration;
use std::sync::{Arc, Mutex};

use tiny_http::{Server, Response};

struct TestSender {
    sent: Arc<Mutex<Vec<(String, String)>>>,
}
impl TestSender {
    fn new() -> Self { Self { sent: Arc::new(Mutex::new(Vec::new())) } }
}
impl uptui::alert::Sender for TestSender {
    fn send(&self, monitor_id: &str, message: &str) -> anyhow::Result<()> {
        let mut s = self.sent.lock().unwrap();
        s.push((monitor_id.to_string(), message.to_string()));
        Ok(())
    }
}

#[test]
fn run_cycle_and_dispatch_sends_alerts() {
    // failing server
    let failing = Server::http("0.0.0.0:0").expect("start failing");
    let failing_addr = failing.server_addr();
    let failing_url = format!("http://{}", failing_addr);
    let f = thread::spawn(move || {
        for request in failing.incoming_requests() {
            let response = Response::from_string("err").with_status_code(500);
            let _ = request.respond(response);
        }
    });

    let tmp = tempfile::NamedTempFile::new().expect("tmpfile");
    let path = tmp.path().to_str().unwrap().to_string();

    let db = uptui::storage::Db::open(&path).expect("open db");
    db.insert_monitor("mrun", "mrun", &failing_url).expect("insert monitor");

    thread::sleep(Duration::from_millis(50));

    let sender = TestSender::new();
    let dispatched = uptui::daemon::run_cycle_and_dispatch(&path, &sender, None).expect("run and dispatch");
    assert!(dispatched >= 1);

    drop(f);
}
