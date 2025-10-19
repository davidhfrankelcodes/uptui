use std::thread;
use std::time::Duration;

use tiny_http::{Server, Response};

#[test]
fn http_check_and_rotation() {
    // start tiny server in background
    let server = Server::http("0.0.0.0:0").expect("start server");
    let addr = server.server_addr();
    let url = format!("http://{}", addr);

    let handle = thread::spawn(move || {
        for request in server.incoming_requests() {
            let response = Response::from_string("ok").with_status_code(200);
            let _ = request.respond(response);
        }
    });

    // use a temp file for the DB
    let tmp = tempfile::NamedTempFile::new().expect("tmpfile");
    let path = tmp.path().to_str().unwrap().to_string();

    // run a check
    let id = uptui::daemon::run_check_once(&path, "m1", &url).expect("run check");
    assert!(id > 0);

    // verify recent results
    let db = uptui::storage::Db::open(&path).expect("open db");
    let results = db.recent_results("m1").expect("recent");
    assert!(!results.is_empty());
    assert!(results[0].success);
    assert_eq!(results[0].status_code, Some(200));

    // test rotation: delete everything older than 0 days (should delete)
    let deleted = db.rotate(0).expect("rotate");
    assert!(deleted >= 1);

    // stop server
    // tiny_http server will stop when process exits; drop handle by sleeping briefly
    thread::sleep(Duration::from_millis(50));
    drop(handle);
}
