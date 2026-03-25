package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"bira/internal/models"
)

// testEnv sets up an isolated bira environment for one test.
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

// run executes a bira command line and returns captured stdout.
// It resets rootCmd output and global flags before each invocation.
func (e *testEnv) run(args ...string) string {
	e.t.Helper()
	e.out.Reset()

	// Reset ALL package-level flag vars to defaults before every invocation.
	jsonOutput = false
	projectFlag = ""
	initName = ""
	taskAddFeature = ""
	taskAddDesc = ""
	taskAddAssign = ""
	taskAddTags = ""
	taskAddDependsOn = ""
	taskAddCriteria = nil
	taskAddFiles = ""
	taskListFeature = ""
	taskListStatus = ""
	taskUpdateStatus = ""
	taskUpdateDesc = ""
	taskUpdateAssign = ""
	taskUpdateTags = ""
	taskUpdateDependsOn = ""
	taskUpdateCriteria = nil
	taskUpdateFiles = ""
	featureAddDesc = ""
	featureAddAssign = ""
	featureAddTags = ""
	featureListStatus = ""
	featureUpdateStatus = ""
	featureUpdateDesc = ""
	featureUpdateAssign = ""
	featureUpdateTags = ""

	rootCmd.SetOut(&e.out)
	rootCmd.SetErr(&e.out)
	rootCmd.SetArgs(args)

	if err := rootCmd.Execute(); err != nil {
		e.t.Fatalf("bira %s: %v\noutput: %s", strings.Join(args, " "), err, e.out.String())
	}
	return e.out.String()
}

// initProject does chdir + bira init and returns the project ID from JSON output.
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

// --- Tests ---

func TestInit(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("test-init")
	if len(pid) != 8 {
		t.Errorf("project ID length = %d, want 8", len(pid))
	}

	// Second init should fail (already initialized)
	e.out.Reset()
	jsonOutput = false
	projectFlag = ""
	initName = ""
	rootCmd.SetOut(&e.out)
	rootCmd.SetErr(&e.out)
	rootCmd.SetArgs([]string{"init", "--name", "dup"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error on double init")
	}
}

func TestFeatureAdd(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("feat-proj")

	out := e.run("feature", "add", "my-feature", "--project", pid, "--json")
	var f models.Feature
	if err := json.Unmarshal([]byte(out), &f); err != nil {
		t.Fatalf("parse feature JSON: %v\nraw: %s", err, out)
	}
	if f.Name != "my-feature" {
		t.Errorf("feature name = %q, want my-feature", f.Name)
	}
	if f.ProjectID != pid {
		t.Errorf("feature project = %q, want %q", f.ProjectID, pid)
	}
}

func TestFeatureList(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("flist-proj")

	e.run("feature", "add", "f1", "--project", pid, "--json")
	e.run("feature", "add", "f2", "--project", pid, "--json")

	out := e.run("feature", "list", "--project", pid, "--json")
	var features []models.Feature
	if err := json.Unmarshal([]byte(out), &features); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	// 2 added + 1 backlog = 3
	if len(features) != 3 {
		t.Errorf("feature count = %d, want 3", len(features))
	}
}

func TestTaskAdd_ToBacklog(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("task-proj")

	out := e.run("task", "add", "backlog-task", "--project", pid, "--json")
	var task models.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if task.Title != "backlog-task" {
		t.Errorf("title = %q", task.Title)
	}
	if task.Status != models.StatusTodo {
		t.Errorf("status = %q, want todo", task.Status)
	}
}

func TestTaskAdd_ToFeature(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("tfeat-proj")

	fout := e.run("feature", "add", "feat", "--project", pid, "--json")
	var f models.Feature
	json.Unmarshal([]byte(fout), &f)

	out := e.run("task", "add", "feat-task", "--feature", f.ID, "--project", pid, "--json")
	var task models.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if task.FeatureID != f.ID {
		t.Errorf("featureID = %q, want %q", task.FeatureID, f.ID)
	}
}

func TestTaskList(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("tlist-proj")

	e.run("task", "add", "t1", "--project", pid, "--json")
	e.run("task", "add", "t2", "--project", pid, "--json")

	out := e.run("task", "list", "--project", pid, "--json")
	var tasks []models.Task
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if len(tasks) != 2 {
		t.Errorf("task count = %d, want 2", len(tasks))
	}
}

func TestTaskShow(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("tshow-proj")

	aout := e.run("task", "add", "showme", "--project", pid, "--json")
	var added models.Task
	json.Unmarshal([]byte(aout), &added)

	out := e.run("task", "show", added.ID, "--project", pid, "--json")
	var task models.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if task.Title != "showme" {
		t.Errorf("title = %q", task.Title)
	}
}

func TestTaskUpdate(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("tupd-proj")

	aout := e.run("task", "add", "updme", "--project", pid, "--json")
	var added models.Task
	json.Unmarshal([]byte(aout), &added)

	out := e.run("task", "update", added.ID, "--status", "in-progress", "--project", pid, "--json")
	var task models.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if task.Status != models.StatusInProgress {
		t.Errorf("status = %q, want in-progress", task.Status)
	}
}

func TestTaskDone(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("tdone-proj")

	aout := e.run("task", "add", "doneme", "--project", pid, "--json")
	var added models.Task
	json.Unmarshal([]byte(aout), &added)

	out := e.run("task", "done", added.ID, "--project", pid, "--json")
	var task models.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if task.Status != models.StatusDone {
		t.Errorf("status = %q, want done", task.Status)
	}
}

func TestTaskDelete(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("tdel-proj")

	aout := e.run("task", "add", "deleteme", "--project", pid, "--json")
	var added models.Task
	json.Unmarshal([]byte(aout), &added)

	e.run("task", "delete", added.ID, "--project", pid)

	// Verify it's gone by checking the task list is empty
	out := e.run("task", "list", "--project", pid, "--json")
	var tasks []models.Task
	if err := json.Unmarshal([]byte(out), &tasks); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks after delete, got %d", len(tasks))
	}
}

