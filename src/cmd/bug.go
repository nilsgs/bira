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

func newBugCmd() *cobra.Command {
	bugCmd := &cobra.Command{
		Use:   "bug",
		Short: "Manage bugs",
	}

	createCmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Report a new bug (status: open)",
		Args:  cobra.ExactArgs(1),
		RunE:  runBugCreate,
	}
	createCmd.Flags().String("desc", "", "description")
	createCmd.Flags().String("criticality", "", "criticality (low|medium|high|critical)")
	createCmd.Flags().String("reported-by", "", "reporter name or identifier")
	createCmd.Flags().String("tags", "", "comma-separated tags")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List bugs",
		RunE:  runBugList,
	}
	listCmd.Flags().String("status", "", "filter by status")

	showCmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show bug details",
		Args:  cobra.ExactArgs(1),
		RunE:  runBugShow,
	}

	triageCmd := &cobra.Command{
		Use:   "triage <id>",
		Short: "Move bug to triaged status",
		Args:  cobra.ExactArgs(1),
		RunE:  runBugTriage,
	}
	triageCmd.Flags().String("criticality", "", "criticality (low|medium|high|critical)")
	triageCmd.Flags().String("note", "", "optional note")

	startCmd := &cobra.Command{
		Use:   "start <id>",
		Short: "Mark bug as in-progress",
		Args:  cobra.ExactArgs(1),
		RunE:  runBugStart,
	}
	startCmd.Flags().String("note", "", "optional note")

	doneCmd := &cobra.Command{
		Use:   "done <id>",
		Short: "Mark bug as fixed",
		Args:  cobra.ExactArgs(1),
		RunE:  runBugDone,
	}
	doneCmd.Flags().String("note", "", "optional note")

	wontFixCmd := &cobra.Command{
		Use:   "wont-fix <id>",
		Short: "Mark bug as wont-fix",
		Args:  cobra.ExactArgs(1),
		RunE:  runBugWontFix,
	}
	wontFixCmd.Flags().String("note", "", "reason")

	noteCmd := &cobra.Command{
		Use:   "note <id> <message>",
		Short: "Append a note to a bug",
		Args:  cobra.ExactArgs(2),
		RunE:  runBugNote,
	}

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update bug fields",
		Args:  cobra.ExactArgs(1),
		RunE:  runBugUpdate,
	}
	updateCmd.Flags().String("title", "", "new title")
	updateCmd.Flags().String("desc", "", "new description")
	updateCmd.Flags().String("criticality", "", "new criticality")
	updateCmd.Flags().String("reported-by", "", "new reporter")
	updateCmd.Flags().String("tags", "", "new comma-separated tags")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a bug",
		Args:  cobra.ExactArgs(1),
		RunE:  runBugDelete,
	}

	bugCmd.AddCommand(createCmd, listCmd, showCmd, triageCmd, startCmd, doneCmd, wontFixCmd, noteCmd, updateCmd, deleteCmd)
	return bugCmd
}

