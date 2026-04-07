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

func newSessionCmd() *cobra.Command {
	sessionCmd := &cobra.Command{
		Use:   "session",
		Short: "Manage agent sessions",
	}

	// --- start ---

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new agent session",
		RunE:  runSessionStart,
	}
	startCmd.Flags().String("label", "", "optional human-readable session label (must be unique)")

	// --- list ---

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List active sessions for the current project",
		RunE:  runSessionList,
	}

	// --- show ---

	showCmd := &cobra.Command{
		Use:   "show <session-id>",
		Short: "Show session details and its claimed tasks",
		Args:  cobra.ExactArgs(1),
		RunE:  runSessionShow,
	}

	// --- end ---

	endCmd := &cobra.Command{
		Use:   "end <session-id>",
		Short: "End a session, releasing all its claimed tasks",
		Args:  cobra.ExactArgs(1),
		RunE:  runSessionEnd,
	}
	endCmd.Flags().String("release-status", "todo", "status to set on released tasks (todo|blocked)")

	sessionCmd.AddCommand(startCmd, listCmd, showCmd, endCmd)
	return sessionCmd
}

// --- implementations ---

func runSessionStart(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	label, _ := cmd.Flags().GetString("label")

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return err
	}

	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return err
	}
	defer lock.Release()

	// Validate label uniqueness if supplied.
	if label != "" {
		existing, err := loadAllSessions(projectID)
		if err != nil {
			return err
		}
		for _, s := range existing {
			if s.Label == label {
				return fmt.Errorf("session label %q is already in use by session %s", label, s.ID)
			}
		}
	}

	session := models.Session{
		ID:        store.NewID(),
		ProjectID: projectID,
		Label:     label,
		StartedAt: time.Now().UTC(),
	}

	path := filepath.Join(projectDir, "sessions", session.ID+".json")
	if err := store.SaveJSON(path, &session); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	output(cmd, &session, func(w io.Writer) {
		label := session.Label
		if label == "" {
			label = "(no label)"
		}
		fmt.Fprintf(w, "Session started: %s (%s)\n", session.ID, label)
		fmt.Fprintf(w, "Export with: export BIRA_SESSION=%s\n", session.ID)
	})
	return nil
}

func runSessionList(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}

	sessions, err := loadAllSessions(projectID)
	if err != nil {
		return err
	}

	tasks, err := loadAllTasks(projectID)
	if err != nil {
		return err
	}

	type sessionListEntry struct {
		models.Session
		ClaimedTaskCount int `json:"claimed_task_count"`
	}

	entries := make([]sessionListEntry, 0, len(sessions))
	for _, s := range sessions {
		count := 0
		for _, t := range tasks {
			if t.ClaimedBy == s.ID {
				count++
			}
		}
		entries = append(entries, sessionListEntry{Session: s, ClaimedTaskCount: count})
	}

	output(cmd, entries, func(w io.Writer) {
		if len(entries) == 0 {
			fmt.Fprintln(w, "No active sessions.")
			return
		}
		headers := []string{"SESSION", "LABEL", "STARTED", "CLAIMED"}
		var rows [][]string
		for _, e := range entries {
			rows = append(rows, []string{
				e.ID,
				e.Label,
				e.StartedAt.Format("2006-01-02 15:04:05"),
				fmt.Sprintf("%d", e.ClaimedTaskCount),
			})
		}
		printTable(w, headers, rows)
	})
	return nil
}

func runSessionShow(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	sessionID := args[0]

	session, err := loadSession(projectID, sessionID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("session", sessionID)
		}
		return err
	}

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

	var claimed []taskView
	for _, t := range tasks {
		if t.ClaimedBy == session.ID {
			r := computeReadiness(t, tasksByID, featuresByID, timeoutMinutes)
			claimed = append(claimed, taskView{Task: t, TaskReadiness: r})
		}
	}

	type sessionShowOutput struct {
		models.Session
		ClaimedTasks []taskView `json:"claimed_tasks"`
	}

	out := sessionShowOutput{Session: *session, ClaimedTasks: claimed}
	if out.ClaimedTasks == nil {
		out.ClaimedTasks = []taskView{}
	}

	output(cmd, out, func(w io.Writer) {
		label := session.Label
		if label == "" {
			label = "(no label)"
		}
		fmt.Fprintf(w, "Session:   %s (%s)\n", session.ID, label)
		fmt.Fprintf(w, "Started:   %s\n", session.StartedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Claimed:   %d task(s)\n", len(claimed))
		for _, t := range claimed {
			fmt.Fprintf(w, "  - %s: %s [%s]\n", t.ID, t.Title, t.Status)
		}
	})
	return nil
}

func runSessionEnd(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	sessionID := args[0]
	releaseStatus, _ := cmd.Flags().GetString("release-status")

	if releaseStatus != models.StatusTodo && releaseStatus != models.StatusBlocked {
		return fmt.Errorf("invalid release-status %q (valid: todo, blocked)", releaseStatus)
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

	_, err = loadSession(projectID, sessionID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("session", sessionID)
		}
		return err
	}

	tasks, err := loadAllTasks(projectID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	var released []string
	for i := range tasks {
		t := &tasks[i]
		if t.ClaimedBy != sessionID {
			continue
		}
		t.ClaimedBy = ""
		t.ClaimedAt = nil
		t.Status = releaseStatus
		t.Notes = append(t.Notes, models.Note{
			Timestamp: now,
			Body:      fmt.Sprintf("Released by session end (%s)", sessionID),
		})
		t.UpdatedAt = now
		path := filepath.Join(projectDir, "tasks", t.ID+".json")
		if err := store.SaveJSON(path, t); err != nil {
			return fmt.Errorf("save task %s: %w", t.ID, err)
		}
		released = append(released, t.ID)
	}

	// Delete session file.
	sessionPath := filepath.Join(projectDir, "sessions", sessionID+".json")
	if err := store.DeleteFile(sessionPath); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	type endOutput struct {
		ReleasedTasks []string `json:"released_tasks"`
	}
	if released == nil {
		released = []string{}
	}
	out := endOutput{ReleasedTasks: released}

	output(cmd, out, func(w io.Writer) {
		fmt.Fprintf(w, "Session %s ended. Released %d task(s).\n", sessionID, len(released))
		for _, id := range released {
			fmt.Fprintf(w, "  - %s\n", id)
		}
	})
	return nil
}

// --- helpers ---

func loadAllSessions(projectID string) ([]models.Session, error) {
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	sessionsDir := filepath.Join(projectDir, "sessions")
	names, err := store.ListDir(sessionsDir)
	if err != nil {
		return nil, err
	}
	sessions := make([]models.Session, 0, len(names))
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		var s models.Session
		path := filepath.Join(sessionsDir, name)
		if err := store.LoadJSON(path, &s); err != nil {
			return nil, fmt.Errorf("load %s: %w", path, err)
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func loadSession(projectID, sessionID string) (*models.Session, error) {
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	var s models.Session
	path := filepath.Join(projectDir, "sessions", sessionID+".json")
	if err := store.LoadJSON(path, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
