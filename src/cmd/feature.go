package cmd

import (
	"errors"
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

	addCmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a new feature (status: proposed)",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureAdd,
	}
	addCmd.Flags().String("desc", "", "description")
	addCmd.Flags().String("impact", "", "impact level (low|medium|high)")
	addCmd.Flags().String("complexity", "", "complexity level (low|medium|high)")
	addCmd.Flags().String("tags", "", "comma-separated tags")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List features",
		RunE:  runFeatureList,
	}
	listCmd.Flags().String("status", "", "filter by status")

	showCmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show feature details",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureShow,
	}

	triageCmd := &cobra.Command{
		Use:   "triage <id>",
		Short: "Move feature to triaged status",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureTriage,
	}
	triageCmd.Flags().String("impact", "", "impact level (low|medium|high)")
	triageCmd.Flags().String("complexity", "", "complexity level (low|medium|high)")
	triageCmd.Flags().String("note", "", "optional note")

	startCmd := &cobra.Command{
		Use:   "start <id>",
		Short: "Mark feature as in-progress",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureStart,
	}
	startCmd.Flags().String("note", "", "optional note")

	doneCmd := &cobra.Command{
		Use:   "done <id>",
		Short: "Mark feature as done",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureDone,
	}
	doneCmd.Flags().String("note", "", "optional note")

	rejectCmd := &cobra.Command{
		Use:   "reject <id>",
		Short: "Reject a feature",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureReject,
	}
	rejectCmd.Flags().String("note", "", "reason for rejection")

	noteCmd := &cobra.Command{
		Use:   "note <id> <message>",
		Short: "Append a note to a feature",
		Args:  cobra.ExactArgs(2),
		RunE:  runFeatureNote,
	}

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update feature fields",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureUpdate,
	}
	updateCmd.Flags().String("title", "", "new title")
	updateCmd.Flags().String("desc", "", "new description")
	updateCmd.Flags().String("impact", "", "new impact")
	updateCmd.Flags().String("complexity", "", "new complexity")
	updateCmd.Flags().String("tags", "", "new comma-separated tags")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a feature",
		Args:  cobra.ExactArgs(1),
		RunE:  runFeatureDelete,
	}

	featureCmd.AddCommand(addCmd, listCmd, showCmd, triageCmd, startCmd, doneCmd, rejectCmd, noteCmd, updateCmd, deleteCmd)
	return featureCmd
}

func runFeatureAdd(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	desc, _ := cmd.Flags().GetString("desc")
	impact, _ := cmd.Flags().GetString("impact")
	complexity, _ := cmd.Flags().GetString("complexity")
	tags, _ := cmd.Flags().GetString("tags")

	if impact != "" && !models.IsValidLevel(impact) {
		return fmt.Errorf("invalid impact %q (valid: low, medium, high)", impact)
	}
	if complexity != "" && !models.IsValidLevel(complexity) {
		return fmt.Errorf("invalid complexity %q (valid: low, medium, high)", complexity)
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

	now := time.Now().UTC()
	feature := models.Feature{
		ID:          store.NewID(),
		ProjectID:   projectID,
		Title:       args[0],
		Description: desc,
		Impact:      impact,
		Complexity:  complexity,
		Tags:        parseTags(tags),
		Status:      models.FeatureStatusProposed,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.SaveJSON(path, &feature); err != nil {
		return fmt.Errorf("save feature: %w", err)
	}

	output(cmd, &feature, func(w io.Writer) {
		fmt.Fprintf(w, "Created feature %q (%s)\n", feature.Title, feature.ID)
	})
	return nil
}

func runFeatureList(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	statusFilter, _ := cmd.Flags().GetString("status")

	features, err := loadAllFeatures(projectID)
	if err != nil {
		return err
	}

	if statusFilter != "" {
		if !models.IsValidFeatureStatus(statusFilter) {
			return fmt.Errorf("invalid status %q (valid: %s)", statusFilter, strings.Join(models.ValidFeatureStatuses, ", "))
		}
		var filtered []models.Feature
		for _, f := range features {
			if f.Status == statusFilter {
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
		headers := []string{"ID", "TITLE", "STATUS", "IMPACT", "COMPLEXITY"}
		var rows [][]string
		for _, f := range features {
			rows = append(rows, []string{f.ID, f.Title, f.Status, f.Impact, f.Complexity})
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
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "ID:           %s\n", feature.ID)
		fmt.Fprintf(w, "Title:        %s\n", feature.Title)
		fmt.Fprintf(w, "Status:       %s\n", feature.Status)
		if feature.Impact != "" {
			fmt.Fprintf(w, "Impact:       %s\n", feature.Impact)
		}
		if feature.Complexity != "" {
			fmt.Fprintf(w, "Complexity:   %s\n", feature.Complexity)
		}
		if feature.Description != "" {
			fmt.Fprintf(w, "Desc:         %s\n", feature.Description)
		}
		if feature.PromotedFrom != "" {
			fmt.Fprintf(w, "PromotedFrom: %s\n", feature.PromotedFrom)
		}
		if len(feature.Tags) > 0 {
			fmt.Fprintf(w, "Tags:         %s\n", strings.Join(feature.Tags, ", "))
		}
		if len(feature.Notes) > 0 {
			fmt.Fprintf(w, "Notes:\n")
			for _, n := range feature.Notes {
				fmt.Fprintf(w, "  [%s] %s\n", n.Timestamp.Format("2006-01-02 15:04:05"), n.Body)
			}
		}
		fmt.Fprintf(w, "Created:      %s\n", feature.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Updated:      %s\n", feature.UpdatedAt.Format("2006-01-02 15:04:05"))
	})
	return nil
}

func runFeatureTriage(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	impact, _ := cmd.Flags().GetString("impact")
	complexity, _ := cmd.Flags().GetString("complexity")
	note, _ := cmd.Flags().GetString("note")

	if impact != "" && !models.IsValidLevel(impact) {
		return fmt.Errorf("invalid impact %q (valid: low, medium, high)", impact)
	}
	if complexity != "" && !models.IsValidLevel(complexity) {
		return fmt.Errorf("invalid complexity %q (valid: low, medium, high)", complexity)
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
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	feature.Status = models.FeatureStatusTriaged
	if impact != "" {
		feature.Impact = impact
	}
	if complexity != "" {
		feature.Complexity = complexity
	}
	if note != "" {
		feature.Notes = append(feature.Notes, models.Note{Timestamp: now, Body: note})
	}
	feature.UpdatedAt = now

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.SaveJSON(path, feature); err != nil {
		return fmt.Errorf("save feature: %w", err)
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "Triaged feature %q (%s)\n", feature.Title, feature.ID)
	})
	return nil
}

func runFeatureStart(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	note, _ := cmd.Flags().GetString("note")

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
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	feature.Status = models.FeatureStatusInProgress
	if note != "" {
		feature.Notes = append(feature.Notes, models.Note{Timestamp: now, Body: note})
	}
	feature.UpdatedAt = now

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.SaveJSON(path, feature); err != nil {
		return fmt.Errorf("save feature: %w", err)
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "Started feature %q (%s)\n", feature.Title, feature.ID)
	})
	return nil
}

func runFeatureDone(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	note, _ := cmd.Flags().GetString("note")

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
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	feature.Status = models.FeatureStatusDone
	if note != "" {
		feature.Notes = append(feature.Notes, models.Note{Timestamp: now, Body: note})
	}
	feature.UpdatedAt = now

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.SaveJSON(path, feature); err != nil {
		return fmt.Errorf("save feature: %w", err)
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "Completed feature %q (%s)\n", feature.Title, feature.ID)
	})
	return nil
}

