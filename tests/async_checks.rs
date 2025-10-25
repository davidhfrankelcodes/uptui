use tempfile::NamedTempFile;
use uptui::daemon::{run_check_once_async, run_one_cycle_async};
use uptui::storage::Db;

#[tokio::test]
async fn test_async_check_success() {
    let temp_db = NamedTempFile::new().unwrap();
    let db_path = temp_db.path().to_str().unwrap();

    // Run an async check against httpbin
    let result_id = run_check_once_async(db_path, "test1", "https://httpbin.org/status/200")
        .await
        .expect("async check should succeed");

    // Verify the result was stored
    let db = Db::open(db_path).unwrap();
    let results = db.recent_results("test1").unwrap();
    
    assert_eq!(results.len(), 1);
    assert_eq!(results[0].id, result_id);
    assert!(results[0].success);
    assert_eq!(results[0].status_code, Some(200));
}

#[tokio::test]
async fn test_async_check_http_error() {
    let temp_db = NamedTempFile::new().unwrap();
    let db_path = temp_db.path().to_str().unwrap();

    // Run an async check against a 500 status
    let result_id = run_check_once_async(db_path, "test2", "https://httpbin.org/status/500")
        .await
        .expect("async check should complete even with HTTP error");

    // Verify the result was stored
    let db = Db::open(db_path).unwrap();
    let results = db.recent_results("test2").unwrap();
    
    assert_eq!(results.len(), 1);
    assert_eq!(results[0].id, result_id);
    assert!(!results[0].success);
    assert_eq!(results[0].status_code, Some(500));
}

#[tokio::test]
async fn test_async_check_network_error() {
    let temp_db = NamedTempFile::new().unwrap();
    let db_path = temp_db.path().to_str().unwrap();

    // Run an async check against an invalid domain
    let result_id = run_check_once_async(db_path, "test3", "https://invalid-domain-that-does-not-exist.com")
        .await
        .expect("async check should complete even with network error");

    // Verify the result was stored
    let db = Db::open(db_path).unwrap();
    let results = db.recent_results("test3").unwrap();
    
    assert_eq!(results.len(), 1);
    assert_eq!(results[0].id, result_id);
    assert!(!results[0].success);
    assert_eq!(results[0].status_code, None); // No HTTP status for network errors
}

#[tokio::test]
async fn test_async_cycle_concurrent_checks() {
    let temp_db = NamedTempFile::new().unwrap();
    let db_path = temp_db.path().to_str().unwrap();
    
    // Set up multiple monitors
    let db = Db::open(db_path).unwrap();
    db.insert_monitor("m1", "Monitor 1", "https://httpbin.org/status/200").unwrap();
    db.insert_monitor("m2", "Monitor 2", "https://httpbin.org/status/404").unwrap();
    db.insert_monitor("m3", "Monitor 3", "https://httpbin.org/status/500").unwrap();

    // Run async cycle with concurrency limit
    run_one_cycle_async(db_path, 2).await.expect("async cycle should complete");

    // Verify all monitors were checked
    let results_m1 = db.recent_results("m1").unwrap();
    let results_m2 = db.recent_results("m2").unwrap();
    let results_m3 = db.recent_results("m3").unwrap();

    assert_eq!(results_m1.len(), 1);
    assert_eq!(results_m2.len(), 1);
    assert_eq!(results_m3.len(), 1);

    // Verify results
    assert!(results_m1[0].success);
    assert_eq!(results_m1[0].status_code, Some(200));
    
    assert!(!results_m2[0].success);
    assert_eq!(results_m2[0].status_code, Some(404));
    
    assert!(!results_m3[0].success);
    assert_eq!(results_m3[0].status_code, Some(500));

    // Verify alerts were created for failed checks
    let alerts = db.fetch_alerts(None).unwrap();
    assert_eq!(alerts.len(), 2); // m2 and m3 should have alerts
    
    let alert_monitors: Vec<&str> = alerts.iter().map(|a| a.monitor_id.as_str()).collect();
    assert!(alert_monitors.contains(&"m2"));
    assert!(alert_monitors.contains(&"m3"));
}

#[tokio::test]
async fn test_async_cycle_empty_monitors() {
    let temp_db = NamedTempFile::new().unwrap();
    let db_path = temp_db.path().to_str().unwrap();
    
    // Initialize empty database
    let _db = Db::open(db_path).unwrap();

    // Run async cycle with no monitors
    let result = run_one_cycle_async(db_path, 5).await;
    
    // Should complete successfully even with no monitors
    assert!(result.is_ok());
}