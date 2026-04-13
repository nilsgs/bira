package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bira/internal/models"
	"bira/internal/store"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start an MCP server (stdio transport)",
		Long:  "Starts a Model Context Protocol server over stdio, exposing bira tools to AI agents.",
		RunE:  runMCP,
	}
}

func runMCP(cmd *cobra.Command, args []string) error {
	s := mcpserver.NewMCPServer(
		"bira",
		version+"+"+commit,
		mcpserver.WithToolCapabilities(true),
	)

	registerMCPTools(s)

	return mcpserver.ServeStdio(s)
}

func registerMCPTools(s *mcpserver.MCPServer) {
	// --- context ---

	s.AddTool(mcp.NewTool("bira_context",
		mcp.WithDescription("Get a snapshot of a bira project: counts of ideas, bugs, and features."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithBoolean("full", mcp.Description("Include per-status breakdown")),
	), mcpContext)

	// --- ideas ---

	s.AddTool(mcp.NewTool("bira_idea_add",
		mcp.WithDescription("Capture a new idea in a project."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Idea title"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Optional description")),
		mcp.WithString("priority", mcp.Description("Priority: low, medium, high")),
		mcp.WithString("tags", mcp.Description("Comma-separated tags")),
	), mcpIdeaAdd)

	s.AddTool(mcp.NewTool("bira_idea_list",
		mcp.WithDescription("List ideas in a project."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("status", mcp.Description("Filter by status: inbox, triaged, promoted, rejected")),
	), mcpIdeaList)

	s.AddTool(mcp.NewTool("bira_idea_show",
		mcp.WithDescription("Show details of a specific idea."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("idea_id", mcp.Description("Idea ID"), mcp.Required()),
	), mcpIdeaShow)

	s.AddTool(mcp.NewTool("bira_idea_triage",
		mcp.WithDescription("Move an idea to triaged status."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("idea_id", mcp.Description("Idea ID"), mcp.Required()),
		mcp.WithString("priority", mcp.Description("Priority: low, medium, high")),
		mcp.WithString("note", mcp.Description("Optional note")),
	), mcpIdeaTriage)

	s.AddTool(mcp.NewTool("bira_idea_promote",
		mcp.WithDescription("Promote an idea to a feature."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("idea_id", mcp.Description("Idea ID"), mcp.Required()),
		mcp.WithString("impact", mcp.Description("Impact level: low, medium, high")),
		mcp.WithString("complexity", mcp.Description("Complexity level: low, medium, high")),
	), mcpIdeaPromote)

	s.AddTool(mcp.NewTool("bira_idea_reject",
		mcp.WithDescription("Reject an idea."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("idea_id", mcp.Description("Idea ID"), mcp.Required()),
		mcp.WithString("note", mcp.Description("Reason for rejection")),
	), mcpIdeaReject)

	// --- bugs ---

	s.AddTool(mcp.NewTool("bira_bug_create",
		mcp.WithDescription("Report a new bug."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Bug title"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Optional description")),
		mcp.WithString("criticality", mcp.Description("Criticality: low, medium, high, critical")),
		mcp.WithString("reported_by", mcp.Description("Reporter name or identifier")),
		mcp.WithString("tags", mcp.Description("Comma-separated tags")),
	), mcpBugCreate)

	s.AddTool(mcp.NewTool("bira_bug_list",
		mcp.WithDescription("List bugs in a project."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("status", mcp.Description("Filter by status: open, triaged, in-progress, fixed, wont-fix")),
	), mcpBugList)

	s.AddTool(mcp.NewTool("bira_bug_show",
		mcp.WithDescription("Show details of a specific bug."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("bug_id", mcp.Description("Bug ID"), mcp.Required()),
	), mcpBugShow)

	s.AddTool(mcp.NewTool("bira_bug_triage",
		mcp.WithDescription("Move a bug to triaged status."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("bug_id", mcp.Description("Bug ID"), mcp.Required()),
		mcp.WithString("criticality", mcp.Description("Criticality: low, medium, high, critical")),
		mcp.WithString("note", mcp.Description("Optional note")),
	), mcpBugTriage)

	s.AddTool(mcp.NewTool("bira_bug_start",
		mcp.WithDescription("Mark a bug as in-progress."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("bug_id", mcp.Description("Bug ID"), mcp.Required()),
		mcp.WithString("note", mcp.Description("Optional note")),
	), mcpBugStart)

	s.AddTool(mcp.NewTool("bira_bug_done",
		mcp.WithDescription("Mark a bug as fixed."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("bug_id", mcp.Description("Bug ID"), mcp.Required()),
		mcp.WithString("note", mcp.Description("Optional note")),
	), mcpBugDone)

	s.AddTool(mcp.NewTool("bira_bug_wont_fix",
		mcp.WithDescription("Mark a bug as wont-fix."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("bug_id", mcp.Description("Bug ID"), mcp.Required()),
		mcp.WithString("note", mcp.Description("Reason")),
	), mcpBugWontFix)

	// --- features ---

	s.AddTool(mcp.NewTool("bira_feature_add",
		mcp.WithDescription("Add a new feature to a project."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Feature title"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Optional description")),
		mcp.WithString("impact", mcp.Description("Impact level: low, medium, high")),
		mcp.WithString("complexity", mcp.Description("Complexity level: low, medium, high")),
		mcp.WithString("tags", mcp.Description("Comma-separated tags")),
	), mcpFeatureAdd)

	s.AddTool(mcp.NewTool("bira_feature_list",
		mcp.WithDescription("List features in a project."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("status", mcp.Description("Filter by status: proposed, triaged, in-progress, done, rejected")),
	), mcpFeatureList)

	s.AddTool(mcp.NewTool("bira_feature_show",
		mcp.WithDescription("Show details of a specific feature."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("feature_id", mcp.Description("Feature ID"), mcp.Required()),
	), mcpFeatureShow)

	s.AddTool(mcp.NewTool("bira_feature_triage",
		mcp.WithDescription("Move a feature to triaged status."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("feature_id", mcp.Description("Feature ID"), mcp.Required()),
		mcp.WithString("impact", mcp.Description("Impact level: low, medium, high")),
		mcp.WithString("complexity", mcp.Description("Complexity level: low, medium, high")),
		mcp.WithString("note", mcp.Description("Optional note")),
	), mcpFeatureTriage)

	s.AddTool(mcp.NewTool("bira_feature_start",
		mcp.WithDescription("Mark a feature as in-progress."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("feature_id", mcp.Description("Feature ID"), mcp.Required()),
		mcp.WithString("note", mcp.Description("Optional note")),
	), mcpFeatureStart)

	s.AddTool(mcp.NewTool("bira_feature_done",
		mcp.WithDescription("Mark a feature as done."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("feature_id", mcp.Description("Feature ID"), mcp.Required()),
		mcp.WithString("note", mcp.Description("Optional note")),
	), mcpFeatureDone)

	s.AddTool(mcp.NewTool("bira_feature_reject",
		mcp.WithDescription("Reject a feature."),
		mcp.WithString("project_id", mcp.Description("Project ID"), mcp.Required()),
		mcp.WithString("feature_id", mcp.Description("Feature ID"), mcp.Required()),
		mcp.WithString("note", mcp.Description("Reason for rejection")),
	), mcpFeatureReject)
}

// mcpResult encodes v as JSON text content.
func mcpResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: string(data)},
		},
	}, nil
}

func mcpErr(err error) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: err.Error()},
		},
	}, nil
}

// --- context handler ---

func mcpContext(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	full := req.GetBool("full", false)

	meta, err := store.LoadProjectMeta(projectID)
	if err != nil {
		return mcpErr(err)
	}
	ideas, err := loadAllIdeas(projectID)
	if err != nil {
		return mcpErr(err)
	}
	bugs, err := loadAllBugs(projectID)
	if err != nil {
		return mcpErr(err)
	}
	features, err := loadAllFeatures(projectID)
	if err != nil {
		return mcpErr(err)
	}

	if !full {
		return mcpResult(map[string]any{
			"project_id":   projectID,
			"project_name": meta.Name,
			"ideas":        len(ideas),
			"bugs":         len(bugs),
			"features":     len(features),
		})
	}

	countByStatus := func(statuses []string) []map[string]any {
		m := map[string]int{}
		order := []string{}
		seen := map[string]bool{}
		for _, s := range statuses {
			m[s]++
			if !seen[s] {
				seen[s] = true
				order = append(order, s)
			}
		}
		out := make([]map[string]any, 0, len(order))
		for _, s := range order {
			out = append(out, map[string]any{"status": s, "count": m[s]})
		}
		return out
	}

	ideaStatuses := make([]string, len(ideas))
	for i, x := range ideas {
		ideaStatuses[i] = x.Status
	}
	bugStatuses := make([]string, len(bugs))
	for i, x := range bugs {
		bugStatuses[i] = x.Status
	}
	featStatuses := make([]string, len(features))
	for i, x := range features {
		featStatuses[i] = x.Status
	}

	return mcpResult(map[string]any{
		"project_id":    projectID,
		"project_name":  meta.Name,
		"idea_count":    len(ideas),
		"bug_count":     len(bugs),
		"feature_count": len(features),
		"ideas":         countByStatus(ideaStatuses),
		"bugs":          countByStatus(bugStatuses),
		"features":      countByStatus(featStatuses),
	})
}

// --- idea handlers ---

func mcpIdeaAdd(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	title, err := req.RequireString("title")
	if err != nil {
		return mcpErr(err)
	}
	desc := req.GetString("description", "")
	priority := req.GetString("priority", "")
	tags := req.GetString("tags", "")

	if priority != "" && !models.IsValidPriority(priority) {
		return mcpErr(fmt.Errorf("invalid priority %q", priority))
	}

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	now := time.Now().UTC()
	idea := models.Idea{
		ID:          store.NewID(),
		ProjectID:   projectID,
		Title:       title,
		Description: desc,
		Priority:    priority,
		Tags:        parseTags(tags),
		Status:      models.IdeaStatusInbox,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	path := fmt.Sprintf("%s/ideas/%s.json", projectDir, idea.ID)
	if err := store.SaveJSON(path, &idea); err != nil {
		return mcpErr(err)
	}
	return mcpResult(&idea)
}

func mcpIdeaList(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	statusFilter := req.GetString("status", "")

	ideas, err := loadAllIdeas(projectID)
	if err != nil {
		return mcpErr(err)
	}

	if statusFilter != "" {
		var filtered []models.Idea
		for _, x := range ideas {
			if x.Status == statusFilter {
				filtered = append(filtered, x)
			}
		}
		ideas = filtered
	}
	return mcpResult(ideas)
}

func mcpIdeaShow(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	ideaID, err := req.RequireString("idea_id")
	if err != nil {
		return mcpErr(err)
	}
	idea, err := loadIdea(projectID, ideaID)
	if err != nil {
		return mcpErr(err)
	}
	return mcpResult(idea)
}

func mcpIdeaTriage(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	ideaID, err := req.RequireString("idea_id")
	if err != nil {
		return mcpErr(err)
	}
	priority := req.GetString("priority", "")
	note := req.GetString("note", "")

	if priority != "" && !models.IsValidPriority(priority) {
		return mcpErr(fmt.Errorf("invalid priority %q", priority))
	}

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	idea, err := loadIdea(projectID, ideaID)
	if err != nil {
		return mcpErr(err)
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

	path := fmt.Sprintf("%s/ideas/%s.json", projectDir, idea.ID)
	if err := store.SaveJSON(path, idea); err != nil {
		return mcpErr(err)
	}
	return mcpResult(idea)
}

func mcpIdeaPromote(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	ideaID, err := req.RequireString("idea_id")
	if err != nil {
		return mcpErr(err)
	}
	impact := req.GetString("impact", "")
	complexity := req.GetString("complexity", "")

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	idea, err := loadIdea(projectID, ideaID)
	if err != nil {
		return mcpErr(err)
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

	featurePath := fmt.Sprintf("%s/features/%s.json", projectDir, feature.ID)
	if err := store.SaveJSON(featurePath, &feature); err != nil {
		return mcpErr(err)
	}

	idea.Status = models.IdeaStatusPromoted
	idea.UpdatedAt = now
	ideaPath := fmt.Sprintf("%s/ideas/%s.json", projectDir, idea.ID)
	if err := store.SaveJSON(ideaPath, idea); err != nil {
		return mcpErr(err)
	}

	return mcpResult(&feature)
}

func mcpIdeaReject(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	ideaID, err := req.RequireString("idea_id")
	if err != nil {
		return mcpErr(err)
	}
	note := req.GetString("note", "")

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	idea, err := loadIdea(projectID, ideaID)
	if err != nil {
		return mcpErr(err)
	}

	now := time.Now().UTC()
	idea.Status = models.IdeaStatusRejected
	if note != "" {
		idea.Notes = append(idea.Notes, models.Note{Timestamp: now, Body: note})
	}
	idea.UpdatedAt = now

	path := fmt.Sprintf("%s/ideas/%s.json", projectDir, idea.ID)
	if err := store.SaveJSON(path, idea); err != nil {
		return mcpErr(err)
	}
	return mcpResult(idea)
}

// --- bug handlers ---

func mcpBugCreate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	title, err := req.RequireString("title")
	if err != nil {
		return mcpErr(err)
	}
	desc := req.GetString("description", "")
	criticality := req.GetString("criticality", "")
	reportedBy := req.GetString("reported_by", "")
	tags := req.GetString("tags", "")

	if criticality != "" && !models.IsValidCriticality(criticality) {
		return mcpErr(fmt.Errorf("invalid criticality %q", criticality))
	}

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	now := time.Now().UTC()
	bug := models.Bug{
		ID:          store.NewID(),
		ProjectID:   projectID,
		Title:       title,
		Description: desc,
		Criticality: criticality,
		ReportedBy:  reportedBy,
		Tags:        parseTags(tags),
		Status:      models.BugStatusOpen,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	path := fmt.Sprintf("%s/bugs/%s.json", projectDir, bug.ID)
	if err := store.SaveJSON(path, &bug); err != nil {
		return mcpErr(err)
	}
	return mcpResult(&bug)
}

func mcpBugList(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	statusFilter := req.GetString("status", "")

	bugs, err := loadAllBugs(projectID)
	if err != nil {
		return mcpErr(err)
	}

	if statusFilter != "" {
		var filtered []models.Bug
		for _, b := range bugs {
			if b.Status == statusFilter {
				filtered = append(filtered, b)
			}
		}
		bugs = filtered
	}
	return mcpResult(bugs)
}

func mcpBugShow(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	bugID, err := req.RequireString("bug_id")
	if err != nil {
		return mcpErr(err)
	}
	bug, err := loadBug(projectID, bugID)
	if err != nil {
		return mcpErr(err)
	}
	return mcpResult(bug)
}

func mcpBugTriage(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	bugID, err := req.RequireString("bug_id")
	if err != nil {
		return mcpErr(err)
	}
	criticality := req.GetString("criticality", "")
	note := req.GetString("note", "")

	if criticality != "" && !models.IsValidCriticality(criticality) {
		return mcpErr(fmt.Errorf("invalid criticality %q", criticality))
	}

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	bug, err := loadBug(projectID, bugID)
	if err != nil {
		return mcpErr(err)
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

	path := fmt.Sprintf("%s/bugs/%s.json", projectDir, bug.ID)
	if err := store.SaveJSON(path, bug); err != nil {
		return mcpErr(err)
	}
	return mcpResult(bug)
}

func mcpBugStart(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	bugID, err := req.RequireString("bug_id")
	if err != nil {
		return mcpErr(err)
	}
	note := req.GetString("note", "")

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	bug, err := loadBug(projectID, bugID)
	if err != nil {
		return mcpErr(err)
	}

	now := time.Now().UTC()
	bug.Status = models.BugStatusInProgress
	if note != "" {
		bug.Notes = append(bug.Notes, models.Note{Timestamp: now, Body: note})
	}
	bug.UpdatedAt = now

	path := fmt.Sprintf("%s/bugs/%s.json", projectDir, bug.ID)
	if err := store.SaveJSON(path, bug); err != nil {
		return mcpErr(err)
	}
	return mcpResult(bug)
}

func mcpBugDone(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	bugID, err := req.RequireString("bug_id")
	if err != nil {
		return mcpErr(err)
	}
	note := req.GetString("note", "")

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	bug, err := loadBug(projectID, bugID)
	if err != nil {
		return mcpErr(err)
	}

	now := time.Now().UTC()
	bug.Status = models.BugStatusFixed
	if note != "" {
		bug.Notes = append(bug.Notes, models.Note{Timestamp: now, Body: note})
	}
	bug.UpdatedAt = now

	path := fmt.Sprintf("%s/bugs/%s.json", projectDir, bug.ID)
	if err := store.SaveJSON(path, bug); err != nil {
		return mcpErr(err)
	}
	return mcpResult(bug)
}

func mcpBugWontFix(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	bugID, err := req.RequireString("bug_id")
	if err != nil {
		return mcpErr(err)
	}
	note := req.GetString("note", "")

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	bug, err := loadBug(projectID, bugID)
	if err != nil {
		return mcpErr(err)
	}

	now := time.Now().UTC()
	bug.Status = models.BugStatusWontFix
	if note != "" {
		bug.Notes = append(bug.Notes, models.Note{Timestamp: now, Body: note})
	}
	bug.UpdatedAt = now

	path := fmt.Sprintf("%s/bugs/%s.json", projectDir, bug.ID)
	if err := store.SaveJSON(path, bug); err != nil {
		return mcpErr(err)
	}
	return mcpResult(bug)
}

// --- feature handlers ---

func mcpFeatureAdd(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	title, err := req.RequireString("title")
	if err != nil {
		return mcpErr(err)
	}
	desc := req.GetString("description", "")
	impact := req.GetString("impact", "")
	complexity := req.GetString("complexity", "")
	tags := req.GetString("tags", "")

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	now := time.Now().UTC()
	feature := models.Feature{
		ID:          store.NewID(),
		ProjectID:   projectID,
		Title:       title,
		Description: desc,
		Impact:      impact,
		Complexity:  complexity,
		Tags:        parseTags(tags),
		Status:      models.FeatureStatusProposed,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	path := fmt.Sprintf("%s/features/%s.json", projectDir, feature.ID)
	if err := store.SaveJSON(path, &feature); err != nil {
		return mcpErr(err)
	}
	return mcpResult(&feature)
}

func mcpFeatureList(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	statusFilter := req.GetString("status", "")

	features, err := loadAllFeatures(projectID)
	if err != nil {
		return mcpErr(err)
	}

	if statusFilter != "" {
		var filtered []models.Feature
		for _, f := range features {
			if f.Status == statusFilter {
				filtered = append(filtered, f)
			}
		}
		features = filtered
	}
	return mcpResult(features)
}

func mcpFeatureShow(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	featureID, err := req.RequireString("feature_id")
	if err != nil {
		return mcpErr(err)
	}
	feature, err := loadFeature(projectID, featureID)
	if err != nil {
		return mcpErr(err)
	}
	return mcpResult(feature)
}

func mcpFeatureTriage(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	featureID, err := req.RequireString("feature_id")
	if err != nil {
		return mcpErr(err)
	}
	impact := req.GetString("impact", "")
	complexity := req.GetString("complexity", "")
	note := req.GetString("note", "")

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	feature, err := loadFeature(projectID, featureID)
	if err != nil {
		return mcpErr(err)
	}

	now := time.Now().UTC()
	feature.Status = models.FeatureStatusTriaged
	if impact != "" {
		feature.Impact = impact
	}
	if complexity != "" {
		feature.Complexity = complexity
	}
	if note != "" {
		feature.Notes = append(feature.Notes, models.Note{Timestamp: now, Body: note})
	}
	feature.UpdatedAt = now

	path := fmt.Sprintf("%s/features/%s.json", projectDir, feature.ID)
	if err := store.SaveJSON(path, feature); err != nil {
		return mcpErr(err)
	}
	return mcpResult(feature)
}

func mcpFeatureStart(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	featureID, err := req.RequireString("feature_id")
	if err != nil {
		return mcpErr(err)
	}
	note := req.GetString("note", "")

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	feature, err := loadFeature(projectID, featureID)
	if err != nil {
		return mcpErr(err)
	}

	now := time.Now().UTC()
	feature.Status = models.FeatureStatusInProgress
	if note != "" {
		feature.Notes = append(feature.Notes, models.Note{Timestamp: now, Body: note})
	}
	feature.UpdatedAt = now

	path := fmt.Sprintf("%s/features/%s.json", projectDir, feature.ID)
	if err := store.SaveJSON(path, feature); err != nil {
		return mcpErr(err)
	}
	return mcpResult(feature)
}

func mcpFeatureDone(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	featureID, err := req.RequireString("feature_id")
	if err != nil {
		return mcpErr(err)
	}
	note := req.GetString("note", "")

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	feature, err := loadFeature(projectID, featureID)
	if err != nil {
		return mcpErr(err)
	}

	now := time.Now().UTC()
	feature.Status = models.FeatureStatusDone
	if note != "" {
		feature.Notes = append(feature.Notes, models.Note{Timestamp: now, Body: note})
	}
	feature.UpdatedAt = now

	path := fmt.Sprintf("%s/features/%s.json", projectDir, feature.ID)
	if err := store.SaveJSON(path, feature); err != nil {
		return mcpErr(err)
	}
	return mcpResult(feature)
}

func mcpFeatureReject(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projectID, err := req.RequireString("project_id")
	if err != nil {
		return mcpErr(err)
	}
	featureID, err := req.RequireString("feature_id")
	if err != nil {
		return mcpErr(err)
	}
	note := req.GetString("note", "")

	projectDir, err := store.ProjectDir(projectID)
	if err != nil {
		return mcpErr(err)
	}
	lock, err := store.AcquireLock(projectDir)
	if err != nil {
		return mcpErr(err)
	}
	defer lock.Release()

	feature, err := loadFeature(projectID, featureID)
	if err != nil {
		return mcpErr(err)
	}

	now := time.Now().UTC()
	feature.Status = models.FeatureStatusRejected
	if note != "" {
		feature.Notes = append(feature.Notes, models.Note{Timestamp: now, Body: note})
	}
	feature.UpdatedAt = now

	path := fmt.Sprintf("%s/features/%s.json", projectDir, feature.ID)
	if err := store.SaveJSON(path, feature); err != nil {
		return mcpErr(err)
	}
	return mcpResult(feature)
}
