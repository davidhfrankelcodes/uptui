package ipc

import "uptui/internal/models"

type Action string

const (
	ActionList   Action = "list"
	ActionAdd    Action = "add"
	ActionDelete Action = "delete"
	ActionPause  Action = "pause"
	ActionResume Action = "resume"
)

type Request struct {
	Action  Action          `json:"action"`
	Monitor *models.Monitor `json:"monitor,omitempty"`
	ID      int             `json:"id,omitempty"`
}

type Response struct {
	OK       bool                    `json:"ok"`
	Error    string                  `json:"error,omitempty"`
	Monitors []*models.MonitorStatus `json:"monitors,omitempty"`
	Monitor  *models.MonitorStatus   `json:"monitor,omitempty"`
}
