package cmd

import (
	"fmt"

	"bira/internal/models"

	"github.com/spf13/cobra"
)

var contextFull bool

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Show current project context",
	Long:  "Dumps the current project context. Use --full for a detailed breakdown of features and tasks by status.",
	RunE:  runContext,
}

func init() {
	contextCmd.Flags().BoolVar(&contextFull, "full", false, "show full breakdown of features and tasks by status")
	rootCmd.AddCommand(contextCmd)
}

type contextOutput struct {
	ProjectID     string           `json:"project_id"`
	ProjectName   string           `json:"project_name"`
	OpenTaskCount int              `json:"open_task_count"`
	Features      []featureSummary `json:"features,omitempty"`
}

type featureSummary struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	IsBacklog bool           `json:"is_backlog"`
	Tasks     map[string]int `json:"tasks"`
}

func runContext(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject()
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

		// Build task counts per feature
		tasksByFeature := make(map[string]map[string]int)
		for _, t := range tasks {
			if tasksByFeature[t.FeatureID] == nil {
				tasksByFeature[t.FeatureID] = make(map[string]int)
			}
			tasksByFeature[t.FeatureID][t.Status]++
		}

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
			})
		}
	}

	output(ctx, func() {
		fmt.Printf("Project: %s (%s)\n", ctx.ProjectName, ctx.ProjectID)
		fmt.Printf("Open tasks: %d\n", ctx.OpenTaskCount)
		if contextFull && len(ctx.Features) > 0 {
			fmt.Println()
			for _, f := range ctx.Features {
				label := f.Name
				if f.IsBacklog {
					label += " (backlog)"
				}
				fmt.Printf("  Feature: %s [%s] (%s)\n", label, f.Status, f.ID)
				if len(f.Tasks) > 0 {
					for status, count := range f.Tasks {
						fmt.Printf("    %s: %d\n", status, count)
					}
				} else {
					fmt.Printf("    (no tasks)\n")
				}
			}
		}
	})
	return nil
}
