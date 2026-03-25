package models

import "time"

type Note struct {
	Timestamp time.Time `json:"timestamp"`
	Body      string    `json:"body"`
}
