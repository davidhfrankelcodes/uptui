#![cfg(feature = "smtp")]

use tempfile::tempdir;
use uptui::storage::Db;
use uptui::alert;

#[test]
fn lettre_sender_dispatches_alerts() {
    let dir = tempdir().expect("tempdir");
    let path = dir.path().join("test_smtp.db");
    let path = path.to_str().unwrap().to_string();

    // init db and insert a monitor + alert
    let db = Db::open(&path).expect("open db");
    db.insert_monitor("m1", "m1", "http://localhost/").expect("insert monitor");
    let now = chrono::Utc::now();
    db.insert_alert("m1", "test alert", now).expect("insert alert");

    // build a lettre sender using the stub constructor
    let sender = uptui::smtp_lettre::SmtpSenderL::new_with_stub("uptui@test").expect("build sender");

    let dispatched = alert::dispatch_pending_alerts(&sender, &path, None).expect("dispatch");
    assert_eq!(dispatched, 1);

    // ensure it's marked sent
    let alerts = db.fetch_alerts(None).expect("fetch alerts");
    assert!(alerts.iter().all(|a| a.sent));
}
