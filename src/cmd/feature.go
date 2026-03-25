package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bira/internal/models"
	"bira/internal/store"

	"github.com/spf13/cobra"
)

var featureCmd = &cobra.Command{
	Use:   "feature",
	Short: "Manage features",
}

// --- add ---

var featureAddDesc, featureAddAssign, featureAddTags string

var featureAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new feature",
	Args:  cobra.ExactArgs(1),
	RunE:  runFeatureAdd,
}

// --- list ---

var featureListStatus string

var featureListCmd = &cobra.Command{
	Use:   "list",
	Short: "List features in the current project",
	RunE:  runFeatureList,
}

// --- show ---

var featureShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show feature details",
	Args:  cobra.ExactArgs(1),
	RunE:  runFeatureShow,
}

// --- update ---

var featureUpdateStatus, featureUpdateDesc, featureUpdateAssign, featureUpdateTags string

var featureUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a feature",
	Args:  cobra.ExactArgs(1),
	RunE:  runFeatureUpdate,
}

// --- delete ---

var featureDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a feature",
	Args:  cobra.ExactArgs(1),
	RunE:  runFeatureDelete,
}

func init() {
	featureAddCmd.Flags().StringVar(&featureAddDesc, "desc", "", "feature description")
	featureAddCmd.Flags().StringVar(&featureAddAssign, "assign", "", "assigned agent/user")
	featureAddCmd.Flags().StringVar(&featureAddTags, "tags", "", "comma-separated tags")

	featureListCmd.Flags().StringVar(&featureListStatus, "status", "", "filter by status")

	featureUpdateCmd.Flags().StringVar(&featureUpdateStatus, "status", "", "new status")
	featureUpdateCmd.Flags().StringVar(&featureUpdateDesc, "desc", "", "new description")
	featureUpdateCmd.Flags().StringVar(&featureUpdateAssign, "assign", "", "new assignee")
	featureUpdateCmd.Flags().StringVar(&featureUpdateTags, "tags", "", "new comma-separated tags")

	featureCmd.AddCommand(featureAddCmd, featureListCmd, featureShowCmd, featureUpdateCmd, featureDeleteCmd)
	rootCmd.AddCommand(featureCmd)
}

// --- implementations ---

func runFeatureAdd(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject()
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

	output(&feature, func() {
		fmt.Printf("Created feature %q (%s)\n", feature.Name, feature.ID)
	})
	return nil
}

func runFeatureList(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject()
	if err != nil {
		return err
	}

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

	output(features, func() {
		if len(features) == 0 {
			fmt.Println("No features found.")
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
		printTable(headers, rows)
	})
	return nil
}

func runFeatureShow(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject()
	if err != nil {
		return err
	}
	feature, err := loadFeature(projectID, args[0])
	if err != nil {
		if os.IsNotExist(err) {
			exitNotFound("feature", args[0])
		}
		return err
	}

	output(feature, func() {
		fmt.Printf("ID:          %s\n", feature.ID)
		fmt.Printf("Name:        %s\n", feature.Name)
		fmt.Printf("Status:      %s\n", feature.Status)
		if feature.Description != "" {
			fmt.Printf("Description: %s\n", feature.Description)
		}
		if feature.AssignedTo != "" {
			fmt.Printf("Assigned:    %s\n", feature.AssignedTo)
		}
		if len(feature.Tags) > 0 {
			fmt.Printf("Tags:        %s\n", strings.Join(feature.Tags, ", "))
		}
		fmt.Printf("Backlog:     %v\n", feature.IsBacklog)
		fmt.Printf("Created:     %s\n", feature.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated:     %s\n", feature.UpdatedAt.Format("2006-01-02 15:04:05"))
	})
	return nil
}

func runFeatureUpdate(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject()
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
			exitNotFound("feature", args[0])
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

	output(feature, func() {
		fmt.Printf("Updated feature %q (%s)\n", feature.Name, feature.ID)
	})
	return nil
}

func runFeatureDelete(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject()
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
			exitNotFound("feature", args[0])
		}
		return err
	}

	if feature.IsBacklog {
		return fmt.Errorf("cannot delete the backlog feature")
	}

	path := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.DeleteFile(path); err != nil {
		return fmt.Errorf("delete feature: %w", err)
	}

	output(feature, func() {
		fmt.Printf("Deleted feature %q (%s)\n", feature.Name, feature.ID)
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
	var features []models.Feature
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		var f models.Feature
		if err := store.LoadJSON(filepath.Join(featuresDir, name), &f); err != nil {
			continue
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
