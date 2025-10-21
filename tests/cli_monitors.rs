use assert_cmd::Command;
use predicates::prelude::*;

#[test]
fn cli_monitor_add_list_remove() {
    let tmp = tempfile::NamedTempFile::new().expect("tmpfile");
    let path = tmp.path().to_str().unwrap().to_string();

    // add monitor
    let mut cmd = Command::cargo_bin("uptui").expect("binary");
    cmd.args(["--db", &path, "monitor", "add", "mcli", "cli monitor", "http://example.local"]);
    cmd.assert().success().stdout(predicate::str::contains("monitor mcli added"));

    // list monitors
    let mut cmd2 = Command::cargo_bin("uptui").expect("binary");
    cmd2.args(["--db", &path, "monitor", "list"]);
    cmd2.assert().success().stdout(predicate::str::contains("mcli\tcli monitor\thttp://example.local"));

    // remove monitor
    let mut cmd3 = Command::cargo_bin("uptui").expect("binary");
    cmd3.args(["--db", &path, "monitor", "remove", "mcli"]);
    cmd3.assert().success().stdout(predicate::str::contains("deleted monitor mcli"));
}
