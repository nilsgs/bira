package cmd

import (
	"time"

	"bira/internal/models"
)

// TaskReadiness holds derived (non-persisted) readiness fields for a task.
type TaskReadiness struct {
	ReadyForAgent     bool              `json:"ready_for_agent"`
	StaleClaim        bool              `json:"stale_claim"`
	MissingFields     []string          `json:"missing_fields"`
	BlockedReasons    []string          `json:"blocked_reasons"`
	DependencySummary DependencySummary `json:"dependency_summary"`
}

// DependencySummary summarises the state of a task's declared dependencies.
type DependencySummary struct {
	Total   int `json:"total"`
	Done    int `json:"done"`
	Missing int `json:"missing"`
	NotDone int `json:"not_done"`
}

// ResolvedDep is one entry in the resolved dependencies array on task show.
type ResolvedDep struct {
	ID     string `json:"id"`
	Exists bool   `json:"exists"`
	Status string `json:"status,omitempty"`
	Title  string `json:"title,omitempty"`
}

// computeReadiness derives readiness fields for a task.
// allTasksByID and allFeaturesByID must include all tasks/features in the project.
// timeoutMinutes is the claim timeout; use config.DefaultClaimTimeoutMinutes when not overridden.
func computeReadiness(
	task models.Task,
	allTasksByID map[string]models.Task,
	allFeaturesByID map[string]models.Feature,
	timeoutMinutes int,
) TaskReadiness {
	var r TaskReadiness
	r.MissingFields = []string{}
	r.BlockedReasons = []string{}

	// Stale claim check.
	if task.ClaimedBy != "" && task.ClaimedAt != nil {
		timeout := time.Duration(timeoutMinutes) * time.Minute
		if time.Since(*task.ClaimedAt) > timeout {
			r.StaleClaim = true
		}
	}

	// missing_fields
	if task.Description == "" {
		r.MissingFields = append(r.MissingFields, "missing_description")
	}
	if len(task.AcceptanceCriteria) == 0 {
		r.MissingFields = append(r.MissingFields, "missing_acceptance_criteria")
	}
	_, featureExists := allFeaturesByID[task.FeatureID]
	if !featureExists {
		r.MissingFields = append(r.MissingFields, "missing_feature")
	}

	// blocked_reasons
	if task.Status != models.StatusTodo {
		r.BlockedReasons = append(r.BlockedReasons, "status_not_todo")
	}
	if task.ClaimedBy != "" && !r.StaleClaim {
		r.BlockedReasons = append(r.BlockedReasons, "already_claimed")
	}
	if featureExists {
		f := allFeaturesByID[task.FeatureID]
		if f.Status == models.StatusDone {
			r.BlockedReasons = append(r.BlockedReasons, "feature_done")
		}
	}

	// dependency summary
	r.DependencySummary.Total = len(task.DependsOn)
	for _, depID := range task.DependsOn {
		dep, ok := allTasksByID[depID]
		if !ok {
			r.DependencySummary.Missing++
			r.BlockedReasons = append(r.BlockedReasons, "dependency_missing:"+depID)
		} else if dep.Status == models.StatusDone {
			r.DependencySummary.Done++
		} else {
			r.DependencySummary.NotDone++
			r.BlockedReasons = append(r.BlockedReasons, "dependency_not_done:"+depID)
		}
	}

	// ready_for_agent: no missing fields, no blocked reasons (stale claim is ok)
	r.ReadyForAgent = len(r.MissingFields) == 0 && len(r.BlockedReasons) == 0

	return r
}
