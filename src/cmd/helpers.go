package cmd

import (
	"os"
	"sort"

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
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].CreatedAt.Equal(tasks[j].CreatedAt) {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
}

// sortFeatures sorts features deterministically:
// backlog first (is_backlog DESC), then created_at ASC, then id ASC.
func sortFeatures(features []models.Feature) {
	sort.Slice(features, func(i, j int) bool {
		if features[i].IsBacklog != features[j].IsBacklog {
			return features[i].IsBacklog // backlog sorts first
		}
		if features[i].CreatedAt.Equal(features[j].CreatedAt) {
			return features[i].ID < features[j].ID
		}
		return features[i].CreatedAt.Before(features[j].CreatedAt)
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
