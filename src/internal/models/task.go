package models

import "time"

type Task struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	FeatureID   string    `json:"feature_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	AssignedTo  string    `json:"assigned_to,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	DependsOn   []string  `json:"depends_on,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
