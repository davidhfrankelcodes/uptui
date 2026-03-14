package models

import "time"

type MonitorType string

const (
	HTTP MonitorType = "http"
	TCP  MonitorType = "tcp"
)

type Status string

const (
	StatusUp      Status = "up"
	StatusDown    Status = "down"
	StatusPending Status = "pending"
	StatusPaused  Status = "paused"
)

type Monitor struct {
	ID       int         `json:"id"`
	Name     string      `json:"name"`
	Type     MonitorType `json:"type"`
	Target   string      `json:"target"` // URL for HTTP, host:port for TCP
	Interval int         `json:"interval"` // seconds between checks
	Timeout  int         `json:"timeout"`  // seconds before timeout
	Active   bool        `json:"active"`
}

type Result struct {
	Timestamp time.Time `json:"ts"`
	Status    Status    `json:"status"`
	Latency   int       `json:"latency_ms"`
	Message   string    `json:"message,omitempty"`
}

type MonitorStatus struct {
	Monitor   Monitor  `json:"monitor"`
	Status    Status   `json:"status"`
	Latency   int      `json:"latency_ms"`
	LastCheck time.Time `json:"last_check"`
	Uptime24h float64  `json:"uptime_24h"`
	History   []Result `json:"history"` // last N results, oldest first
}
