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

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
}

// --- add ---

var taskAddFeature, taskAddDesc, taskAddAssign, taskAddTags, taskAddDependsOn string
var taskAddCriteria []string
var taskAddFiles string

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
var taskUpdateCriteria []string
var taskUpdateFiles string

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

// --- note ---

var taskNoteCmd = &cobra.Command{
	Use:   "note <id> <message>",
	Short: "Append a note to a task",
	Args:  cobra.ExactArgs(2),
	RunE:  runTaskNote,
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
	taskAddCmd.Flags().StringArrayVar(&taskAddCriteria, "criteria", nil, "acceptance criterion (repeatable)")
	taskAddCmd.Flags().StringVar(&taskAddFiles, "files", "", "comma-separated file references")

	taskListCmd.Flags().StringVar(&taskListFeature, "feature", "", "filter by feature ID")
	taskListCmd.Flags().StringVar(&taskListStatus, "status", "", "filter by status")

	taskUpdateCmd.Flags().StringVar(&taskUpdateStatus, "status", "", "new status")
	taskUpdateCmd.Flags().StringVar(&taskUpdateDesc, "desc", "", "new description")
	taskUpdateCmd.Flags().StringVar(&taskUpdateAssign, "assign", "", "new assignee")
	taskUpdateCmd.Flags().StringVar(&taskUpdateTags, "tags", "", "new comma-separated tags")
	taskUpdateCmd.Flags().StringVar(&taskUpdateDependsOn, "depends-on", "", "new comma-separated dependency IDs")
	taskUpdateCmd.Flags().StringArrayVar(&taskUpdateCriteria, "criteria", nil, "acceptance criteria, replaces existing (repeatable)")
	taskUpdateCmd.Flags().StringVar(&taskUpdateFiles, "files", "", "new comma-separated file references")

	taskCmd.AddCommand(taskAddCmd, taskListCmd, taskShowCmd, taskUpdateCmd, taskDeleteCmd, taskNoteCmd, taskDoneCmd)
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
				return notFoundErr("feature", featureID)
			}
			return err
		}
	}

	now := time.Now().UTC()
	task := models.Task{
		ID:                 store.NewID(),
		ProjectID:          projectID,
		FeatureID:          featureID,
		Title:              args[0],
		Description:        taskAddDesc,
		Status:             models.StatusTodo,
		AssignedTo:         taskAddAssign,
		Tags:               parseTags(taskAddTags),
		DependsOn:          parseTags(taskAddDependsOn),
		AcceptanceCriteria: taskAddCriteria,
		Files:              parseTags(taskAddFiles),
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.SaveJSON(path, &task); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	output(cmd, &task, func(w io.Writer) {
		fmt.Fprintf(w, "Created task %q (%s) in feature %s\n", task.Title, task.ID, task.FeatureID)
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

	output(cmd, tasks, func(w io.Writer) {
		if len(tasks) == 0 {
			fmt.Fprintln(w, "No tasks found.")
			return
		}
		headers := []string{"ID", "TITLE", "STATUS", "FEATURE", "ASSIGNED"}
		var rows [][]string
		for _, t := range tasks {
			rows = append(rows, []string{t.ID, t.Title, t.Status, t.FeatureID, t.AssignedTo})
		}
		printTable(w, headers, rows)
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
			return notFoundErr("task", args[0])
		}
		return err
	}

	output(cmd, task, func(w io.Writer) {
		fmt.Fprintf(w, "ID:          %s\n", task.ID)
		fmt.Fprintf(w, "Title:       %s\n", task.Title)
		fmt.Fprintf(w, "Status:      %s\n", task.Status)
		fmt.Fprintf(w, "Feature:     %s\n", task.FeatureID)
		if task.Description != "" {
			fmt.Fprintf(w, "Description: %s\n", task.Description)
		}
		if task.AssignedTo != "" {
			fmt.Fprintf(w, "Assigned:    %s\n", task.AssignedTo)
		}
		if len(task.Tags) > 0 {
			fmt.Fprintf(w, "Tags:        %s\n", strings.Join(task.Tags, ", "))
		}
		if len(task.DependsOn) > 0 {
			fmt.Fprintf(w, "Depends on:  %s\n", strings.Join(task.DependsOn, ", "))
		}
		if len(task.AcceptanceCriteria) > 0 {
			fmt.Fprintf(w, "Criteria:\n")
			for _, c := range task.AcceptanceCriteria {
				fmt.Fprintf(w, "  - %s\n", c)
			}
		}
		if len(task.Files) > 0 {
			fmt.Fprintf(w, "Files:       %s\n", strings.Join(task.Files, ", "))
		}
		if len(task.Notes) > 0 {
			fmt.Fprintf(w, "Notes:\n")
			for _, n := range task.Notes {
				fmt.Fprintf(w, "  [%s] %s\n", n.Timestamp.Format("2006-01-02 15:04:05"), n.Body)
			}
		}
		fmt.Fprintf(w, "Created:     %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Updated:     %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))
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
			return notFoundErr("task", args[0])
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
	if cmd.Flags().Changed("criteria") {
		task.AcceptanceCriteria = taskUpdateCriteria
	}
	if cmd.Flags().Changed("files") {
		task.Files = parseTags(taskUpdateFiles)
	}
	task.UpdatedAt = time.Now().UTC()

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.SaveJSON(path, task); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	output(cmd, task, func(w io.Writer) {
		fmt.Fprintf(w, "Updated task %q (%s)\n", task.Title, task.ID)
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
			return notFoundErr("task", args[0])
		}
		return err
	}

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.DeleteFile(path); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	output(cmd, task, func(w io.Writer) {
		fmt.Fprintf(w, "Deleted task %q (%s)\n", task.Title, task.ID)
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
			return notFoundErr("task", args[0])
		}
		return err
	}

	task.Status = models.StatusDone
	task.UpdatedAt = time.Now().UTC()

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.SaveJSON(path, task); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	output(cmd, task, func(w io.Writer) {
		fmt.Fprintf(w, "Marked task %q (%s) as done\n", task.Title, task.ID)
	})
	return nil
}

// --- helpers ---

func runTaskNote(cmd *cobra.Command, args []string) error {
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
			return notFoundErr("task", args[0])
		}
		return err
	}

	task.Notes = append(task.Notes, models.Note{
		Timestamp: time.Now().UTC(),
		Body:      args[1],
	})
	task.UpdatedAt = time.Now().UTC()

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.SaveJSON(path, task); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	output(cmd, task, func(w io.Writer) {
		fmt.Fprintf(w, "Note added to task %q (%s)\n", task.Title, task.ID)
	})
	return nil
}

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
