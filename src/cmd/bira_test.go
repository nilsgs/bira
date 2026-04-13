package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bira/internal/models"
)

type testEnv struct {
	t       *testing.T
	home    string
	workDir string
	out     bytes.Buffer
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	home := t.TempDir()
	work := t.TempDir()
	t.Setenv("BIRA_HOME", home)
	return &testEnv{t: t, home: home, workDir: work}
}

func (e *testEnv) run(args ...string) string {
	e.t.Helper()
	e.out.Reset()

	root := NewRootCmd()
	root.SetOut(&e.out)
	root.SetErr(&e.out)
	root.SetArgs(args)

	if err := root.Execute(); err != nil {
		e.t.Fatalf("bira %s: %v\noutput: %s", strings.Join(args, " "), err, e.out.String())
	}
	return e.out.String()
}

func (e *testEnv) initProject(name string) string {
	e.t.Helper()
	orig, _ := os.Getwd()
	if err := os.Chdir(e.workDir); err != nil {
		e.t.Fatalf("Chdir: %v", err)
	}
	e.t.Cleanup(func() { os.Chdir(orig) })

	out := e.run("init", "--name", name, "--json")
	var p models.Project
	if err := json.Unmarshal([]byte(out), &p); err != nil {
		e.t.Fatalf("parse init JSON: %v\nraw: %s", err, out)
	}
	return p.ID
}

