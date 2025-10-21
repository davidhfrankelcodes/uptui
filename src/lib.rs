//! uptui library root

pub mod cli;
pub mod config;
pub mod daemon;
pub mod tui;
pub mod monitor;
pub mod data;
pub mod storage;
pub mod alert;

pub use crate::config::Config;
