package models

const (
	StatusTodo       = "todo"
	StatusInProgress = "in-progress"
	StatusDone       = "done"
	StatusBlocked    = "blocked"
)

var ValidStatuses = []string{StatusTodo, StatusInProgress, StatusDone, StatusBlocked}

func IsValidStatus(s string) bool {
	for _, v := range ValidStatuses {
		if v == s {
			return true
		}
	}
	return false
}
