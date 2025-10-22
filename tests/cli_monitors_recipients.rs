use assert_cmd::Command;
use tempfile::tempdir;
use uptui::storage::Db;

#[test]
fn cli_monitor_add_with_recipients() {
    let dir = tempdir().expect("tempdir");
    let dbpath = dir.path().join("cli_monitors_recipients.db");
    let dbpath_s = dbpath.to_str().unwrap().to_string();

    // add monitor with recipients
    let mut cmd = Command::cargo_bin("uptui").expect("binary");
    cmd.arg("monitor").arg("add").arg("m-recip").arg("My Monitor").arg("http://localhost/")
        .arg("--db").arg(&dbpath_s)
        .arg("--recipients").arg("ops@example.org,oncall@example.org");
    cmd.assert().success();

    // verify in DB
    let db = Db::open(&dbpath_s).expect("open db");
    let m = db.get_monitor("m-recip").expect("get monitor").expect("exists");
    assert_eq!(m.recipients.unwrap(), "ops@example.org,oncall@example.org");
}
