package models

import "time"

type Feature struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	AssignedTo  string    `json:"assigned_to,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	IsBacklog   bool      `json:"is_backlog"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
