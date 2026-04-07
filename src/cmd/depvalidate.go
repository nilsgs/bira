package cmd

import (
	"fmt"

	"bira/internal/models"
)

// missingDepError is returned when a depends-on ID does not exist in the project.
// Execute() maps this to exit code 2.
type missingDepError struct {
	depID string
}

func (e *missingDepError) Error() string {
	return fmt.Sprintf("dependency task not found: %s", e.depID)
}

// validateDependsOn checks the proposed dependency list for a task.
// taskID is the task being created or updated (used for self-dep and cycle checks).
// newDeps is the proposed list of dependency IDs.
// allTasksByID is the full project task map.
func validateDependsOn(taskID string, newDeps []string, allTasksByID map[string]models.Task) error {
	// Self-dependency
	for _, id := range newDeps {
		if id == taskID {
			return fmt.Errorf("task cannot depend on itself")
		}
	}

	// Duplicates
	seen := make(map[string]bool, len(newDeps))
	for _, id := range newDeps {
		if seen[id] {
			return fmt.Errorf("duplicate dependency ID: %s", id)
		}
		seen[id] = true
	}

	// Existence
	for _, id := range newDeps {
		if _, ok := allTasksByID[id]; !ok {
			return &missingDepError{depID: id}
		}
	}

	// Cycle detection: build an in-memory copy of the dep graph with the proposed
	// new edges, then check whether taskID is reachable from any of newDeps.
	reachable := func(start, target string, graph map[string][]string) bool {
		visited := map[string]bool{}
		stack := []string{start}
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if cur == target {
				return true
			}
			if visited[cur] {
				continue
			}
			visited[cur] = true
			stack = append(stack, graph[cur]...)
		}
		return false
	}

	// Build the current dep graph from allTasksByID, overriding taskID's deps with newDeps.
	graph := make(map[string][]string, len(allTasksByID))
	for id, t := range allTasksByID {
		if id == taskID {
			graph[id] = newDeps
		} else {
			graph[id] = t.DependsOn
		}
	}

	for _, depID := range newDeps {
		if reachable(depID, taskID, graph) {
			return fmt.Errorf("circular dependency detected: %s → %s creates a cycle", taskID, depID)
		}
	}

	return nil
}
