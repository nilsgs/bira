package models

import "time"

type Idea struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Priority    string    `json:"priority,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Notes       []Note    `json:"notes,omitempty"`
	Status      string    `json:"status"`
	Plan        string    `json:"plan,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
