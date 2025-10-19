use assert_cmd::Command;
use std::fs;

#[test]
fn config_example_roundtrip() {
    let cfg = uptui::Config::example();
    let yaml = serde_yaml::to_string(&cfg).expect("serialize");
    let parsed: uptui::Config = serde_yaml::from_str(&yaml).expect("deserialize");
    assert_eq!(parsed.db.path, cfg.db.path);
}

#[test]
fn cli_init_writes_file() {
    let mut cmd = Command::cargo_bin("uptui").expect("binary exists");
    let tmp = tempfile::NamedTempFile::new().expect("tempfile");
    let path = tmp.path().to_str().unwrap().to_string();
    cmd.args(["init", "--path", &path]);
    cmd.assert().success();
    let contents = fs::read_to_string(&path).expect("read file");
    assert!(contents.contains("db:"));
}