func runBugCreate(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	desc, _ := cmd.Flags().GetString("desc")
	criticality, _ := cmd.Flags().GetString("criticality")
	reportedBy, _ := cmd.Flags().GetString("reported-by")
	tags, _ := cmd.Flags().GetString("tags")

	if criticality != "" && !models.IsValidCriticality(criticality) {
		return fmt.Errorf("invalid criticality %q (valid: low, medium, high, critical)", criticality)
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
	bug := models.Bug{
		ID:          store.NewID(),
		ProjectID:   projectID,
		Title:       args[0],
		Description: desc,
		Criticality: criticality,
		ReportedBy:  reportedBy,
		Tags:        parseTags(tags),
		Status:      models.BugStatusOpen,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	path := filepath.Join(projectDir, "bugs", bug.ID+".json")
	if err := store.SaveJSON(path, &bug); err != nil {
		return fmt.Errorf("save bug: %w", err)
	}

	output(cmd, &bug, func(w io.Writer) {
		fmt.Fprintf(w, "Created bug %q (%s)\n", bug.Title, bug.ID)
	})
	return nil
}

func runBugList(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	statusFilter, _ := cmd.Flags().GetString("status")

	bugs, err := loadAllBugs(projectID)
	if err != nil {
		return err
	}

	if statusFilter != "" {
		if !models.IsValidBugStatus(statusFilter) {
			return fmt.Errorf("invalid status %q (valid: %s)", statusFilter, strings.Join(models.ValidBugStatuses, ", "))
		}
		var filtered []models.Bug
		for _, b := range bugs {
			if b.Status == statusFilter {
				filtered = append(filtered, b)
			}
		}
		bugs = filtered
	}

	output(cmd, bugs, func(w io.Writer) {
		if len(bugs) == 0 {
			fmt.Fprintln(w, "No bugs found.")
			return
		}
		headers := []string{"ID", "TITLE", "STATUS", "CRITICALITY"}
		var rows [][]string
		for _, b := range bugs {
			rows = append(rows, []string{b.ID, b.Title, b.Status, b.Criticality})
		}
		printTable(w, headers, rows)
	})
	return nil
}

func runBugShow(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	bug, err := loadBug(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("bug", args[0])
		}
		return err
	}

	output(cmd, bug, func(w io.Writer) {
		fmt.Fprintf(w, "ID:          %s\n", bug.ID)
		fmt.Fprintf(w, "Title:       %s\n", bug.Title)
		fmt.Fprintf(w, "Status:      %s\n", bug.Status)
		if bug.Criticality != "" {
			fmt.Fprintf(w, "Criticality: %s\n", bug.Criticality)
		}
		if bug.ReportedBy != "" {
			fmt.Fprintf(w, "Reported by: %s\n", bug.ReportedBy)
		}
		if bug.Description != "" {
			fmt.Fprintf(w, "Desc:        %s\n", bug.Description)
		}
		if len(bug.Tags) > 0 {
			fmt.Fprintf(w, "Tags:        %s\n", strings.Join(bug.Tags, ", "))
		}
		if len(bug.Notes) > 0 {
			fmt.Fprintf(w, "Notes:\n")
			for _, n := range bug.Notes {
				fmt.Fprintf(w, "  [%s] %s\n", n.Timestamp.Format("2006-01-02 15:04:05"), n.Body)
			}
		}
		fmt.Fprintf(w, "Created:     %s\n", bug.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Updated:     %s\n", bug.UpdatedAt.Format("2006-01-02 15:04:05"))
	})
	return nil
}

func runBugTriage(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	criticality, _ := cmd.Flags().GetString("criticality")
	note, _ := cmd.Flags().GetString("note")

	if criticality != "" && !models.IsValidCriticality(criticality) {
		return fmt.Errorf("invalid criticality %q (valid: low, medium, high, critical)", criticality)
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

	bug, err := loadBug(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("bug", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	bug.Status = models.BugStatusTriaged
	if criticality != "" {
		bug.Criticality = criticality
	}
	if note != "" {
		bug.Notes = append(bug.Notes, models.Note{Timestamp: now, Body: note})
	}
	bug.UpdatedAt = now

	path := filepath.Join(projectDir, "bugs", bug.ID+".json")
	if err := store.SaveJSON(path, bug); err != nil {
		return fmt.Errorf("save bug: %w", err)
	}

	output(cmd, bug, func(w io.Writer) {
		fmt.Fprintf(w, "Triaged bug %q (%s)\n", bug.Title, bug.ID)
	})
	return nil
}

func runBugStart(cmd *cobra.Command, args []string) error {
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

	bug, err := loadBug(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("bug", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	bug.Status = models.BugStatusInProgress
	if note != "" {
		bug.Notes = append(bug.Notes, models.Note{Timestamp: now, Body: note})
	}
	bug.UpdatedAt = now

	path := filepath.Join(projectDir, "bugs", bug.ID+".json")
	if err := store.SaveJSON(path, bug); err != nil {
		return fmt.Errorf("save bug: %w", err)
	}

	output(cmd, bug, func(w io.Writer) {
		fmt.Fprintf(w, "Started bug %q (%s)\n", bug.Title, bug.ID)
	})
	return nil
}

func runBugDone(cmd *cobra.Command, args []string) error {
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

	bug, err := loadBug(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("bug", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	bug.Status = models.BugStatusFixed
	if note != "" {
		bug.Notes = append(bug.Notes, models.Note{Timestamp: now, Body: note})
	}
	bug.UpdatedAt = now

	path := filepath.Join(projectDir, "bugs", bug.ID+".json")
	if err := store.SaveJSON(path, bug); err != nil {
		return fmt.Errorf("save bug: %w", err)
	}

	output(cmd, bug, func(w io.Writer) {
		fmt.Fprintf(w, "Fixed bug %q (%s)\n", bug.Title, bug.ID)
	})
	return nil
}

func runBugWontFix(cmd *cobra.Command, args []string) error {
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

	bug, err := loadBug(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("bug", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	bug.Status = models.BugStatusWontFix
	if note != "" {
		bug.Notes = append(bug.Notes, models.Note{Timestamp: now, Body: note})
	}
	bug.UpdatedAt = now

	path := filepath.Join(projectDir, "bugs", bug.ID+".json")
	if err := store.SaveJSON(path, bug); err != nil {
		return fmt.Errorf("save bug: %w", err)
	}

	output(cmd, bug, func(w io.Writer) {
		fmt.Fprintf(w, "Marked bug %q as wont-fix (%s)\n", bug.Title, bug.ID)
	})
	return nil
}

func runBugNote(cmd *cobra.Command, args []string) error {
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

	bug, err := loadBug(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("bug", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	bug.Notes = append(bug.Notes, models.Note{Timestamp: now, Body: args[1]})
	bug.UpdatedAt = now

	path := filepath.Join(projectDir, "bugs", bug.ID+".json")
	if err := store.SaveJSON(path, bug); err != nil {
		return fmt.Errorf("save bug: %w", err)
	}

	output(cmd, bug, func(w io.Writer) {
		fmt.Fprintf(w, "Note added to bug %q (%s)\n", bug.Title, bug.ID)
	})
	return nil
}

func runBugUpdate(cmd *cobra.Command, args []string) error {
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

	bug, err := loadBug(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("bug", args[0])
		}
		return err
	}

	if cmd.Flags().Changed("title") {
		title, _ := cmd.Flags().GetString("title")
		bug.Title = title
	}
	if cmd.Flags().Changed("desc") {
		desc, _ := cmd.Flags().GetString("desc")
		bug.Description = desc
	}
	if cmd.Flags().Changed("criticality") {
		criticality, _ := cmd.Flags().GetString("criticality")
		if criticality != "" && !models.IsValidCriticality(criticality) {
			return fmt.Errorf("invalid criticality %q (valid: low, medium, high, critical)", criticality)
		}
		bug.Criticality = criticality
	}
	if cmd.Flags().Changed("reported-by") {
		reportedBy, _ := cmd.Flags().GetString("reported-by")
		bug.ReportedBy = reportedBy
	}
	if cmd.Flags().Changed("tags") {
		tags, _ := cmd.Flags().GetString("tags")
		bug.Tags = parseTags(tags)
	}
	bug.UpdatedAt = time.Now().UTC()

	path := filepath.Join(projectDir, "bugs", bug.ID+".json")
	if err := store.SaveJSON(path, bug); err != nil {
		return fmt.Errorf("save bug: %w", err)
	}

	output(cmd, bug, func(w io.Writer) {
		fmt.Fprintf(w, "Updated bug %q (%s)\n", bug.Title, bug.ID)
	})
	return nil
}

func runBugDelete(cmd *cobra.Command, args []string) error {
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

	bug, err := loadBug(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("bug", args[0])
		}
		return err
	}

	path := filepath.Join(projectDir, "bugs", bug.ID+".json")
	if err := store.DeleteFile(path); err != nil {
		return fmt.Errorf("delete bug: %w", err)
	}

	output(cmd, bug, func(w io.Writer) {
		fmt.Fprintf(w, "Deleted bug %q (%s)\n", bug.Title, bug.ID)
	})
	return nil
}

// --- helpers ---

func loadAllBugs(projectID string) ([]models.Bug, error) {
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	bugsDir := filepath.Join(projectDir, "bugs")
	names, err := store.ListDir(bugsDir)
	if err != nil {
		return nil, err
	}
	bugs := make([]models.Bug, 0, len(names))
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		jsonPath := filepath.Join(bugsDir, name)
		var b models.Bug
		if err := store.LoadJSON(jsonPath, &b); err != nil {
			return nil, fmt.Errorf("load %s: %w", jsonPath, err)
		}
		plan, err := store.LoadPlan(jsonPath)
		if err != nil {
			return nil, err
		}
		b.Plan = plan
		bugs = append(bugs, b)
	}
	return bugs, nil
}

func loadBug(projectID, bugID string) (*models.Bug, error) {
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	jsonPath := filepath.Join(projectDir, "bugs", bugID+".json")
	var b models.Bug
	if err := store.LoadJSON(jsonPath, &b); err != nil {
		return nil, err
	}
	plan, err := store.LoadPlan(jsonPath)
	if err != nil {
		return nil, err
	}
	b.Plan = plan
	return &b, nil
}
