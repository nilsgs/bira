package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bira/internal/models"
	"bira/internal/store"

	"github.com/spf13/cobra"
)

func newTaskCmd() *cobra.Command {
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
	}

	// --- add ---

	addCmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Create a new task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskAdd,
	}
	addCmd.Flags().String("feature", "", "parent feature ID (default: backlog)")
	addCmd.Flags().String("desc", "", "task description")
	addCmd.Flags().String("assign", "", "assigned agent/user")
	addCmd.Flags().String("tags", "", "comma-separated tags")
	addCmd.Flags().String("depends-on", "", "comma-separated task IDs this depends on")
	addCmd.Flags().StringArray("criteria", nil, "acceptance criterion (repeatable)")
	addCmd.Flags().String("files", "", "comma-separated file references")

	// --- list ---

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks in the current project",
		RunE:  runTaskList,
	}
	listCmd.Flags().String("feature", "", "filter by feature ID")
	listCmd.Flags().String("status", "", "filter by status")
	listCmd.Flags().Bool("ready", false, "filter to tasks ready for agent pickup")
	listCmd.Flags().String("assigned", "", "filter by assigned_to value")
	listCmd.Flags().Bool("unassigned", false, "filter to tasks with no assigned_to")
	listCmd.Flags().String("claimed-by", "", "filter to tasks claimed by session ID or label")
	listCmd.Flags().Bool("unclaimed", false, "filter to tasks with no active claim")

	// --- show ---

	showCmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show task details",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskShow,
	}

	// --- update ---

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskUpdate,
	}
	updateCmd.Flags().String("status", "", "new status")
	updateCmd.Flags().String("desc", "", "new description")
	updateCmd.Flags().String("assign", "", "new assignee")
	updateCmd.Flags().String("tags", "", "new comma-separated tags")
	updateCmd.Flags().String("depends-on", "", "new comma-separated dependency IDs")
	updateCmd.Flags().StringArray("criteria", nil, "acceptance criteria, replaces existing (repeatable)")
	updateCmd.Flags().String("files", "", "new comma-separated file references")

	// --- delete ---

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a task",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskDelete,
	}

	// --- note ---

	noteCmd := &cobra.Command{
		Use:   "note <id> <message>",
		Short: "Append a note to a task",
		Args:  cobra.ExactArgs(2),
		RunE:  runTaskNote,
	}

	// --- done ---

	doneCmd := &cobra.Command{
		Use:   "done <id>",
		Short: "Mark a task as done",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskDone,
	}
	doneCmd.Flags().String("session", "", "session that claims this task (or $BIRA_SESSION)")

	// --- claim ---

	claimCmd := &cobra.Command{
		Use:   "claim <id>",
		Short: "Claim a ready task for an agent session",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskClaim,
	}
	claimCmd.Flags().String("session", "", "session ID (or $BIRA_SESSION)")
	claimCmd.Flags().Bool("force", false, "claim even if assigned_to doesn't match session label")

	// --- release ---

	releaseCmd := &cobra.Command{
		Use:   "release <id>",
		Short: "Release a claimed task back to the pool",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskRelease,
	}
	releaseCmd.Flags().String("session", "", "session ID (or $BIRA_SESSION)")
	releaseCmd.Flags().String("status", "todo", "status to set after release (todo|blocked)")
	releaseCmd.Flags().String("note", "", "optional note to append on release")
	releaseCmd.Flags().Bool("force", false, "release even if not claimed by this session")

	taskCmd.AddCommand(addCmd, listCmd, showCmd, updateCmd, deleteCmd, noteCmd, doneCmd, claimCmd, releaseCmd)
	return taskCmd
}

// --- implementations ---

