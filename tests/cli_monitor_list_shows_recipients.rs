use assert_cmd::Command;
use predicates::prelude::*;
use tempfile::tempdir;

#[test]
fn cli_monitor_list_shows_recipients() {
    let dir = tempdir().expect("tempdir");
    let dbpath = dir.path().join("cli_list_recipients.db");
    let dbpath_s = dbpath.to_str().unwrap().to_string();

    // add monitor without recipients
    let mut cmd = Command::cargo_bin("uptui").expect("binary");
    cmd.arg("monitor").arg("add").arg("m-no").arg("No Recip").arg("http://localhost/")
        .arg("--db").arg(&dbpath_s);
    cmd.assert().success();

    // add monitor with recipients
    let mut cmd2 = Command::cargo_bin("uptui").expect("binary");
    cmd2.arg("monitor").arg("add").arg("m-yes").arg("Yes Recip").arg("http://localhost/")
        .arg("--db").arg(&dbpath_s).arg("--recipients").arg("a@x.com,b@x.com");
    cmd2.assert().success();

    // list and assert output contains recipients
    let mut cmd3 = Command::cargo_bin("uptui").expect("binary");
    cmd3.arg("monitor").arg("list").arg("--db").arg(&dbpath_s);
    let assert = cmd3.assert().success().stdout(predicate::str::contains("m-no\tNo Recip\thttp://localhost/\t-"))
        .stdout(predicate::str::contains("m-yes\tYes Recip\thttp://localhost/\ta@x.com,b@x.com"));
    assert;
}
