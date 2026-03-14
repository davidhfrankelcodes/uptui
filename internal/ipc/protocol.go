package ipc

import "uptui/internal/models"

type Action string

const (
	ActionList   Action = "list"
	ActionAdd    Action = "add"
	ActionDelete Action = "delete"
	ActionPause  Action = "pause"
	ActionResume Action = "resume"
	ActionEdit   Action = "edit"
	ActionReload Action = "reload"
)

type Request struct {
	Action  Action          `json:"action"`
	Monitor *models.Monitor `json:"monitor,omitempty"`
	Name    string          `json:"name,omitempty"`     // target monitor name (delete/pause/resume/edit)
	OldName string          `json:"old_name,omitempty"` // original name when editing (rename support)
}

type Response struct {
	OK       bool                    `json:"ok"`
	Error    string                  `json:"error,omitempty"`
	Monitors []*models.MonitorStatus `json:"monitors,omitempty"`
	Monitor  *models.MonitorStatus   `json:"monitor,omitempty"`
}