func runTaskAdd(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	taskAddFeature, _ := cmd.Flags().GetString("feature")
	taskAddDesc, _ := cmd.Flags().GetString("desc")
	taskAddAssign, _ := cmd.Flags().GetString("assign")
	taskAddTags, _ := cmd.Flags().GetString("tags")
	taskAddDependsOn, _ := cmd.Flags().GetString("depends-on")
	taskAddCriteria, _ := cmd.Flags().GetStringArray("criteria")
	taskAddFiles, _ := cmd.Flags().GetString("files")
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
	newID := store.NewID()
	deps := parseTags(taskAddDependsOn)

	// Validate dependencies before saving.
	if len(deps) > 0 {
		allTasks, err := loadAllTasks(projectID)
		if err != nil {
			return err
		}
		// Include the new task in the map so cycle detection works (it has no deps yet).
		tasksByID := tasksToMap(allTasks)
		if err := validateDependsOn(newID, deps, tasksByID); err != nil {
			return err
		}
	}

	task := models.Task{
		ID:                 newID,
		ProjectID:          projectID,
		FeatureID:          featureID,
		Title:              args[0],
		Description:        taskAddDesc,
		Status:             models.StatusTodo,
		AssignedTo:         taskAddAssign,
		Tags:               parseTags(taskAddTags),
		DependsOn:          deps,
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
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	filterFeature, _ := cmd.Flags().GetString("feature")
	filterStatus, _ := cmd.Flags().GetString("status")
	filterReady, _ := cmd.Flags().GetBool("ready")
	filterAssigned, _ := cmd.Flags().GetString("assigned")
	filterUnassigned, _ := cmd.Flags().GetBool("unassigned")
	filterClaimedBy, _ := cmd.Flags().GetString("claimed-by")
	filterUnclaimed, _ := cmd.Flags().GetBool("unclaimed")

	tasks, err := loadAllTasks(projectID)
	if err != nil {
		return err
	}
	features, err := loadAllFeatures(projectID)
	if err != nil {
		return err
	}

	timeoutMinutes := loadClaimTimeout(cmd)
	tasksByID := tasksToMap(tasks)
	featuresByID := featuresToMap(features)

	type taskView struct {
		models.Task
		TaskReadiness
	}

	var views []taskView
	for _, t := range tasks {
		r := computeReadiness(t, tasksByID, featuresByID, timeoutMinutes)
		views = append(views, taskView{Task: t, TaskReadiness: r})
	}

	// Apply filters
	if filterFeature != "" {
		var filtered []taskView
		for _, v := range views {
			if v.FeatureID == filterFeature {
				filtered = append(filtered, v)
			}
		}
		views = filtered
	}
	if filterStatus != "" {
		if !models.IsValidStatus(filterStatus) {
			return fmt.Errorf("invalid status %q (valid: %s)", filterStatus, strings.Join(models.ValidStatuses, ", "))
		}
		var filtered []taskView
		for _, v := range views {
			if v.Status == filterStatus {
				filtered = append(filtered, v)
			}
		}
		views = filtered
	}
	if filterReady {
		var filtered []taskView
		for _, v := range views {
			if v.ReadyForAgent {
				filtered = append(filtered, v)
			}
		}
		views = filtered
	}
	if filterAssigned != "" {
		var filtered []taskView
		for _, v := range views {
			if v.AssignedTo == filterAssigned {
				filtered = append(filtered, v)
			}
		}
		views = filtered
	}
	if filterUnassigned {
		var filtered []taskView
		for _, v := range views {
			if v.AssignedTo == "" {
				filtered = append(filtered, v)
			}
		}
		views = filtered
	}
	if filterClaimedBy != "" {
		var filtered []taskView
		for _, v := range views {
			if v.ClaimedBy == filterClaimedBy {
				filtered = append(filtered, v)
			}
		}
		views = filtered
	}
	if filterUnclaimed {
		var filtered []taskView
		for _, v := range views {
			if v.ClaimedBy == "" {
				filtered = append(filtered, v)
			}
		}
		views = filtered
	}

	// Sort by created_at ASC, id ASC
	sort.Slice(views, func(i, j int) bool {
		a, b := views[i].Task, views[j].Task
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.Before(b.CreatedAt)
		}
		return a.ID < b.ID
	})

	output(cmd, views, func(w io.Writer) {
		if len(views) == 0 {
			fmt.Fprintln(w, "No tasks found.")
			return
		}
		headers := []string{"ID", "TITLE", "STATUS", "READY", "CLAIMED BY", "FEATURE", "ASSIGNED"}
		var rows [][]string
		for _, v := range views {
			ready := "no"
			if v.ReadyForAgent {
				ready = "yes"
			}
			rows = append(rows, []string{
				v.ID, v.Title, v.Status, ready, v.ClaimedBy, v.FeatureID, v.AssignedTo,
			})
		}
		printTable(w, headers, rows)
	})
	return nil
}

