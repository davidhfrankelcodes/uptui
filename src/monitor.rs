// monitor abstractions - placeholder

#[derive(Debug, Clone)]
pub enum MonitorType {
    Tcp { host: String, port: u16 },
    Ping { host: String },
    Http { url: String, accepted_status: (u16, u16) },
}

#[derive(Debug, Clone)]
pub struct Monitor {
    pub id: String,
    pub name: String,
    pub mtype: MonitorType,
}
