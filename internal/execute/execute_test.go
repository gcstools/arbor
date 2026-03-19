package execute

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"arbor/internal/gitutil"
	"arbor/internal/model"
	"arbor/internal/planner"
)

func TestApplyCreatesWorktreeAndCopiesEnv(t *testing.T) {
	root := initRepo(t)
	sourceEnv := filepath.Join(root, ".env")
	writeFile(t, sourceEnv, "A=1")

	summary, err := Runner{}.Apply(context.Background(), planner.CreatePlan{
		RepoState: gitutil.RepoState{Root: root},
		Worktrees: []model.WorktreePlan{
			{
				Name:    "feature",
				Branch:  "feature",
				BaseRef: "main",
				Path:    filepath.Join(filepath.Dir(root), "repo-feature"),
				EnvActions: []model.EnvPlan{
					{Candidate: model.EnvCandidate{ID: "env", SourcePath: sourceEnv, TargetPath: ".env"}, Action: model.ActionCopy},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if summary.HasFailure {
		t.Fatalf("unexpected failure: %#v", summary)
	}
	if !summary.Worktrees[0].Created {
		t.Fatal("expected worktree created")
	}
	if _, err := os.Stat(filepath.Join(summary.Worktrees[0].Path, ".env")); err != nil {
		t.Fatalf("expected copied env file: %v", err)
	}
}

func TestApplyRunsCommand(t *testing.T) {
	root := initRepo(t)
	worktreePath := filepath.Join(filepath.Dir(root), "repo-run")
	var stdout bytes.Buffer

	summary, err := Runner{Stdout: &stdout}.Apply(context.Background(), planner.CreatePlan{
		RepoState: gitutil.RepoState{Root: root},
		Worktrees: []model.WorktreePlan{
			{
				Name:    "run",
				Branch:  "run",
				BaseRef: "main",
				Path:    worktreePath,
				Commands: []model.CommandExecution{
					{Candidate: model.CommandCandidate{ID: "echo", Label: "echo", Command: "echo ok", Scope: model.CommandScopePerWorktree}, Approved: true},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if summary.Worktrees[0].CommandResult[0].Error != "" {
		t.Fatalf("unexpected command error: %#v", summary.Worktrees[0].CommandResult[0])
	}
	if !strings.Contains(stdout.String(), "ok") {
		t.Fatalf("expected command output, got %q", stdout.String())
	}
}

func TestApplyUsesExistingBranchWithoutCreatingIt(t *testing.T) {
	root := initRepo(t)
	runGit(t, root, "branch", "feature/existing", "main")
	worktreePath := filepath.Join(filepath.Dir(root), "repo-existing")

	summary, err := Runner{}.Apply(context.Background(), planner.CreatePlan{
		RepoState: gitutil.RepoState{Root: root},
		Worktrees: []model.WorktreePlan{
			{
				Name:       "existing",
				Branch:     "feature/existing",
				BranchMode: model.BranchModeExisting,
				Path:       worktreePath,
			},
		},
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if summary.HasFailure {
		t.Fatalf("unexpected failure: %#v", summary)
	}
	if !summary.Worktrees[0].Created {
		t.Fatal("expected worktree created")
	}
}

func TestApplyRejectsExistingBranch(t *testing.T) {
	root := initRepo(t)
	runGit(t, root, "branch", "feature", "main")
	summary, err := Runner{}.Apply(context.Background(), planner.CreatePlan{
		RepoState: gitutil.RepoState{Root: root},
		Worktrees: []model.WorktreePlan{
			{Name: "feature", Branch: "feature", BaseRef: "main", Path: filepath.Join(filepath.Dir(root), "repo-feature")},
		},
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if !summary.HasFailure {
		t.Fatal("expected failure")
	}
}

func TestApplyOpensCreatedWorktree(t *testing.T) {
	root := initRepo(t)
	expectedPath := filepath.Join(filepath.Dir(root), "repo-feature")
	var opened []string
	previous := runOpenCommand
	runOpenCommand = func(_ context.Context, app, path string) error {
		opened = append(opened, app+":"+path)
		return nil
	}
	defer func() {
		runOpenCommand = previous
	}()

	summary, err := Runner{}.Apply(context.Background(), planner.CreatePlan{
		RepoState: gitutil.RepoState{Root: root},
		OpenApp:   "cursor",
		Worktrees: []model.WorktreePlan{
			{
				Name:    "feature",
				Branch:  "feature",
				BaseRef: "main",
				Path:    expectedPath,
			},
		},
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(opened) != 1 {
		t.Fatalf("expected one open call, got %#v", opened)
	}
	if summary.OpenApp != "cursor" {
		t.Fatalf("unexpected open app: %q", summary.OpenApp)
	}
	if summary.OpenedPath != expectedPath {
		t.Fatalf("unexpected opened path: %q", summary.OpenedPath)
	}
	if summary.OpenError != "" {
		t.Fatalf("unexpected open error: %q", summary.OpenError)
	}
}

func TestApplyOpenFailureDoesNotMarkSetupFailed(t *testing.T) {
	root := initRepo(t)
	previous := runOpenCommand
	runOpenCommand = func(_ context.Context, app, path string) error {
		return errors.New("open failed")
	}
	defer func() {
		runOpenCommand = previous
	}()

	summary, err := Runner{}.Apply(context.Background(), planner.CreatePlan{
		RepoState: gitutil.RepoState{Root: root},
		OpenApp:   "cursor",
		Worktrees: []model.WorktreePlan{
			{
				Name:    "feature",
				Branch:  "feature",
				BaseRef: "main",
				Path:    filepath.Join(filepath.Dir(root), "repo-feature"),
			},
		},
	})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if summary.HasFailure {
		t.Fatalf("expected setup success despite open failure: %#v", summary)
	}
	if summary.OpenApp != "cursor" {
		t.Fatalf("unexpected open app: %q", summary.OpenApp)
	}
	if summary.OpenError != "open failed" {
		t.Fatalf("unexpected open error: %q", summary.OpenError)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	runGit(t, root, "init", "-b", "main")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	writeFile(t, filepath.Join(root, "README.md"), "hello")
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "init")
	return root
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
