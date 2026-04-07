package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bira/internal/models"
	"bira/internal/store"

	"github.com/spf13/cobra"
)

func newFeatureCmd() *cobra.Command {
	featureCmd := &cobra.Command{
		Use:   "feature",
		Short: "Manage features",
	}

	// --- add ---

	addCmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new feature",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureAdd,
	}
	addCmd.Flags().String("desc", "", "feature description")
	addCmd.Flags().String("assign", "", "assigned agent/user")
	addCmd.Flags().String("tags", "", "comma-separated tags")

	// --- list ---

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List features in the current project",
		RunE:  runFeatureList,
	}
	listCmd.Flags().String("status", "", "filter by status")

	// --- show ---

	showCmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show feature details",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureShow,
	}

	// --- update ---

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a feature",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureUpdate,
	}
	updateCmd.Flags().String("status", "", "new status")
	updateCmd.Flags().String("desc", "", "new description")
	updateCmd.Flags().String("assign", "", "new assignee")
	updateCmd.Flags().String("tags", "", "new comma-separated tags")

	// --- delete ---

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a feature",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureDelete,
	}
	deleteCmd.Flags().String("move-tasks-to", "", "move tasks to this feature ID before deleting")

	// --- note ---

	noteCmd := &cobra.Command{
		Use:   "note <id> <message>",
		Short: "Append a note to a feature",
		Args:  cobra.ExactArgs(2),
		RunE:  runFeatureNote,
	}

	featureCmd.AddCommand(addCmd, listCmd, showCmd, updateCmd, deleteCmd, noteCmd)
	return featureCmd
}

// --- implementations ---

func runFeatureAdd(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	featureAddDesc, _ := cmd.Flags().GetString("desc")
	featureAddAssign, _ := cmd.Flags().GetString("assign")
	featureAddTags, _ := cmd.Flags().GetString("tags")
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return err
	}

	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return err
	}
	defer lock.Release()

	now := time.Now().UTC()
	feature := models.Feature{
		ID:          store.NewID(),
		ProjectID:   projectID,
		Name:        args[0],
		Description: featureAddDesc,
		Status:      models.StatusTodo,
		AssignedTo:  featureAddAssign,
		Tags:        parseTags(featureAddTags),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.SaveJSON(path, &feature); err != nil {
		return fmt.Errorf("save feature: %w", err)
	}

	output(cmd, &feature, func(w io.Writer) {
		fmt.Fprintf(w, "Created feature %q (%s)\n", feature.Name, feature.ID)
	})
	return nil
}

func runFeatureList(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	featureListStatus, _ := cmd.Flags().GetString("status")

	features, err := loadAllFeatures(projectID)
	if err != nil {
		return err
	}

	if featureListStatus != "" {
		if !models.IsValidStatus(featureListStatus) {
			return fmt.Errorf("invalid status %q (valid: %s)", featureListStatus, strings.Join(models.ValidStatuses, ", "))
		}
		var filtered []models.Feature
		for _, f := range features {
			if f.Status == featureListStatus {
				filtered = append(filtered, f)
			}
		}
		features = filtered
	}

	output(cmd, features, func(w io.Writer) {
		if len(features) == 0 {
			fmt.Fprintln(w, "No features found.")
			return
		}
		headers := []string{"ID", "NAME", "STATUS", "ASSIGNED", "BACKLOG"}
		var rows [][]string
		for _, f := range features {
			bl := ""
			if f.IsBacklog {
				bl = "yes"
			}
			rows = append(rows, []string{f.ID, f.Name, f.Status, f.AssignedTo, bl})
		}
		printTable(w, headers, rows)
	})
	return nil
}

func runFeatureShow(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	feature, err := loadFeature(projectID, args[0])
	if err != nil {
		if os.IsNotExist(err) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "ID:          %s\n", feature.ID)
		fmt.Fprintf(w, "Name:        %s\n", feature.Name)
		fmt.Fprintf(w, "Status:      %s\n", feature.Status)
		if feature.Description != "" {
			fmt.Fprintf(w, "Description: %s\n", feature.Description)
		}
		if feature.AssignedTo != "" {
			fmt.Fprintf(w, "Assigned:    %s\n", feature.AssignedTo)
		}
		if len(feature.Tags) > 0 {
			fmt.Fprintf(w, "Tags:        %s\n", strings.Join(feature.Tags, ", "))
		}
		if len(feature.Notes) > 0 {
			fmt.Fprintf(w, "Notes:\n")
			for _, n := range feature.Notes {
				fmt.Fprintf(w, "  [%s] %s\n", n.Timestamp.Format("2006-01-02 15:04:05"), n.Body)
			}
		}
		fmt.Fprintf(w, "Backlog:     %v\n", feature.IsBacklog)
		fmt.Fprintf(w, "Created:     %s\n", feature.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Updated:     %s\n", feature.UpdatedAt.Format("2006-01-02 15:04:05"))
	})
	return nil
}

