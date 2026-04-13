package models

import "time"

type Feature struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description,omitempty"`
	Impact       string    `json:"impact,omitempty"`
	Complexity   string    `json:"complexity,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	Notes        []Note    `json:"notes,omitempty"`
	Status       string    `json:"status"`
	PromotedFrom string    `json:"promoted_from,omitempty"`
	Plan         string    `json:"plan,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
