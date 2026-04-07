package cmd

import (
	"fmt"
	"io"

	"bira/internal/models"

	"github.com/spf13/cobra"
)

func newContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Show current project context",
		Long:  "Dumps the current project context. Use --full for a detailed breakdown of features and tasks by status.",
		RunE:  runContext,
	}
	cmd.Flags().Bool("full", false, "show full breakdown of features and tasks by status")
	return cmd
}

type contextOutput struct {
	ProjectID          string           `json:"project_id"`
	ProjectName        string           `json:"project_name"`
	OpenTaskCount      int              `json:"open_task_count"`
	ReadyTaskCount     int              `json:"ready_task_count,omitempty"`
	ClaimedTaskCount   int              `json:"claimed_task_count,omitempty"`
	InvalidTaskCount   int              `json:"invalid_task_count,omitempty"`
	ActiveSessionCount int              `json:"active_session_count,omitempty"`
	Features           []featureSummary `json:"features,omitempty"`
}

type featureAgentSummary struct {
	Ready   int `json:"ready"`
	Claimed int `json:"claimed"`
	Invalid int `json:"invalid"`
}

type featureSummary struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	Status    string               `json:"status"`
	IsBacklog bool                 `json:"is_backlog"`
	Tasks     map[string]int       `json:"tasks"`
	Agent     *featureAgentSummary `json:"agent,omitempty"`
}

func runContext(cmd *cobra.Command, args []string) error {
	contextFull, _ := cmd.Flags().GetBool("full")
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}

	project, err := loadProject(projectID)
	if err != nil {
		return err
	}

	tasks, err := loadAllTasks(projectID)
	if err != nil {
		return err
	}

	openCount := 0
	for _, t := range tasks {
		if t.Status != models.StatusDone {
			openCount++
		}
	}

	ctx := contextOutput{
		ProjectID:     project.ID,
		ProjectName:   project.Name,
		OpenTaskCount: openCount,
	}

	if contextFull {
		features, err := loadAllFeatures(projectID)
		if err != nil {
			return err
		}

		sessions, err := loadAllSessions(projectID)
		if err != nil {
			return err
		}
		ctx.ActiveSessionCount = len(sessions)

		timeoutMinutes := loadClaimTimeout(cmd)
		tasksByID := tasksToMap(tasks)
		featuresByID := featuresToMap(features)

		// Per-task readiness and global counters.
		type taskReadinessTuple struct {
			task      models.Task
			readiness TaskReadiness
		}
		taskReadiness := make([]taskReadinessTuple, len(tasks))
		for i, t := range tasks {
			r := computeReadiness(t, tasksByID, featuresByID, timeoutMinutes)
			taskReadiness[i] = taskReadinessTuple{task: t, readiness: r}
			if r.ReadyForAgent {
				ctx.ReadyTaskCount++
			}
			// Claimed: ClaimedBy set and not stale
			if t.ClaimedBy != "" && !r.StaleClaim {
				ctx.ClaimedTaskCount++
			}
			// Invalid: status==todo, unclaimed (or stale), not ready
			if t.Status == models.StatusTodo && t.ClaimedBy == "" && !r.ReadyForAgent {
				ctx.InvalidTaskCount++
			}
		}

		// Build task counts per feature
		tasksByFeature := make(map[string]map[string]int)
		agentByFeature := make(map[string]*featureAgentSummary)
		for _, tr := range taskReadiness {
			t := tr.task
			r := tr.readiness
			if tasksByFeature[t.FeatureID] == nil {
				tasksByFeature[t.FeatureID] = make(map[string]int)
				agentByFeature[t.FeatureID] = &featureAgentSummary{}
			}
			tasksByFeature[t.FeatureID][t.Status]++
			ag := agentByFeature[t.FeatureID]
			if r.ReadyForAgent {
				ag.Ready++
			}
			if t.ClaimedBy != "" && !r.StaleClaim {
				ag.Claimed++
			}
			if t.Status == models.StatusTodo && t.ClaimedBy == "" && !r.ReadyForAgent {
				ag.Invalid++
			}
		}

		// Sort features: backlog first, then created_at ASC, id ASC
		sortFeatures(features)

		for _, f := range features {
			counts := tasksByFeature[f.ID]
			if counts == nil {
				counts = make(map[string]int)
			}
			ctx.Features = append(ctx.Features, featureSummary{
				ID:        f.ID,
				Name:      f.Name,
				Status:    f.Status,
				IsBacklog: f.IsBacklog,
				Tasks:     counts,
				Agent:     agentByFeature[f.ID],
			})
		}
	}

	output(cmd, ctx, func(w io.Writer) {
		fmt.Fprintf(w, "Project: %s (%s)\n", ctx.ProjectName, ctx.ProjectID)
		fmt.Fprintf(w, "Open tasks: %d\n", ctx.OpenTaskCount)
		if contextFull {
			fmt.Fprintf(w, "Ready: %d  Claimed: %d  Invalid: %d  Sessions: %d\n",
				ctx.ReadyTaskCount, ctx.ClaimedTaskCount, ctx.InvalidTaskCount, ctx.ActiveSessionCount)
		}
		if contextFull && len(ctx.Features) > 0 {
			fmt.Fprintln(w)
			for _, f := range ctx.Features {
				label := f.Name
				if f.IsBacklog {
					label += " (backlog)"
				}
				fmt.Fprintf(w, "  Feature: %s [%s] (%s)\n", label, f.Status, f.ID)
				if len(f.Tasks) > 0 {
					for status, count := range f.Tasks {
						fmt.Fprintf(w, "    %s: %d\n", status, count)
					}
				} else {
					fmt.Fprintf(w, "    (no tasks)\n")
				}
			}
		}
	})
	return nil
}
