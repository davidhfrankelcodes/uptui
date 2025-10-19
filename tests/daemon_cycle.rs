use std::thread;
use std::time::Duration;

use tiny_http::{Server, Response};

#[test]
fn daemon_one_cycle_creates_alerts() {
    // start healthy server (200)
    let healthy = Server::http("0.0.0.0:0").expect("start healthy");
    let healthy_addr = healthy.server_addr();
    let healthy_url = format!("http://{}", healthy_addr);
    let h = thread::spawn(move || {
        for request in healthy.incoming_requests() {
            let response = Response::from_string("ok").with_status_code(200);
            let _ = request.respond(response);
        }
    });

    // start failing server (500)
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

    // create monitors in DB
    let db = uptui::storage::Db::open(&path).expect("open db");
    db.insert_monitor("healthy", "healthy", &healthy_url).expect("insert");
    db.insert_monitor("failing", "failing", &failing_url).expect("insert");

    // allow servers to start
    thread::sleep(Duration::from_millis(50));

    // run one cycle
    uptui::daemon::run_one_cycle(&path).expect("run cycle");

    // inspect alerts
    let alerts_all = db.fetch_alerts(None).expect("fetch alerts");
    // there should be at least one alert for failing monitor
    let failing_alerts: Vec<_> = alerts_all.iter().filter(|a| a.monitor_id == "failing").collect();
    assert!(!failing_alerts.is_empty());

    // healthy should have no alerts
    let healthy_alerts: Vec<_> = alerts_all.iter().filter(|a| a.monitor_id == "healthy").collect();
    assert!(healthy_alerts.is_empty());

    // cleanup
    thread::sleep(Duration::from_millis(50));
    drop(h);
    drop(f);
}
