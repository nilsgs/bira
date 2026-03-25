package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bira/internal/config"
	"bira/internal/models"
	"bira/internal/store"

	"github.com/spf13/cobra"
)

var initName string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a bira project in the current directory",
	Long:  "Creates a new project, writes a .bira config file, and sets up the project in ~/.bira.",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().StringVar(&initName, "name", "", "project name (default: current directory name)")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if config.ExistsIn(cwd) {
		return fmt.Errorf("project already initialized in this directory (found %s)", config.FileName)
	}

	name := initName
	if name == "" {
		name = filepath.Base(cwd)
	}

	now := time.Now().UTC()
	projectID := store.NewID()

	project := models.Project{
		ID:        projectID,
		Name:      name,
		RepoPath:  cwd,
		CreatedAt: now,
		UpdatedAt: now,
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

	metaPath := filepath.Join(projectDir, "meta.json")
	if err := store.SaveJSON(metaPath, &project); err != nil {
		return fmt.Errorf("save project: %w", err)
	}

	// Create backlog feature
	backlogID := store.NewID()
	backlog := models.Feature{
		ID:        backlogID,
		ProjectID: projectID,
		Name:      "backlog",
		Status:    models.StatusTodo,
		IsBacklog: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	featurePath := filepath.Join(projectDir, "features", backlogID+".json")
	if err := store.SaveJSON(featurePath, &backlog); err != nil {
		return fmt.Errorf("save backlog feature: %w", err)
	}

	cfg := &config.Config{
		ProjectID:   projectID,
		ProjectName: name,
	}
	if err := config.Save(cwd, cfg); err != nil {
		return fmt.Errorf("write .bira config: %w", err)
	}

	output(&project, func() {
		fmt.Printf("Initialized project %q (%s)\n", name, projectID)
		fmt.Printf("Config written to %s\n", filepath.Join(cwd, config.FileName))
	})

	return nil
}
