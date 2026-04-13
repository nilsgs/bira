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

func newIdeaCmd() *cobra.Command {
	ideaCmd := &cobra.Command{
		Use:   "idea",
		Short: "Manage ideas",
	}

	addCmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Capture a new idea (status: inbox)",
		Args:  cobra.ExactArgs(1),
		RunE:  runIdeaAdd,
	}
	addCmd.Flags().String("desc", "", "description")
	addCmd.Flags().String("tags", "", "comma-separated tags")
	addCmd.Flags().String("priority", "", "priority (low|medium|high)")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List ideas",
		RunE:  runIdeaList,
	}
	listCmd.Flags().String("status", "", "filter by status")

	showCmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show idea details",
		Args:  cobra.ExactArgs(1),
		RunE:  runIdeaShow,
	}

	triageCmd := &cobra.Command{
		Use:   "triage <id>",
		Short: "Move idea to triaged status",
		Args:  cobra.ExactArgs(1),
		RunE:  runIdeaTriage,
	}
	triageCmd.Flags().String("priority", "", "priority (low|medium|high)")
	triageCmd.Flags().String("note", "", "optional note")

	rejectCmd := &cobra.Command{
		Use:   "reject <id>",
		Short: "Reject an idea",
		Args:  cobra.ExactArgs(1),
		RunE:  runIdeaReject,
	}
	rejectCmd.Flags().String("note", "", "reason for rejection")

	promoteCmd := &cobra.Command{
		Use:   "promote <id>",
		Short: "Promote an idea to a feature",
		Args:  cobra.ExactArgs(1),
		RunE:  runIdeaPromote,
	}
	promoteCmd.Flags().String("impact", "", "impact level (low|medium|high)")
	promoteCmd.Flags().String("complexity", "", "complexity level (low|medium|high)")

	noteCmd := &cobra.Command{
		Use:   "note <id> <message>",
		Short: "Append a note to an idea",
		Args:  cobra.ExactArgs(2),
		RunE:  runIdeaNote,
	}

	updateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update idea fields",
		Args:  cobra.ExactArgs(1),
		RunE:  runIdeaUpdate,
	}
	updateCmd.Flags().String("title", "", "new title")
	updateCmd.Flags().String("desc", "", "new description")
	updateCmd.Flags().String("tags", "", "new comma-separated tags")
	updateCmd.Flags().String("priority", "", "new priority")

	deleteCmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an idea",
		Args:  cobra.ExactArgs(1),
		RunE:  runIdeaDelete,
	}

	ideaCmd.AddCommand(addCmd, listCmd, showCmd, triageCmd, rejectCmd, promoteCmd, noteCmd, updateCmd, deleteCmd)
	return ideaCmd
}

