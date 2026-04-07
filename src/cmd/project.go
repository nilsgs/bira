package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"bira/internal/models"
	"bira/internal/store"

	"github.com/spf13/cobra"
)

func newProjectCmd() *cobra.Command {
	projectCmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE:  runProjectList,
	}

	showCmd := &cobra.Command{
		Use:   "show [id]",
		Short: "Show project details",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runProjectShow,
	}

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a project and all its data",
		Args:  cobra.ExactArgs(1),
		RunE:  runProjectDelete,
	}
	deleteCmd.Flags().Bool("yes", false, "confirm deletion without prompt")

	projectCmd.AddCommand(listCmd, showCmd, deleteCmd)
	return projectCmd
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

	output(cmd, projects, func(w io.Writer) {
		if len(projects) == 0 {
			fmt.Fprintln(w, "No projects found. Run 'bira init' in a repository.")
			return
		}
		headers := []string{"ID", "NAME", "REPO PATH"}
		var rows [][]string
		for _, p := range projects {
			rows = append(rows, []string{p.ID, p.Name, p.RepoPath})
		}
		printTable(w, headers, rows)
	})
	return nil
}

func runProjectShow(cmd *cobra.Command, args []string) error {
	var id string
	if len(args) > 0 {
		id = args[0]
	} else {
		var err error
		id, err = resolveProject(cmd)
		if err != nil {
			return err
		}
	}

	project, err := loadProject(id)
	if err != nil {
		if os.IsNotExist(err) {
			return notFoundErr("project", id)
		}
		return err
	}

	output(cmd, project, func(w io.Writer) {
		fmt.Fprintf(w, "ID:          %s\n", project.ID)
		fmt.Fprintf(w, "Name:        %s\n", project.Name)
		if project.Description != "" {
			fmt.Fprintf(w, "Description: %s\n", project.Description)
		}
		fmt.Fprintf(w, "Repo:        %s\n", project.RepoPath)
		if len(project.Tags) > 0 {
			fmt.Fprintf(w, "Tags:        %s\n", strings.Join(project.Tags, ", "))
		}
		fmt.Fprintf(w, "Created:     %s\n", project.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Updated:     %s\n", project.UpdatedAt.Format("2006-01-02 15:04:05"))
	})
	return nil
}

func runProjectDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		return fmt.Errorf("project delete is destructive; pass --yes to confirm")
	}

	project, err := loadProject(id)
	if err != nil {
		if os.IsNotExist(err) {
			return notFoundErr("project", id)
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

	output(cmd, project, func(w io.Writer) {
		fmt.Fprintf(w, "Deleted project %q (%s)\n", project.Name, project.ID)
	})
	return nil
}

// --- helpers ---

// resolveProject reads the --project persistent flag from cmd and resolves it
// to a project ID, falling back to .bira walk-up discovery if not set.
func resolveProject(cmd *cobra.Command) (string, error) {
	flag, _ := cmd.Root().PersistentFlags().GetString("project")
	return resolveProjectIDFromFlag(flag)
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
	ProjectID            string `json:"project_id"`
	ProjectName          string `json:"project_name"`
	ClaimTimeoutMinutes  int    `json:"claim_timeout_minutes,omitempty"`
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

// loadClaimTimeout returns the claim timeout minutes from the .bira config in the
// current working directory, falling back to defaultClaimTimeout (1440).
func loadClaimTimeout(_ *cobra.Command) int {
	cwd, err := os.Getwd()
	if err != nil {
		return defaultClaimTimeout
	}
	cfg, _, err := loadConfigFromDir(cwd)
	if err != nil {
		return defaultClaimTimeout
	}
	if cfg.ClaimTimeoutMinutes > 0 {
		return cfg.ClaimTimeoutMinutes
	}
	return defaultClaimTimeout
}

const defaultClaimTimeout = 1440