// TestInit verifies that init creates a project with a valid ID
func TestInit(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("test-init")
	if len(pid) != 8 {
		t.Errorf("project ID length = %d, want 8", len(pid))
	}

	// Second init should fail
	e.out.Reset()
	root := NewRootCmd()
	root.SetOut(&e.out)
	root.SetErr(&e.out)
	root.SetArgs([]string{"init", "--name", "dup"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error on double init")
	}
}

// TestIdeaLifecycle tests add, list, show, triage, promote
func TestIdeaLifecycle(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("idea-proj")

	// add
	out := e.run("idea", "add", "My cool idea", "--project", pid, "--json")
	var idea models.Idea
	if err := json.Unmarshal([]byte(out), &idea); err != nil {
		t.Fatalf("parse idea JSON: %v\nraw: %s", err, out)
	}
	if idea.Title != "My cool idea" {
		t.Errorf("idea title = %q, want 'My cool idea'", idea.Title)
	}
	if idea.Status != models.IdeaStatusInbox {
		t.Errorf("idea status = %q, want inbox", idea.Status)
	}
	if idea.ProjectID != pid {
		t.Errorf("idea project_id = %q, want %q", idea.ProjectID, pid)
	}

	// list
	out = e.run("idea", "list", "--project", pid, "--json")
	var ideas []models.Idea
	if err := json.Unmarshal([]byte(out), &ideas); err != nil {
		t.Fatalf("parse ideas list JSON: %v\nraw: %s", err, out)
	}
	if len(ideas) != 1 {
		t.Errorf("idea count = %d, want 1", len(ideas))
	}

	// show
	out = e.run("idea", "show", idea.ID, "--project", pid, "--json")
	var shown models.Idea
	if err := json.Unmarshal([]byte(out), &shown); err != nil {
		t.Fatalf("parse idea show JSON: %v\nraw: %s", err, out)
	}
	if shown.ID != idea.ID {
		t.Errorf("shown idea ID = %q, want %q", shown.ID, idea.ID)
	}

	// triage
	out = e.run("idea", "triage", idea.ID, "--project", pid, "--priority", "high", "--json")
	var triaged models.Idea
	if err := json.Unmarshal([]byte(out), &triaged); err != nil {
		t.Fatalf("parse triaged idea JSON: %v\nraw: %s", err, out)
	}
	if triaged.Status != models.IdeaStatusTriaged {
		t.Errorf("triaged status = %q, want triaged", triaged.Status)
	}
	if triaged.Priority != "high" {
		t.Errorf("triaged priority = %q, want high", triaged.Priority)
	}

	// promote
	out = e.run("idea", "promote", idea.ID, "--project", pid, "--json")
	var feature models.Feature
	if err := json.Unmarshal([]byte(out), &feature); err != nil {
		t.Fatalf("parse promoted feature JSON: %v\nraw: %s", err, out)
	}
	if feature.PromotedFrom != idea.ID {
		t.Errorf("feature.promoted_from = %q, want %q", feature.PromotedFrom, idea.ID)
	}
	if feature.Status != models.FeatureStatusProposed {
		t.Errorf("feature status = %q, want proposed", feature.Status)
	}
	if feature.Title != idea.Title {
		t.Errorf("feature title = %q, want %q", feature.Title, idea.Title)
	}

	// verify idea is marked promoted
	out = e.run("idea", "show", idea.ID, "--project", pid, "--json")
	var promotedIdea models.Idea
	if err := json.Unmarshal([]byte(out), &promotedIdea); err != nil {
		t.Fatalf("parse promoted idea JSON: %v\nraw: %s", err, out)
	}
	if promotedIdea.Status != models.IdeaStatusPromoted {
		t.Errorf("idea status after promote = %q, want promoted", promotedIdea.Status)
	}
}

// TestIdeaPlan tests plan sidecar retrieval via show
func TestIdeaPlan(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("plan-proj")

	out := e.run("idea", "add", "Plan test idea", "--project", pid, "--json")
	var idea models.Idea
	if err := json.Unmarshal([]byte(out), &idea); err != nil {
		t.Fatalf("parse idea JSON: %v", err)
	}

	// Write the plan sidecar directly to the store location
	ideasDir := filepath.Join(e.home, "projects", pid, "ideas")
	planContent := "# My Plan\n\nStep 1: Do something\nStep 2: Do more\n"
	planPath := filepath.Join(ideasDir, idea.ID+".plan.md")
	if err := os.MkdirAll(ideasDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		t.Fatalf("write plan: %v", err)
	}

	// show should include plan
	out = e.run("idea", "show", idea.ID, "--project", pid, "--json")
	var shown models.Idea
	if err := json.Unmarshal([]byte(out), &shown); err != nil {
		t.Fatalf("parse idea show JSON: %v\nraw: %s", err, out)
	}
	if shown.Plan != planContent {
		t.Errorf("idea plan = %q, want %q", shown.Plan, planContent)
	}
}

// TestBugLifecycle tests create, list, show, triage, start, done
func TestBugLifecycle(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("bug-proj")

	// create
	out := e.run("bug", "create", "Login crash", "--project", pid, "--json")
	var bug models.Bug
	if err := json.Unmarshal([]byte(out), &bug); err != nil {
		t.Fatalf("parse bug JSON: %v\nraw: %s", err, out)
	}
	if bug.Title != "Login crash" {
		t.Errorf("bug title = %q, want 'Login crash'", bug.Title)
	}
	if bug.Status != models.BugStatusOpen {
		t.Errorf("bug status = %q, want open", bug.Status)
	}

	// list
	out = e.run("bug", "list", "--project", pid, "--json")
	var bugs []models.Bug
	if err := json.Unmarshal([]byte(out), &bugs); err != nil {
		t.Fatalf("parse bugs list JSON: %v\nraw: %s", err, out)
	}
	if len(bugs) != 1 {
		t.Errorf("bug count = %d, want 1", len(bugs))
	}

	// show
	out = e.run("bug", "show", bug.ID, "--project", pid, "--json")
	var shown models.Bug
	if err := json.Unmarshal([]byte(out), &shown); err != nil {
		t.Fatalf("parse bug show JSON: %v", err)
	}
	if shown.ID != bug.ID {
		t.Errorf("shown bug ID = %q, want %q", shown.ID, bug.ID)
	}

	// triage
	out = e.run("bug", "triage", bug.ID, "--project", pid, "--criticality", "high", "--json")
	var triaged models.Bug
	if err := json.Unmarshal([]byte(out), &triaged); err != nil {
		t.Fatalf("parse triaged bug JSON: %v", err)
	}
	if triaged.Status != models.BugStatusTriaged {
		t.Errorf("triaged bug status = %q, want triaged", triaged.Status)
	}
	if triaged.Criticality != "high" {
		t.Errorf("triaged bug criticality = %q, want high", triaged.Criticality)
	}

	// start
	out = e.run("bug", "start", bug.ID, "--project", pid, "--json")
	var started models.Bug
	if err := json.Unmarshal([]byte(out), &started); err != nil {
		t.Fatalf("parse started bug JSON: %v", err)
	}
	if started.Status != models.BugStatusInProgress {
		t.Errorf("started bug status = %q, want in-progress", started.Status)
	}

	// done
	out = e.run("bug", "done", bug.ID, "--project", pid, "--json")
	var done models.Bug
	if err := json.Unmarshal([]byte(out), &done); err != nil {
		t.Fatalf("parse done bug JSON: %v", err)
	}
	if done.Status != models.BugStatusFixed {
		t.Errorf("done bug status = %q, want fixed", done.Status)
	}
}

// TestBugWontFix tests wont-fix with note
func TestBugWontFix(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("wontfix-proj")

	out := e.run("bug", "create", "Not a bug", "--project", pid, "--json")
	var bug models.Bug
	if err := json.Unmarshal([]byte(out), &bug); err != nil {
		t.Fatalf("parse bug JSON: %v", err)
	}

	out = e.run("bug", "wont-fix", bug.ID, "--project", pid, "--note", "By design", "--json")
	var wf models.Bug
	if err := json.Unmarshal([]byte(out), &wf); err != nil {
		t.Fatalf("parse wont-fix bug JSON: %v\nraw: %s", err, out)
	}
	if wf.Status != models.BugStatusWontFix {
		t.Errorf("wont-fix bug status = %q, want wont-fix", wf.Status)
	}
	if len(wf.Notes) != 1 {
		t.Errorf("wont-fix note count = %d, want 1", len(wf.Notes))
	}
	if wf.Notes[0].Body != "By design" {
		t.Errorf("wont-fix note = %q, want 'By design'", wf.Notes[0].Body)
	}
}

// TestFeatureLifecycle tests add, list, triage, start, done
func TestFeatureLifecycle(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("feature-proj")

	// add
	out := e.run("feature", "add", "Dark mode", "--project", pid, "--json")
	var f models.Feature
	if err := json.Unmarshal([]byte(out), &f); err != nil {
		t.Fatalf("parse feature JSON: %v\nraw: %s", err, out)
	}
	if f.Title != "Dark mode" {
		t.Errorf("feature title = %q, want 'Dark mode'", f.Title)
	}
	if f.Status != models.FeatureStatusProposed {
		t.Errorf("feature status = %q, want proposed", f.Status)
	}

	// list
	out = e.run("feature", "list", "--project", pid, "--json")
	var features []models.Feature
	if err := json.Unmarshal([]byte(out), &features); err != nil {
		t.Fatalf("parse features list JSON: %v\nraw: %s", err, out)
	}
	if len(features) != 1 {
		t.Errorf("feature count = %d, want 1", len(features))
	}

	// triage
	out = e.run("feature", "triage", f.ID, "--project", pid, "--impact", "high", "--complexity", "medium", "--json")
	var triaged models.Feature
	if err := json.Unmarshal([]byte(out), &triaged); err != nil {
		t.Fatalf("parse triaged feature JSON: %v", err)
	}
	if triaged.Status != models.FeatureStatusTriaged {
		t.Errorf("triaged feature status = %q, want triaged", triaged.Status)
	}
	if triaged.Impact != "high" {
		t.Errorf("feature impact = %q, want high", triaged.Impact)
	}
	if triaged.Complexity != "medium" {
		t.Errorf("feature complexity = %q, want medium", triaged.Complexity)
	}

	// start
	out = e.run("feature", "start", f.ID, "--project", pid, "--json")
	var started models.Feature
	if err := json.Unmarshal([]byte(out), &started); err != nil {
		t.Fatalf("parse started feature JSON: %v", err)
	}
	if started.Status != models.FeatureStatusInProgress {
		t.Errorf("started feature status = %q, want in-progress", started.Status)
	}

	// done
	out = e.run("feature", "done", f.ID, "--project", pid, "--json")
	var done models.Feature
	if err := json.Unmarshal([]byte(out), &done); err != nil {
		t.Fatalf("parse done feature JSON: %v", err)
	}
	if done.Status != models.FeatureStatusDone {
		t.Errorf("done feature status = %q, want done", done.Status)
	}
}

// TestFeatureReject tests reject with note
func TestFeatureReject(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("reject-proj")

	out := e.run("feature", "add", "Bad idea feature", "--project", pid, "--json")
	var f models.Feature
	if err := json.Unmarshal([]byte(out), &f); err != nil {
		t.Fatalf("parse feature JSON: %v", err)
	}

	out = e.run("feature", "reject", f.ID, "--project", pid, "--note", "Out of scope", "--json")
	var rejected models.Feature
	if err := json.Unmarshal([]byte(out), &rejected); err != nil {
		t.Fatalf("parse rejected feature JSON: %v\nraw: %s", err, out)
	}
	if rejected.Status != models.FeatureStatusRejected {
		t.Errorf("rejected feature status = %q, want rejected", rejected.Status)
	}
	if len(rejected.Notes) != 1 {
		t.Errorf("reject note count = %d, want 1", len(rejected.Notes))
	}
}

// TestContext tests context command counts
func TestContext(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("ctx-proj")

	e.run("idea", "add", "Idea 1", "--project", pid)
	e.run("bug", "create", "Bug 1", "--project", pid)
	e.run("feature", "add", "Feature 1", "--project", pid)

	type contextSummary struct {
		ProjectID   string `json:"project_id"`
		ProjectName string `json:"project_name"`
		Ideas       int    `json:"ideas"`
		Bugs        int    `json:"bugs"`
		Features    int    `json:"features"`
	}

	out := e.run("context", "--project", pid, "--json")
	var ctx contextSummary
	if err := json.Unmarshal([]byte(out), &ctx); err != nil {
		t.Fatalf("parse context JSON: %v\nraw: %s", err, out)
	}
	if ctx.Ideas != 1 {
		t.Errorf("context ideas = %d, want 1", ctx.Ideas)
	}
	if ctx.Bugs != 1 {
		t.Errorf("context bugs = %d, want 1", ctx.Bugs)
	}
	if ctx.Features != 1 {
		t.Errorf("context features = %d, want 1", ctx.Features)
	}
}

// TestContextFull tests context --full breakdown
func TestContextFull(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("ctx-full-proj")

	e.run("idea", "add", "Idea inbox", "--project", pid)
	e.run("bug", "create", "Bug 1", "--project", pid)
	e.run("feature", "add", "Feature 1", "--project", pid)

	type statusCount struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	type fullCtx struct {
		ProjectID    string        `json:"project_id"`
		IdeaCount    int           `json:"idea_count"`
		BugCount     int           `json:"bug_count"`
		FeatureCount int           `json:"feature_count"`
		Ideas        []statusCount `json:"ideas"`
		Bugs         []statusCount `json:"bugs"`
		Features     []statusCount `json:"features"`
	}

	out := e.run("context", "--project", pid, "--full", "--json")
	var ctx fullCtx
	if err := json.Unmarshal([]byte(out), &ctx); err != nil {
		t.Fatalf("parse full context JSON: %v\nraw: %s", err, out)
	}
	if ctx.IdeaCount != 1 {
		t.Errorf("full context idea_count = %d, want 1", ctx.IdeaCount)
	}
	if ctx.BugCount != 1 {
		t.Errorf("full context bug_count = %d, want 1", ctx.BugCount)
	}
	if ctx.FeatureCount != 1 {
		t.Errorf("full context feature_count = %d, want 1", ctx.FeatureCount)
	}
	if len(ctx.Ideas) != 1 {
		t.Errorf("full context ideas status breakdown len = %d, want 1", len(ctx.Ideas))
	}
}