func TestContext(t *testing.T) {
	e := newTestEnv(t)
	e.initProject("ctx-proj")

	out := e.run("context", "--json")
	if !strings.Contains(out, "ctx-proj") {
		t.Errorf("context output missing project name:\n%s", out)
	}
}

func TestContextFull(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("ctxfull-proj")

	e.run("feature", "add", "ctx-feat", "--project", pid, "--json")
	e.run("task", "add", "ctx-task", "--project", pid, "--json")

	out := e.run("context", "--full", "--json")
	if !strings.Contains(out, "ctxfull-proj") {
		t.Errorf("missing project name")
	}
	if !strings.Contains(out, "ctx-feat") {
		t.Errorf("missing feature name")
	}
	// Full context shows task counts per feature, not individual task titles
	if !strings.Contains(out, "todo") {
		t.Errorf("missing task status counts")
	}
}

func TestJSONOutput(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("json-proj")

	out := e.run("task", "add", "json-task", "--project", pid, "--json")
	if !json.Valid([]byte(out)) {
		t.Errorf("output is not valid JSON:\n%s", out)
	}

	// Human output should not be JSON
	out = e.run("task", "list", "--project", pid)
	if json.Valid([]byte(out)) && strings.HasPrefix(strings.TrimSpace(out), "[") {
		t.Errorf("human output looks like JSON:\n%s", out)
	}
}

func TestTaskAddCriteria(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("tcrit-proj")

	out := e.run("task", "add", "crit-task", "--project", pid,
		"--criteria", "system returns 200",
		"--criteria", "response includes, comma check",
		"--json")
	var task models.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if len(task.AcceptanceCriteria) != 2 {
		t.Errorf("acceptance_criteria len = %d, want 2", len(task.AcceptanceCriteria))
	}
	if task.AcceptanceCriteria[0] != "system returns 200" {
		t.Errorf("criteria[0] = %q", task.AcceptanceCriteria[0])
	}
	if task.AcceptanceCriteria[1] != "response includes, comma check" {
		t.Errorf("criteria[1] = %q; commas should be preserved", task.AcceptanceCriteria[1])
	}
}

func TestTaskAddFiles(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("tfiles-proj")

	out := e.run("task", "add", "files-task", "--project", pid,
		"--files", "src/cmd/task.go,src/internal/models/task.go",
		"--json")
	var task models.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if len(task.Files) != 2 {
		t.Errorf("files len = %d, want 2", len(task.Files))
	}
	if task.Files[0] != "src/cmd/task.go" {
		t.Errorf("files[0] = %q", task.Files[0])
	}
}

func TestTaskNote(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("tnote-proj")

	aout := e.run("task", "add", "note-task", "--project", pid, "--json")
	var added models.Task
	json.Unmarshal([]byte(aout), &added)

	e.run("task", "note", added.ID, "agent observation one", "--project", pid)
	e.run("task", "note", added.ID, "agent observation two", "--project", pid)

	out := e.run("task", "show", added.ID, "--project", pid, "--json")
	var task models.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if len(task.Notes) != 2 {
		t.Errorf("notes len = %d, want 2", len(task.Notes))
	}
	if task.Notes[0].Body != "agent observation one" {
		t.Errorf("notes[0].body = %q", task.Notes[0].Body)
	}
	if task.Notes[1].Body != "agent observation two" {
		t.Errorf("notes[1].body = %q", task.Notes[1].Body)
	}
}

func TestFeatureNote(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("fnote-proj")

	fout := e.run("feature", "add", "note-feat", "--project", pid, "--json")
	var added models.Feature
	json.Unmarshal([]byte(fout), &added)

	e.run("feature", "note", added.ID, "feature finding alpha", "--project", pid)

	out := e.run("feature", "show", added.ID, "--project", pid, "--json")
	var feature models.Feature
	if err := json.Unmarshal([]byte(out), &feature); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if len(feature.Notes) != 1 {
		t.Errorf("notes len = %d, want 1", len(feature.Notes))
	}
	if feature.Notes[0].Body != "feature finding alpha" {
		t.Errorf("notes[0].body = %q", feature.Notes[0].Body)
	}
}

func TestTaskUpdateCriteria(t *testing.T) {
	e := newTestEnv(t)
	pid := e.initProject("tupdcrit-proj")

	aout := e.run("task", "add", "updcrit-task", "--project", pid, "--json")
	var added models.Task
	json.Unmarshal([]byte(aout), &added)

	out := e.run("task", "update", added.ID, "--project", pid,
		"--criteria", "updated criterion",
		"--files", "src/cmd/task.go",
		"--json")
	var task models.Task
	if err := json.Unmarshal([]byte(out), &task); err != nil {
		t.Fatalf("parse: %v\nraw: %s", err, out)
	}
	if len(task.AcceptanceCriteria) != 1 || task.AcceptanceCriteria[0] != "updated criterion" {
		t.Errorf("acceptance_criteria = %v", task.AcceptanceCriteria)
	}
	if len(task.Files) != 1 || task.Files[0] != "src/cmd/task.go" {
		t.Errorf("files = %v", task.Files)
	}
}
