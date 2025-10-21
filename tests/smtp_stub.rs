#[test]
fn smtp_sender_stub_ok() {
    use uptui::alert::Sender;
    let s = uptui::smtp::SmtpSender::new("noreply@example.org", None);
    let r = s.send("m1", "test message");
    assert!(r.is_ok());
}