func runTaskShow(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
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

	allTasks, err := loadAllTasks(projectID)
	if err != nil {
		return err
	}
	allFeatures, err := loadAllFeatures(projectID)
	if err != nil {
		return err
	}

	timeoutMinutes := loadClaimTimeout(cmd)
	tasksByID := tasksToMap(allTasks)
	featuresByID := featuresToMap(allFeatures)
	readiness := computeReadiness(*task, tasksByID, featuresByID, timeoutMinutes)

	type featureSummary struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	var featureSum *featureSummary
	if f, ok := featuresByID[task.FeatureID]; ok {
		featureSum = &featureSummary{ID: f.ID, Name: f.Name, Status: f.Status}
	}

	type depView struct {
		ID     string `json:"id"`
		Exists bool   `json:"exists"`
		Status string `json:"status,omitempty"`
		Title  string `json:"title,omitempty"`
	}
	depViews := make([]depView, 0, len(task.DependsOn))
	for _, depID := range task.DependsOn {
		if dep, ok := tasksByID[depID]; ok {
			depViews = append(depViews, depView{ID: depID, Exists: true, Status: dep.Status, Title: dep.Title})
		} else {
			depViews = append(depViews, depView{ID: depID, Exists: false})
		}
	}

	type taskShowOutput struct {
		*models.Task
		TaskReadiness
		Feature      *featureSummary `json:"feature,omitempty"`
		Dependencies []depView       `json:"dependencies"`
	}
	out := taskShowOutput{
		Task:          task,
		TaskReadiness: readiness,
		Feature:       featureSum,
		Dependencies:  depViews,
	}

	output(cmd, out, func(w io.Writer) {
		fmt.Fprintf(w, "ID:            %s\n", task.ID)
		fmt.Fprintf(w, "Title:         %s\n", task.Title)
		fmt.Fprintf(w, "Status:        %s\n", task.Status)
		if featureSum != nil {
			fmt.Fprintf(w, "Feature:       %s (%s) [%s]\n", featureSum.ID, featureSum.Name, featureSum.Status)
		} else {
			fmt.Fprintf(w, "Feature:       %s\n", task.FeatureID)
		}
		if task.Description != "" {
			fmt.Fprintf(w, "Description:   %s\n", task.Description)
		}
		if task.AssignedTo != "" {
			fmt.Fprintf(w, "Assigned:      %s\n", task.AssignedTo)
		}
		if task.ClaimedBy != "" {
			fmt.Fprintf(w, "Claimed by:    %s\n", task.ClaimedBy)
			if task.ClaimedAt != nil {
				fmt.Fprintf(w, "Claimed at:    %s\n", task.ClaimedAt.Format("2006-01-02 15:04:05"))
			}
			if readiness.StaleClaim {
				fmt.Fprintf(w, "Stale claim:   yes\n")
			}
		}
		ready := "no"
		if readiness.ReadyForAgent {
			ready = "yes"
		}
		fmt.Fprintf(w, "Ready:         %s\n", ready)
		if len(readiness.MissingFields) > 0 {
			fmt.Fprintf(w, "Missing:       %s\n", strings.Join(readiness.MissingFields, ", "))
		}
		if len(readiness.BlockedReasons) > 0 {
			fmt.Fprintf(w, "Blocked:       %s\n", strings.Join(readiness.BlockedReasons, ", "))
		}
		if len(task.Tags) > 0 {
			fmt.Fprintf(w, "Tags:          %s\n", strings.Join(task.Tags, ", "))
		}
		if len(depViews) > 0 {
			fmt.Fprintf(w, "Depends on:\n")
			for _, d := range depViews {
				if d.Exists {
					fmt.Fprintf(w, "  - %s: %s [%s]\n", d.ID, d.Title, d.Status)
				} else {
					fmt.Fprintf(w, "  - %s (not found)\n", d.ID)
				}
			}
		}
		if len(task.AcceptanceCriteria) > 0 {
			fmt.Fprintf(w, "Criteria:\n")
			for _, c := range task.AcceptanceCriteria {
				fmt.Fprintf(w, "  - %s\n", c)
			}
		}
		if len(task.Files) > 0 {
			fmt.Fprintf(w, "Files:         %s\n", strings.Join(task.Files, ", "))
		}
		if len(task.Notes) > 0 {
			fmt.Fprintf(w, "Notes:\n")
			for _, n := range task.Notes {
				fmt.Fprintf(w, "  [%s] %s\n", n.Timestamp.Format("2006-01-02 15:04:05"), n.Body)
			}
		}
		fmt.Fprintf(w, "Created:       %s\n", task.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Updated:       %s\n", task.UpdatedAt.Format("2006-01-02 15:04:05"))
	})
	return nil
}


func runTaskUpdate(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	taskUpdateStatus, _ := cmd.Flags().GetString("status")
	taskUpdateDesc, _ := cmd.Flags().GetString("desc")
	taskUpdateAssign, _ := cmd.Flags().GetString("assign")
	taskUpdateTags, _ := cmd.Flags().GetString("tags")
	taskUpdateDependsOn, _ := cmd.Flags().GetString("depends-on")
	taskUpdateCriteria, _ := cmd.Flags().GetStringArray("criteria")
	taskUpdateFiles, _ := cmd.Flags().GetString("files")
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

	// Block status changes on claimed tasks.
	if cmd.Flags().Changed("status") && task.ClaimedBy != "" {
		return fmt.Errorf("task %s is claimed by session %s; use 'bira task release' or 'bira task done' instead", task.ID, task.ClaimedBy)
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
		newDeps := parseTags(taskUpdateDependsOn)
		allTasks, err := loadAllTasks(projectID)
		if err != nil {
			return err
		}
		tasksByID := tasksToMap(allTasks)
		if err := validateDependsOn(task.ID, newDeps, tasksByID); err != nil {
			return err
		}
		task.DependsOn = newDeps
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

	task, err := loadTask(projectID, args[0])
	if err != nil {
		if os.IsNotExist(err) {
			return notFoundErr("task", args[0])
		}
		return err
	}

	// Block delete if any non-done task depends on this one.
	allTasks, err := loadAllTasks(projectID)
	if err != nil {
		return err
	}
	var blockers []string
	for _, t := range allTasks {
		if t.ID == task.ID || t.Status == models.StatusDone {
			continue
		}
		for _, dep := range t.DependsOn {
			if dep == task.ID {
				blockers = append(blockers, t.ID)
				break
			}
		}
	}
	if len(blockers) > 0 {
		return fmt.Errorf("cannot delete task %s: referenced by task(s) %s", task.ID, strings.Join(blockers, ", "))
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

	task, err := loadTask(projectID, args[0])
	if err != nil {
		if os.IsNotExist(err) {
			return notFoundErr("task", args[0])
		}
		return err
	}

	sessionFlag, _ := cmd.Flags().GetString("session")
	sessionID := resolveSession(sessionFlag)
	if sessionID != "" && task.ClaimedBy != "" && task.ClaimedBy != sessionID {
		return fmt.Errorf("task %s is claimed by session %s; use that session or release first", task.ID, task.ClaimedBy)
	}

	now := time.Now().UTC()
	task.Status = models.StatusDone
	task.ClaimedBy = ""
	task.ClaimedAt = nil
	task.UpdatedAt = now

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.SaveJSON(path, task); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	output(cmd, task, func(w io.Writer) {
		fmt.Fprintf(w, "Marked task %q (%s) as done\n", task.Title, task.ID)
	})
	return nil
}

func runTaskClaim(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return err
	}

	sessionFlag, _ := cmd.Flags().GetString("session")
	sessionID := resolveSession(sessionFlag)
	if sessionID == "" {
		return fmt.Errorf("session ID is required (use --session or set $BIRA_SESSION)")
	}
	force, _ := cmd.Flags().GetBool("force")

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

	session, err := loadSession(projectID, sessionID)
	if err != nil {
		if os.IsNotExist(err) {
			return notFoundErr("session", sessionID)
		}
		return err
	}

	allTasks, err := loadAllTasks(projectID)
	if err != nil {
		return err
	}
	allFeatures, err := loadAllFeatures(projectID)
	if err != nil {
		return err
	}

	timeoutMinutes := loadClaimTimeout(cmd)
	tasksByID := tasksToMap(allTasks)
	featuresByID := featuresToMap(allFeatures)
	readiness := computeReadiness(*task, tasksByID, featuresByID, timeoutMinutes)

	if !readiness.ReadyForAgent {
		return fmt.Errorf("task %s is not ready for agent (missing: %v, blocked: %v)",
			task.ID, readiness.MissingFields, readiness.BlockedReasons)
	}

	// Check if already claimed by another (non-stale) session.
	if task.ClaimedBy != "" && !readiness.StaleClaim {
		return fmt.Errorf("task %s is already claimed by session %s", task.ID, task.ClaimedBy)
	}

	// Enforce assigned_to unless --force.
	if !force && task.AssignedTo != "" && task.AssignedTo != session.Label && task.AssignedTo != session.ID {
		return fmt.Errorf("task %s is assigned to %q; session label is %q (use --force to override)",
			task.ID, task.AssignedTo, session.Label)
	}

	now := time.Now().UTC()
	task.ClaimedBy = session.ID
	task.ClaimedAt = &now
	task.Status = models.StatusInProgress
	task.UpdatedAt = now

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.SaveJSON(path, task); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	type claimOutput struct {
		*models.Task
		TaskReadiness
	}
	out := claimOutput{Task: task, TaskReadiness: computeReadiness(*task, tasksByID, featuresByID, timeoutMinutes)}

	output(cmd, out, func(w io.Writer) {
		label := session.Label
		if label == "" {
			label = session.ID
		}
		fmt.Fprintf(w, "Claimed task %q (%s) by %s (%s)\n", task.Title, task.ID, label, session.ID)
	})
	return nil
}

func runTaskRelease(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return err
	}

	sessionFlag, _ := cmd.Flags().GetString("session")
	sessionID := resolveSession(sessionFlag)
	if sessionID == "" {
		return fmt.Errorf("session ID is required (use --session or set $BIRA_SESSION)")
	}
	force, _ := cmd.Flags().GetBool("force")
	releaseStatus, _ := cmd.Flags().GetString("status")
	note, _ := cmd.Flags().GetString("note")

	if releaseStatus == models.StatusDone {
		return fmt.Errorf("cannot release a task to status %q; use 'bira task done' instead", releaseStatus)
	}
	if releaseStatus != models.StatusTodo && releaseStatus != models.StatusBlocked {
		return fmt.Errorf("invalid status %q for release (valid: todo, blocked)", releaseStatus)
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

	session, err := loadSession(projectID, sessionID)
	if err != nil {
		if os.IsNotExist(err) {
			return notFoundErr("session", sessionID)
		}
		return err
	}

	if task.ClaimedBy != sessionID && !force {
		return fmt.Errorf("task %s is not claimed by session %s (use --force to override)", task.ID, sessionID)
	}

	now := time.Now().UTC()
	task.ClaimedBy = ""
	task.ClaimedAt = nil
	task.Status = releaseStatus
	task.UpdatedAt = now

	if note != "" {
		task.Notes = append(task.Notes, models.Note{
			Timestamp: now,
			Body:      note,
		})
	}

	path := filepath.Join(projectDir, "tasks", task.ID+".json")
	if err := store.SaveJSON(path, task); err != nil {
		return fmt.Errorf("save task: %w", err)
	}

	output(cmd, task, func(w io.Writer) {
		label := session.Label
		if label == "" {
			label = session.ID
		}
		fmt.Fprintf(w, "Released task %q (%s) by %s (%s) [status: %s]\n",
			task.Title, task.ID, label, session.ID, task.Status)
	})
	return nil
}


// --- helpers ---

func runTaskNote(cmd *cobra.Command, args []string) error {
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
		path := filepath.Join(tasksDir, name)
		if err := store.LoadJSON(path, &t); err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", path, err)
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
