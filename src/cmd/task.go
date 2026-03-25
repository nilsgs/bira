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

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
}

// --- add ---

var taskAddFeature, taskAddDesc, taskAddAssign, taskAddTags, taskAddDependsOn string

var taskAddCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Create a new task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskAdd,
}

// --- list ---

var taskListFeature, taskListStatus string

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks in the current project",
	RunE:  runTaskList,
}

// --- show ---

var taskShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show task details",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskShow,
}

// --- update ---

var taskUpdateStatus, taskUpdateDesc, taskUpdateAssign, taskUpdateTags, taskUpdateDependsOn string

var taskUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskUpdate,
}

// --- delete ---

var taskDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskDelete,
}

// --- done ---

var taskDoneCmd = &cobra.Command{
	Use:   "done <id>",
	Short: "Mark a task as done",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskDone,
}

func init() {
	taskAddCmd.Flags().StringVar(&taskAddFeature, "feature", "", "parent feature ID (default: backlog)")
	taskAddCmd.Flags().StringVar(&taskAddDesc, "desc", "", "task description")
	taskAddCmd.Flags().StringVar(&taskAddAssign, "assign", "", "assigned agent/user")
	taskAddCmd.Flags().StringVar(&taskAddTags, "tags", "", "comma-separated tags")
	taskAddCmd.Flags().StringVar(&taskAddDependsOn, "depends-on", "", "comma-separated task IDs this depends on")

	taskListCmd.Flags().StringVar(&taskListFeature, "feature", "", "filter by feature ID")
	taskListCmd.Flags().StringVar(&taskListStatus, "status", "", "filter by status")

	taskUpdateCmd.Flags().StringVar(&taskUpdateStatus, "status", "", "new status")
	taskUpdateCmd.Flags().StringVar(&taskUpdateDesc, "desc", "", "new description")
	taskUpdateCmd.Flags().StringVar(&taskUpdateAssign, "assign", "", "new assignee")
	taskUpdateCmd.Flags().StringVar(&taskUpdateTags, "tags", "", "new comma-separated tags")
	taskUpdateCmd.Flags().StringVar(&taskUpdateDependsOn, "depends-on", "", "new comma-separated dependency IDs")

	taskCmd.AddCommand(taskAddCmd, taskListCmd, taskShowCmd, taskUpdateCmd, taskDeleteCmd, taskDoneCmd)
	rootCmd.AddCommand(taskCmd)
}

// --- implementations ---