func runIdeaAdd(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	desc, _ := cmd.Flags().GetString("desc")
	tags, _ := cmd.Flags().GetString("tags")
	priority, _ := cmd.Flags().GetString("priority")

	if priority != "" && !models.IsValidPriority(priority) {
		return fmt.Errorf("invalid priority %q (valid: low, medium, high)", priority)
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
	idea := models.Idea{
		ID:          store.NewID(),
		ProjectID:   projectID,
		Title:       args[0],
		Description: desc,
		Priority:    priority,
		Tags:        parseTags(tags),
		Status:      models.IdeaStatusInbox,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	path := filepath.Join(projectDir, "ideas", idea.ID+".json")
	if err := store.SaveJSON(path, &idea); err != nil {
		return fmt.Errorf("save idea: %w", err)
	}

	output(cmd, &idea, func(w io.Writer) {
		fmt.Fprintf(w, "Created idea %q (%s)\n", idea.Title, idea.ID)
	})
	return nil
}

func runIdeaList(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	statusFilter, _ := cmd.Flags().GetString("status")

	ideas, err := loadAllIdeas(projectID)
	if err != nil {
		return err
	}

	if statusFilter != "" {
		if !models.IsValidIdeaStatus(statusFilter) {
			return fmt.Errorf("invalid status %q (valid: %s)", statusFilter, strings.Join(models.ValidIdeaStatuses, ", "))
		}
		var filtered []models.Idea
		for _, x := range ideas {
			if x.Status == statusFilter {
				filtered = append(filtered, x)
			}
		}
		ideas = filtered
	}

	output(cmd, ideas, func(w io.Writer) {
		if len(ideas) == 0 {
			fmt.Fprintln(w, "No ideas found.")
			return
		}
		headers := []string{"ID", "TITLE", "STATUS", "PRIORITY"}
		var rows [][]string
		for _, x := range ideas {
			rows = append(rows, []string{x.ID, x.Title, x.Status, x.Priority})
		}
		printTable(w, headers, rows)
	})
	return nil
}

func runIdeaShow(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	idea, err := loadIdea(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("idea", args[0])
		}
		return err
	}

	output(cmd, idea, func(w io.Writer) {
		fmt.Fprintf(w, "ID:       %s\n", idea.ID)
		fmt.Fprintf(w, "Title:    %s\n", idea.Title)
		fmt.Fprintf(w, "Status:   %s\n", idea.Status)
		if idea.Priority != "" {
			fmt.Fprintf(w, "Priority: %s\n", idea.Priority)
		}
		if idea.Description != "" {
			fmt.Fprintf(w, "Desc:     %s\n", idea.Description)
		}
		if len(idea.Tags) > 0 {
			fmt.Fprintf(w, "Tags:     %s\n", strings.Join(idea.Tags, ", "))
		}
		if len(idea.Notes) > 0 {
			fmt.Fprintf(w, "Notes:\n")
			for _, n := range idea.Notes {
				fmt.Fprintf(w, "  [%s] %s\n", n.Timestamp.Format("2006-01-02 15:04:05"), n.Body)
			}
		}
		fmt.Fprintf(w, "Created:  %s\n", idea.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(w, "Updated:  %s\n", idea.UpdatedAt.Format("2006-01-02 15:04:05"))
	})
	return nil
}

func runIdeaTriage(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	priority, _ := cmd.Flags().GetString("priority")
	note, _ := cmd.Flags().GetString("note")

	if priority != "" && !models.IsValidPriority(priority) {
		return fmt.Errorf("invalid priority %q (valid: low, medium, high)", priority)
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

	idea, err := loadIdea(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("idea", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	idea.Status = models.IdeaStatusTriaged
	if priority != "" {
		idea.Priority = priority
	}
	if note != "" {
		idea.Notes = append(idea.Notes, models.Note{Timestamp: now, Body: note})
	}
	idea.UpdatedAt = now

	path := filepath.Join(projectDir, "ideas", idea.ID+".json")
	if err := store.SaveJSON(path, idea); err != nil {
		return fmt.Errorf("save idea: %w", err)
	}

	output(cmd, idea, func(w io.Writer) {
		fmt.Fprintf(w, "Triaged idea %q (%s)\n", idea.Title, idea.ID)
	})
	return nil
}

func runIdeaReject(cmd *cobra.Command, args []string) error {
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

	idea, err := loadIdea(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("idea", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	idea.Status = models.IdeaStatusRejected
	if note != "" {
		idea.Notes = append(idea.Notes, models.Note{Timestamp: now, Body: note})
	}
	idea.UpdatedAt = now

	path := filepath.Join(projectDir, "ideas", idea.ID+".json")
	if err := store.SaveJSON(path, idea); err != nil {
		return fmt.Errorf("save idea: %w", err)
	}

	output(cmd, idea, func(w io.Writer) {
		fmt.Fprintf(w, "Rejected idea %q (%s)\n", idea.Title, idea.ID)
	})
	return nil
}

func runIdeaPromote(cmd *cobra.Command, args []string) error {
	projectID, err := resolveProject(cmd)
	if err != nil {
		return err
	}
	impact, _ := cmd.Flags().GetString("impact")
	complexity, _ := cmd.Flags().GetString("complexity")

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

	idea, err := loadIdea(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("idea", args[0])
		}
		return err
	}

	now := time.Now().UTC()

	feature := models.Feature{
		ID:           store.NewID(),
		ProjectID:    projectID,
		Title:        idea.Title,
		Description:  idea.Description,
		Impact:       impact,
		Complexity:   complexity,
		Tags:         idea.Tags,
		Status:       models.FeatureStatusProposed,
		PromotedFrom: idea.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	featurePath := filepath.Join(projectDir, "features", feature.ID+".json")
	if err := store.SaveJSON(featurePath, &feature); err != nil {
		return fmt.Errorf("save feature: %w", err)
	}

	idea.Status = models.IdeaStatusPromoted
	idea.UpdatedAt = now
	ideaPath := filepath.Join(projectDir, "ideas", idea.ID+".json")
	if err := store.SaveJSON(ideaPath, idea); err != nil {
		return fmt.Errorf("update idea: %w", err)
	}

	output(cmd, &feature, func(w io.Writer) {
		fmt.Fprintf(w, "Promoted idea %q to feature %s\n", idea.Title, feature.ID)
	})
	return nil
}

func runIdeaNote(cmd *cobra.Command, args []string) error {
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

	idea, err := loadIdea(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("idea", args[0])
		}
		return err
	}

	now := time.Now().UTC()
	idea.Notes = append(idea.Notes, models.Note{Timestamp: now, Body: args[1]})
	idea.UpdatedAt = now

	path := filepath.Join(projectDir, "ideas", idea.ID+".json")
	if err := store.SaveJSON(path, idea); err != nil {
		return fmt.Errorf("save idea: %w", err)
	}

	output(cmd, idea, func(w io.Writer) {
		fmt.Fprintf(w, "Note added to idea %q (%s)\n", idea.Title, idea.ID)
	})
	return nil
}

func runIdeaUpdate(cmd *cobra.Command, args []string) error {
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

	idea, err := loadIdea(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("idea", args[0])
		}
		return err
	}

	if cmd.Flags().Changed("title") {
		title, _ := cmd.Flags().GetString("title")
		idea.Title = title
	}
	if cmd.Flags().Changed("desc") {
		desc, _ := cmd.Flags().GetString("desc")
		idea.Description = desc
	}
	if cmd.Flags().Changed("tags") {
		tags, _ := cmd.Flags().GetString("tags")
		idea.Tags = parseTags(tags)
	}
	if cmd.Flags().Changed("priority") {
		priority, _ := cmd.Flags().GetString("priority")
		if priority != "" && !models.IsValidPriority(priority) {
			return fmt.Errorf("invalid priority %q (valid: low, medium, high)", priority)
		}
		idea.Priority = priority
	}
	idea.UpdatedAt = time.Now().UTC()

	path := filepath.Join(projectDir, "ideas", idea.ID+".json")
	if err := store.SaveJSON(path, idea); err != nil {
		return fmt.Errorf("save idea: %w", err)
	}

	output(cmd, idea, func(w io.Writer) {
		fmt.Fprintf(w, "Updated idea %q (%s)\n", idea.Title, idea.ID)
	})
	return nil
}

func runIdeaDelete(cmd *cobra.Command, args []string) error {
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

	idea, err := loadIdea(projectID, args[0])
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return notFoundErr("idea", args[0])
		}
		return err
	}

	path := filepath.Join(projectDir, "ideas", idea.ID+".json")
	if err := store.DeleteFile(path); err != nil {
		return fmt.Errorf("delete idea: %w", err)
	}

	output(cmd, idea, func(w io.Writer) {
		fmt.Fprintf(w, "Deleted idea %q (%s)\n", idea.Title, idea.ID)
	})
	return nil
}

// --- helpers ---

func loadAllIdeas(projectID string) ([]models.Idea, error) {
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	ideasDir := filepath.Join(projectDir, "ideas")
	names, err := store.ListDir(ideasDir)
	if err != nil {
		return nil, err
	}
	ideas := make([]models.Idea, 0, len(names))
	for _, name := range names {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		jsonPath := filepath.Join(ideasDir, name)
		var x models.Idea
		if err := store.LoadJSON(jsonPath, &x); err != nil {
			return nil, fmt.Errorf("load %s: %w", jsonPath, err)
		}
		plan, err := store.LoadPlan(jsonPath)
		if err != nil {
			return nil, err
		}
		x.Plan = plan
		ideas = append(ideas, x)
	}
	return ideas, nil
}

func loadIdea(projectID, ideaID string) (*models.Idea, error) {
	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	jsonPath := filepath.Join(projectDir, "ideas", ideaID+".json")
	var x models.Idea
	if err := store.LoadJSON(jsonPath, &x); err != nil {
		return nil, err
	}
	plan, err := store.LoadPlan(jsonPath)
	if err != nil {
		return nil, err
	}
	x.Plan = plan
	return &x, nil
}
