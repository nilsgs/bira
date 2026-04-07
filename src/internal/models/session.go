package models

import "time"

type Session struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Label     string    `json:"label,omitempty"`
	StartedAt time.Time `json:"started_at"`
}