func runTaskAdd(cmd *cobra.Command, args []string) error {
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

	featureID := taskAddFeature
	if featureID == "" {
		backlog, err := findBacklogFeature(projectID)
		if err != nil {
			return err
		}
		featureID = backlog.ID
	} else {
		// Verify feature exists
		if _, err := loadFeature(projectID, featureID); err != nil {
			if os.IsNotExist(err) {
				exitNotFound("feature", featureID)
			}
			return err
		}
	}

	now := time.Now().UTC()
	task := models.Task{
		ID:          store.NewID(),
		ProjectID:   projectID,
		FeatureID:   featureID,
		Title:       args[0],
		Description: taskAddDesc,
		Status:      models.StatusTodo,
		AssignedTo:  taskAddAssign,
		Tags:        parseTags(taskAddTags),
		DependsOn:   parseTags(taskAddDependsOn),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.SaveJSON(path, &task); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	output(&task, func() {
		fmt.Printf("Created task %q (%s) in feature %s\n", task.Title, task.ID, task.FeatureID)
	})
	return nil
}

func runTaskList(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject()
	if err != nil {
		return err
	}

	tasks, err := loadAllTasks(projectID)
	if err != nil {
		return err
	}

	// Apply filters
	if taskListFeature != "" {
		var filtered []models.Task
		for _, t := range tasks {
			if t.FeatureID == taskListFeature {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	if taskListStatus != "" {
		if !models.IsValidStatus(taskListStatus) {
			return fmt.Errorf("invalid status %q (valid: %s)", taskListStatus, strings.Join(models.ValidStatuses, ", "))
		}
		var filtered []models.Task
		for _, t := range tasks {
			if t.Status == taskListStatus {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	output(tasks, func() {
		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return
		}
		headers := []string{"ID", "TITLE", "STATUS", "FEATURE", "ASSIGNED"}
		var rows [][]string
		for _, t := range tasks {
			rows = append(rows, []string{t.ID, t.Title, t.Status, t.FeatureID, t.AssignedTo})
		}
		printTable(headers, rows)
	})
	return nil
}

func runTaskShow(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject()
	if err != nil {
		return err
	}
	task, err := loadTask(projectID, args[0])
	if err != nil {
		if os.IsNotExist(err) {
			exitNotFound("task", args[0])
		}
		return err
	}

	output(task, func() {
		fmt.Printf("ID:          %s\n", task.ID)
		fmt.Printf("Title:       %s\n", task.Title)
		fmt.Printf("Status:      %s\n", task.Status)
		fmt.Printf("Feature:     %s\n", task.FeatureID)
		if task.Description != "" {
			fmt.Printf("Description: %s\n", task.Description)
		}
		if task.AssignedTo != "" {
			fmt.Printf("Assigned:    %s\n", task.AssignedTo)
		}
		if len(task.Tags) > 0 {
			fmt.Printf("Tags:        %s\n", strings.Join(task.Tags, ", "))
		}
		if len(task.DependsOn) > 0 {
			fmt.Printf("Depends on:  %s\n", strings.Join(task.DependsOn, ", "))
		}
		fmt.Printf("Created:     %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Updated:     %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))
	})
	return nil
}

func runTaskUpdate(cmd *cobra.Command, args []string) error {
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

	task, err := loadTask(projectID, args[0])
	if err != nil {
		if os.IsNotExist(err) {
			exitNotFound("task", args[0])
		}
		return err
	}

	if taskUpdateStatus != "" {
		if !models.IsValidStatus(taskUpdateStatus) {
			return fmt.Errorf("invalid status %q (valid: %s)", taskUpdateStatus, strings.Join(models.ValidStatuses, ", "))
		}
		task.Status = taskUpdateStatus
	}
	if cmd.Flags().Changed("desc") {
		task.Description = taskUpdateDesc
	}
	if cmd.Flags().Changed("assign") {
		task.AssignedTo = taskUpdateAssign
	}
	if cmd.Flags().Changed("tags") {
		task.Tags = parseTags(taskUpdateTags)
	}
	if cmd.Flags().Changed("depends-on") {
		task.DependsOn = parseTags(taskUpdateDependsOn)
	}
	task.UpdatedAt = time.Now().UTC()

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.SaveJSON(path, task); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	output(task, func() {
		fmt.Printf("Updated task %q (%s)\n", task.Title, task.ID)
	})
	return nil
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
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

	task, err := loadTask(projectID, args[0])
	if err != nil {
		if os.IsNotExist(err) {
			exitNotFound("task", args[0])
		}
		return err
	}

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.DeleteFile(path); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	output(task, func() {
		fmt.Printf("Deleted task %q (%s)\n", task.Title, task.ID)
	})
	return nil
}

func runTaskDone(cmd *cobra.Command, args []string) error {
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

	task, err := loadTask(projectID, args[0])
	if err != nil {
		if os.IsNotExist(err) {
			exitNotFound("task", args[0])
		}
		return err
	}

	task.Status = models.StatusDone
	task.UpdatedAt = time.Now().UTC()

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.SaveJSON(path, task); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	output(task, func() {
		fmt.Printf("Marked task %q (%s) as done\n", task.Title, task.ID)
	})
	return nil
}

// --- helpers ---

func loadAllTasks(projectID string) ([]models.Task, error) {
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	tasksDir := filepath.Join(projectDir, "tasks")
	names, err := store.ListDir(tasksDir)
	if err != nil {
		return nil, err
	}
	var tasks []models.Task
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		var t models.Task
		if err := store.LoadJSON(filepath.Join(tasksDir, name), &t); err != nil {
			continue
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func loadTask(projectID, taskID string) (*models.Task, error) {
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	var t models.Task
	path := filepath.Join(projectDir, "tasks", taskID+".json")
	if err := store.LoadJSON(path, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
