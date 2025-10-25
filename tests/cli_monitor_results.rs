use assert_cmd::Command;
use tempfile::NamedTempFile;
use uptui::storage::Db;
use chrono::Utc;

#[test]
fn test_cli_monitor_results_command() {
    let temp_db = NamedTempFile::new().unwrap();
    let db_path = temp_db.path().to_str().unwrap();

    // Set up test data
    let db = Db::open(db_path).unwrap();
    db.insert_monitor("test1", "Test Monitor", "https://example.com").unwrap();
    
    // Insert some test results
    let now = Utc::now();
    db.insert_result("test1", true, Some(200), now).unwrap();
    db.insert_result("test1", false, Some(500), now).unwrap();
    db.insert_result("test1", false, None, now).unwrap();

    // Test the results command
    let mut cmd = Command::cargo_bin("uptui").unwrap();
    let output = cmd
        .args(&["--db", db_path, "monitor", "results", "test1"])
        .output()
        .unwrap();

    assert!(output.status.success());
    
    let stdout = String::from_utf8(output.stdout).unwrap();
    
    // Check that the output contains expected elements
    assert!(stdout.contains("Recent results for monitor 'Test Monitor'"));
    assert!(stdout.contains("https://example.com"));
    assert!(stdout.contains("ID\tSuccess\tStatus\tTimestamp"));
    assert!(stdout.contains("✓\t200"));  // Success with status
    assert!(stdout.contains("✗\t500"));  // Failure with status
    assert!(stdout.contains("✗\t-"));    // Failure without status
}

#[test]
fn test_cli_monitor_results_with_limit() {
    let temp_db = NamedTempFile::new().unwrap();
    let db_path = temp_db.path().to_str().unwrap();

    // Set up test data
    let db = Db::open(db_path).unwrap();
    db.insert_monitor("test1", "Test Monitor", "https://example.com").unwrap();
    
    // Insert multiple results
    let now = Utc::now();
    for i in 0..5 {
        db.insert_result("test1", i % 2 == 0, Some(200), now).unwrap();
    }

    // Test with limit
    let mut cmd = Command::cargo_bin("uptui").unwrap();
    let output = cmd
        .args(&["--db", db_path, "monitor", "results", "test1", "--limit", "3"])
        .output()
        .unwrap();

    assert!(output.status.success());
    
    let stdout = String::from_utf8(output.stdout).unwrap();
    
    // Should show "more results available" message
    assert!(stdout.contains("more results available"));
    
    // Count the number of result lines (excluding headers)
    let result_lines: Vec<&str> = stdout.lines()
        .filter(|line| line.chars().next().map_or(false, |c| c.is_ascii_digit()))
        .collect();
    assert_eq!(result_lines.len(), 3);
}

#[test]
fn test_cli_monitor_results_nonexistent_monitor() {
    let temp_db = NamedTempFile::new().unwrap();
    let db_path = temp_db.path().to_str().unwrap();

    // Initialize empty database
    let _db = Db::open(db_path).unwrap();

    // Test with non-existent monitor
    let mut cmd = Command::cargo_bin("uptui").unwrap();
    let output = cmd
        .args(&["--db", db_path, "monitor", "results", "nonexistent"])
        .output()
        .unwrap();

    assert!(output.status.success());
    
    let stdout = String::from_utf8(output.stdout).unwrap();
    assert!(stdout.contains("monitor nonexistent not found"));
}

#[test]
fn test_cli_monitor_results_no_results() {
    let temp_db = NamedTempFile::new().unwrap();
    let db_path = temp_db.path().to_str().unwrap();

    // Set up monitor with no results
    let db = Db::open(db_path).unwrap();
    db.insert_monitor("test1", "Test Monitor", "https://example.com").unwrap();

    // Test with monitor that has no results
    let mut cmd = Command::cargo_bin("uptui").unwrap();
    let output = cmd
        .args(&["--db", db_path, "monitor", "results", "test1"])
        .output()
        .unwrap();

    assert!(output.status.success());
    
    let stdout = String::from_utf8(output.stdout).unwrap();
    assert!(stdout.contains("no results found for monitor test1"));
}