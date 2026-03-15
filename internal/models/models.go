package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

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
	Name             string      `json:"name"`
	Type             MonitorType `json:"type"`
	Target           string      `json:"target"`            // URL for HTTP, host:port for TCP
	Interval         int         `json:"interval"`          // seconds between checks
	Timeout          int         `json:"timeout"`           // seconds before timeout
	Active           bool        `json:"active"`
	AcceptedStatuses string      `json:"accepted_statuses"` // HTTP only; e.g. "200-299,401"; empty = any <400
}

// ParseAcceptedStatuses parses a comma-separated list of HTTP status codes and
// ranges (e.g. "200-299,401,403") into [min,max] pairs. Returns nil, nil for an
// empty string. Returns an error if any token is malformed or out of [100,599].
func ParseAcceptedStatuses(s string) ([][2]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var ranges [][2]int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx := strings.Index(part, "-"); idx > 0 {
			lo, err1 := strconv.Atoi(part[:idx])
			hi, err2 := strconv.Atoi(part[idx+1:])
			if err1 != nil || err2 != nil || lo < 100 || hi > 599 || lo > hi {
				return nil, fmt.Errorf("invalid status range %q", part)
			}
			ranges = append(ranges, [2]int{lo, hi})
		} else {
			code, err := strconv.Atoi(part)
			if err != nil || code < 100 || code > 599 {
				return nil, fmt.Errorf("invalid status code %q", part)
			}
			ranges = append(ranges, [2]int{code, code})
		}
	}
	return ranges, nil
}

type Result struct {
	Timestamp time.Time `json:"ts"`
	Status    Status    `json:"status"`
	Latency   int       `json:"latency_ms"`
	Message   string    `json:"message,omitempty"`
}

type MonitorStatus struct {
	Monitor   Monitor   `json:"monitor"`
	Status    Status    `json:"status"`
	Latency   int       `json:"latency_ms"`
	LastCheck time.Time `json:"last_check"`
	Uptime24h float64   `json:"uptime_24h"`
	Uptime7d  float64   `json:"uptime_7d"`
	Uptime30d float64   `json:"uptime_30d"`
	History   []Result  `json:"history"` // last N results, oldest first
}
