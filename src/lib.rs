//! uptui library root

pub mod cli;
pub mod config;
pub mod daemon;
pub mod tui;
pub mod monitor;
pub mod data;
pub mod storage;
pub mod alert;
pub mod smtp;
#[cfg(feature = "smtp")]
pub mod smtp_lettre;

pub use crate::config::Config;