func runFeatureReject(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	note, _ := cmd.Flags().GetString("note")

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
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	feature.Status = models.FeatureStatusRejected
	if note != "" {
		feature.Notes = append(feature.Notes, models.Note{Timestamp: now, Body: note})
	}
	feature.UpdatedAt = now

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.SaveJSON(path, feature); err != nil {
		return fmt.Errorf("save feature: %w", err)
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "Rejected feature %q (%s)\n", feature.Title, feature.ID)
	})
	return nil
}

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
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	feature.Notes = append(feature.Notes, models.Note{Timestamp: now, Body: args[1]})
	feature.UpdatedAt = now

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.SaveJSON(path, feature); err != nil {
		return fmt.Errorf("save feature: %w", err)
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "Note added to feature %q (%s)\n", feature.Title, feature.ID)
	})
	return nil
}

func runFeatureUpdate(cmd *cobra.Command, args []string) error {
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
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	if cmd.Flags().Changed("title") {
		title, _ := cmd.Flags().GetString("title")
		feature.Title = title
	}
	if cmd.Flags().Changed("desc") {
		desc, _ := cmd.Flags().GetString("desc")
		feature.Description = desc
	}
	if cmd.Flags().Changed("impact") {
		impact, _ := cmd.Flags().GetString("impact")
		if impact != "" && !models.IsValidLevel(impact) {
			return fmt.Errorf("invalid impact %q (valid: low, medium, high)", impact)
		}
		feature.Impact = impact
	}
	if cmd.Flags().Changed("complexity") {
		complexity, _ := cmd.Flags().GetString("complexity")
		if complexity != "" && !models.IsValidLevel(complexity) {
			return fmt.Errorf("invalid complexity %q (valid: low, medium, high)", complexity)
		}
		feature.Complexity = complexity
	}
	if cmd.Flags().Changed("tags") {
		tags, _ := cmd.Flags().GetString("tags")
		feature.Tags = parseTags(tags)
	}
	feature.UpdatedAt = time.Now().UTC()

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.SaveJSON(path, feature); err != nil {
		return fmt.Errorf("save feature: %w", err)
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "Updated feature %q (%s)\n", feature.Title, feature.ID)
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
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return err
	}
	defer lock.Release()

	feature, err := loadFeature(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("feature", args[0])
		}
		return err
	}

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.DeleteFile(path); err != nil {
		return fmt.Errorf("delete feature: %w", err)
	}

	output(cmd, feature, func(w io.Writer) {
		fmt.Fprintf(w, "Deleted feature %q (%s)\n", feature.Title, feature.ID)
	})
	return nil
}

// --- helpers ---

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
	features := make([]models.Feature, 0, len(names))
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		jsonPath := filepath.Join(featuresDir, name)
		var f models.Feature
		if err := store.LoadJSON(jsonPath, &f); err != nil {
			return nil, fmt.Errorf("load %s: %w", jsonPath, err)
		}
		plan, err := store.LoadPlan(jsonPath)
		if err != nil {
			return nil, err
		}
		f.Plan = plan
		features = append(features, f)
	}
	return features, nil
}

func loadFeature(projectID, featureID string) (*models.Feature, error) {
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	jsonPath := filepath.Join(projectDir, "features", featureID+".json")
	var f models.Feature
	if err := store.LoadJSON(jsonPath, &f); err != nil {
		return nil, err
	}
	plan, err := store.LoadPlan(jsonPath)
	if err != nil {
		return nil, err
	}
	f.Plan = plan
	return &f, nil
}
