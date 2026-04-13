package models

import "time"

type Bug struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Criticality string    `json:"criticality,omitempty"`
	ReportedBy  string    `json:"reported_by,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Notes       []Note    `json:"notes,omitempty"`
	Status      string    `json:"status"`
	Plan        string    `json:"plan,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
