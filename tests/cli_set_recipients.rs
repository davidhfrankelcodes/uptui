use assert_cmd::Command;
use tempfile::tempdir;
use uptui::storage::Db;

#[test]
fn cli_monitor_set_recipients() {
    let dir = tempdir().expect("tempdir");
    let dbpath = dir.path().join("cli_set_recipients.db");
    let dbpath_s = dbpath.to_str().unwrap().to_string();

    // add a monitor first
    let mut cmd = Command::cargo_bin("uptui").expect("binary");
    cmd.arg("monitor").arg("add").arg("m2").arg("Name").arg("http://localhost/")
        .arg("--db").arg(&dbpath_s);
    cmd.assert().success();

    // set recipients
    let mut cmd2 = Command::cargo_bin("uptui").expect("binary");
    cmd2.arg("monitor").arg("set-recipients").arg("m2").arg("--recipients").arg("a@x.com,b@x.com")
        .arg("--db").arg(&dbpath_s);
    cmd2.assert().success();

    let db = Db::open(&dbpath_s).expect("open db");
    let m = db.get_monitor("m2").expect("get monitor").expect("exists");
    assert_eq!(m.recipients.unwrap(), "a@x.com,b@x.com");
}
