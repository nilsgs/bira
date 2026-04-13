package cmd

import (
	"fmt"
	"io"

	"bira/internal/store"

	"github.com/spf13/cobra"
)

func newContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Show a snapshot of the current project",
		RunE:  runContext,
	}
	cmd.Flags().Bool("full", false, "include per-entity status breakdown")
	return cmd
}

func runContext(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	full, _ := cmd.Flags().GetBool("full")

	meta, err := store.LoadProjectMeta(projectID)
	if err != nil {
		return err
	}

	ideas, err := loadAllIdeas(projectID)
	if err != nil {
		return err
	}
	bugs, err := loadAllBugs(projectID)
	if err != nil {
		return err
	}
	features, err := loadAllFeatures(projectID)
	if err != nil {
		return err
	}

	type statusCount struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	countByStatus := func(statuses []string) []statusCount {
		m := map[string]int{}
		for _, s := range statuses {
			m[s]++
		}
		seen := map[string]bool{}
		var out []statusCount
		for _, s := range statuses {
			if !seen[s] {
				seen[s] = true
				out = append(out, statusCount{Status: s, Count: m[s]})
			}
		}
		return out
	}

	if !full {
		type summary struct {
			ProjectID   string `json:"project_id"`
			ProjectName string `json:"project_name"`
			Ideas       int    `json:"ideas"`
			Bugs        int    `json:"bugs"`
			Features    int    `json:"features"`
		}
		s := summary{
			ProjectID:   projectID,
			ProjectName: meta.Name,
			Ideas:       len(ideas),
			Bugs:        len(bugs),
			Features:    len(features),
		}
		output(cmd, s, func(w io.Writer) {
			fmt.Fprintf(w, "Project:  %s (%s)\n", s.ProjectName, s.ProjectID)
			fmt.Fprintf(w, "Ideas:    %d\n", s.Ideas)
			fmt.Fprintf(w, "Bugs:     %d\n", s.Bugs)
			fmt.Fprintf(w, "Features: %d\n", s.Features)
		})
		return nil
	}

	ideaStatuses := make([]string, len(ideas))
	for i, x := range ideas {
		ideaStatuses[i] = x.Status
	}
	bugStatuses := make([]string, len(bugs))
	for i, x := range bugs {
		bugStatuses[i] = x.Status
	}
	featStatuses := make([]string, len(features))
	for i, x := range features {
		featStatuses[i] = x.Status
	}

	type fullSummary struct {
		ProjectID    string        `json:"project_id"`
		ProjectName  string        `json:"project_name"`
		IdeaCount    int           `json:"idea_count"`
		BugCount     int           `json:"bug_count"`
		FeatureCount int           `json:"feature_count"`
		Ideas        []statusCount `json:"ideas"`
		Bugs         []statusCount `json:"bugs"`
		Features     []statusCount `json:"features"`
	}
	fs := fullSummary{
		ProjectID:    projectID,
		ProjectName:  meta.Name,
		IdeaCount:    len(ideas),
		BugCount:     len(bugs),
		FeatureCount: len(features),
		Ideas:        countByStatus(ideaStatuses),
		Bugs:         countByStatus(bugStatuses),
		Features:     countByStatus(featStatuses),
	}
	output(cmd, fs, func(w io.Writer) {
		fmt.Fprintf(w, "Project:  %s (%s)\n\n", fs.ProjectName, fs.ProjectID)
		fmt.Fprintf(w, "IDEAS (%d):\n", fs.IdeaCount)
		for _, sc := range fs.Ideas {
			fmt.Fprintf(w, "  %-12s %d\n", sc.Status, sc.Count)
		}
		fmt.Fprintf(w, "\nBUGS (%d):\n", fs.BugCount)
		for _, sc := range fs.Bugs {
			fmt.Fprintf(w, "  %-12s %d\n", sc.Status, sc.Count)
		}
		fmt.Fprintf(w, "\nFEATURES (%d):\n", fs.FeatureCount)
		for _, sc := range fs.Features {
			fmt.Fprintf(w, "  %-12s %d\n", sc.Status, sc.Count)
		}
	})
	return nil
}