func runFeatureUpdate(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	featureUpdateStatus, _ := cmd.Flags().GetString("status")
	featureUpdateDesc, _ := cmd.Flags().GetString("desc")
	featureUpdateAssign, _ := cmd.Flags().GetString("assign")
	featureUpdateTags, _ := cmd.Flags().GetString("tags")
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return err
	}

	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return err
	}
	defer lock.Release()

	feature, err := loadFeature(projectID, args[0])
	if err != nil {
		if os.IsNotExist(err) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	if featureUpdateStatus != "" {
		if !models.IsValidStatus(featureUpdateStatus) {
			return fmt.Errorf("invalid status %q (valid: %s)", featureUpdateStatus, strings.Join(models.ValidStatuses, ", "))
		}
		feature.Status = featureUpdateStatus
	}
	if cmd.Flags().Changed("desc") {
		feature.Description = featureUpdateDesc
	}
	if cmd.Flags().Changed("assign") {
		feature.AssignedTo = featureUpdateAssign
	}
	if cmd.Flags().Changed("tags") {
		feature.Tags = parseTags(featureUpdateTags)
	}
	feature.UpdatedAt = time.Now().UTC()

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.SaveJSON(path, feature); err != nil {
		return fmt.Errorf("save feature: %w", err)
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "Updated feature %q (%s)\n", feature.Name, feature.ID)
	})
	return nil
}

func runFeatureDelete(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return err
	}

	moveTasksTo, _ := cmd.Flags().GetString("move-tasks-to")

	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return err
	}
	defer lock.Release()

	feature, err := loadFeature(projectID, args[0])
	if err != nil {
		if os.IsNotExist(err) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	if feature.IsBacklog {
		return fmt.Errorf("cannot delete the backlog feature")
	}

	tasks, err := loadAllTasks(projectID)
	if err != nil {
		return err
	}

	var belongingTasks []*models.Task
	for i := range tasks {
		if tasks[i].FeatureID == feature.ID {
			belongingTasks = append(belongingTasks, &tasks[i])
		}
	}

	if len(belongingTasks) > 0 {
		if moveTasksTo == "" {
			ids := make([]string, len(belongingTasks))
			for i, t := range belongingTasks {
				ids[i] = t.ID
			}
			return fmt.Errorf("feature %s has %d task(s) (%s); use --move-tasks-to to reassign them first",
				feature.ID, len(belongingTasks), strings.Join(ids, ", "))
		}

		// Validate target feature.
		target, err := loadFeature(projectID, moveTasksTo)
		if err != nil {
			if os.IsNotExist(err) {
				return notFoundErr("feature", moveTasksTo)
			}
			return err
		}
		if target.Status == models.StatusDone {
			return fmt.Errorf("cannot move tasks to feature %s because it is done", target.ID)
		}

		now := time.Now().UTC()
		for _, t := range belongingTasks {
			t.FeatureID = target.ID
			t.UpdatedAt = now
			p := filepath.Join(projectDir, "tasks", t.ID+".json")
			if err := store.SaveJSON(p, t); err != nil {
				return fmt.Errorf("update task %s: %w", t.ID, err)
			}
		}
	}

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.DeleteFile(path); err != nil {
		return fmt.Errorf("delete feature: %w", err)
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "Deleted feature %q (%s)\n", feature.Name, feature.ID)
	})
	return nil
}

// --- helpers ---

func runFeatureNote(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return err
	}

	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return err
	}
	defer lock.Release()

	feature, err := loadFeature(projectID, args[0])
	if err != nil {
		if os.IsNotExist(err) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	feature.Notes = append(feature.Notes, models.Note{
		Timestamp: time.Now().UTC(),
		Body:      args[1],
	})
	feature.UpdatedAt = time.Now().UTC()

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.SaveJSON(path, feature); err != nil {
		return fmt.Errorf("save feature: %w", err)
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "Note added to feature %q (%s)\n", feature.Name, feature.ID)
	})
	return nil
}

func loadAllFeatures(projectID string) ([]models.Feature, error) {
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	featuresDir := filepath.Join(projectDir, "features")
	names, err := store.ListDir(featuresDir)
	if err != nil {
		return nil, err
	}
	var features []models.Feature
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		var f models.Feature
		path := filepath.Join(featuresDir, name)
		if err := store.LoadJSON(path, &f); err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", path, err)
		}
		features = append(features, f)
	}
	return features, nil
}

func loadFeature(projectID, featureID string) (*models.Feature, error) {
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	var f models.Feature
	path := filepath.Join(projectDir, "features", featureID+".json")
	if err := store.LoadJSON(path, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func findBacklogFeature(projectID string) (*models.Feature, error) {
	features, err := loadAllFeatures(projectID)
	if err != nil {
		return nil, err
	}
	for _, f := range features {
		if f.IsBacklog {
			return &f, nil
		}
	}
	return nil, fmt.Errorf("no backlog feature found for project %s", projectID)
}

func parseTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}
