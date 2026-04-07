package cmd

import (
	"cmp"
	"os"
	"slices"

	"bira/internal/models"
)

// resolveSession returns the session ID to use for claim/release/done operations.
// flagValue takes priority; falls back to $BIRA_SESSION.
func resolveSession(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv("BIRA_SESSION")
}

// sortTasks sorts tasks deterministically: created_at ASC, then id ASC.
func sortTasks(tasks []models.Task) {
	slices.SortFunc(tasks, func(a, b models.Task) int {
		if n := a.CreatedAt.Compare(b.CreatedAt); n != 0 {
			return n
		}
		return cmp.Compare(a.ID, b.ID)
	})
}

// sortFeatures sorts features deterministically:
// backlog first (is_backlog DESC), then created_at ASC, then id ASC.
func sortFeatures(features []models.Feature) {
	slices.SortFunc(features, func(a, b models.Feature) int {
		if a.IsBacklog != b.IsBacklog {
			if a.IsBacklog {
				return -1
			}
			return 1
		}
		if n := a.CreatedAt.Compare(b.CreatedAt); n != 0 {
			return n
		}
		return cmp.Compare(a.ID, b.ID)
	})
}

// tasksToMap converts a task slice to a map keyed by ID.
func tasksToMap(tasks []models.Task) map[string]models.Task {
	m := make(map[string]models.Task, len(tasks))
	for _, t := range tasks {
		m[t.ID] = t
	}
	return m
}

// featuresToMap converts a feature slice to a map keyed by ID.
func featuresToMap(features []models.Feature) map[string]models.Feature {
	m := make(map[string]models.Feature, len(features))
	for _, f := range features {
		m[f.ID] = f
	}
	return m
}
