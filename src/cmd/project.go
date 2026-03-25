package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bira/internal/models"
	"bira/internal/store"

	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	RunE:  runProjectList,
}

var projectShowCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show project details",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runProjectShow,
}

var projectDeleteYes bool

var projectDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a project and all its data",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectDelete,
}

func init() {
	projectDeleteCmd.Flags().BoolVar(&projectDeleteYes, "yes", false, "confirm deletion without prompt")
	projectCmd.AddCommand(projectListCmd, projectShowCmd, projectDeleteCmd)
	rootCmd.AddCommand(projectCmd)
}

func runProjectList(cmd *cobra.Command, args []string) error {
	dataDir, err := store.DataDir()
	if err != nil {
		return err
	}
	projectsDir := filepath.Join(dataDir, "projects")
	entries, err := store.ListDir(projectsDir)
	if err != nil {
		return err
	}

	var projects []models.Project
	for _, name := range entries {
		metaPath := filepath.Join(projectsDir, name, "meta.json")
		var p models.Project
		if err := store.LoadJSON(metaPath, &p); err != nil {
			continue
		}
		projects = append(projects, p)
	}

	output(projects, func() {
		if len(projects) == 0 {
			fmt.Println("No projects found. Run 'bira init' in a repository.")
			return
		}
		headers := []string{"ID", "NAME", "REPO PATH"}
		var rows [][]string
		for _, p := range projects {
			rows = append(rows, []string{p.ID, p.Name, p.RepoPath})
		}
		printTable(headers, rows)
	})
	return nil
}

func runProjectShow(cmd *cobra.Command, args []string) error {
	var id string
	if len(args) > 0 {
		id = args[0]
	} else {
		var err error
		id, err = resolveProject()
		if err != nil {
			return err
		}
	}

	project, err := loadProject(id)
	if err != nil {
		if os.IsNotExist(err) {
			exitNotFound("project", id)
		}
		return err
	}

	output(project, func() {
		fmt.Printf("ID:          %s\n", project.ID)
		fmt.Printf("Name:        %s\n", project.Name)
		if project.Description != "" {
			fmt.Printf("Description: %s\n", project.Description)
		}
		fmt.Printf("Repo:        %s\n", project.RepoPath)
		if len(project.Tags) > 0 {
			fmt.Printf("Tags:        %s\n", strings.Join(project.Tags, ", "))
		}
		fmt.Printf("Created:     %s\n", project.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated:     %s\n", project.UpdatedAt.Format("2006-01-02 15:04:05"))
	})
	return nil
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	if !projectDeleteYes {
		return fmt.Errorf("project delete is destructive; pass --yes to confirm")
	}

	project, err := loadProject(id)
	if err != nil {
		if os.IsNotExist(err) {
			exitNotFound("project", id)
		}
		return err
	}

	projectDir, err := store.ProjectDir(id)
	if err != nil {
		return err
	}
	if err := store.DeleteDir(projectDir); err != nil {
		return fmt.Errorf("delete project directory: %w", err)
	}

	output(project, func() {
		fmt.Printf("Deleted project %q (%s)\n", project.Name, project.ID)
	})
	return nil
}

// --- helpers ---

func resolveProject() (string, error) {
	return resolveProjectID()
}

func resolveProjectID() (string, error) {
	return resolveProjectIDFromFlag(projectFlag)
}

func resolveProjectIDFromFlag(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	cfg, _, err := loadConfigFromDir(cwd)
	if err != nil {
		return "", err
	}
	return cfg.ProjectID, nil
}

func loadConfigFromDir(dir string) (*configData, string, error) {
	cfgPath := dir
	for {
		path := filepath.Join(cfgPath, ".bira")
		if store.Exists(path) {
			var cfg configData
			if err := store.LoadJSON(path, &cfg); err != nil {
				return nil, "", err
			}
			return &cfg, cfgPath, nil
		}
		parent := filepath.Dir(cfgPath)
		if parent == cfgPath {
			break
		}
		cfgPath = parent
	}
	return nil, "", fmt.Errorf("no .bira config found (run 'bira init' first)")
}

type configData struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
}

func loadProject(id string) (*models.Project, error) {
	projectDir, err := store.ProjectDir(id)
	if err != nil {
		return nil, err
	}
	var p models.Project
	if err := store.LoadJSON(filepath.Join(projectDir, "meta.json"), &p); err != nil {
		return nil, err
	}
	return &p, nil
}
