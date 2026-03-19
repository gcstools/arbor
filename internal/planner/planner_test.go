package planner

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCreatePlanSingleName(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".env"), "A=1")
	writeFile(t, filepath.Join(root, "package.json"), `{"scripts":{"dev":"vite"}}`)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Names:          []string{"feature-auth"},
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if len(plan.Worktrees) != 1 {
		t.Fatalf("got %d worktrees", len(plan.Worktrees))
	}
	if plan.Worktrees[0].Branch != "feature-auth" {
		t.Fatalf("unexpected branch: %#v", plan.Worktrees[0])
	}
	if plan.Worktrees[0].BranchMode != "create" {
		t.Fatalf("unexpected branch mode: %#v", plan.Worktrees[0])
	}
	if !strings.Contains(RenderSummary(plan), "planning only") {
		t.Fatal("expected planning summary")
	}
}

func TestBuildCreatePlanWithExistingBranchUsesBranchNameByDefault(t *testing.T) {
	root := initRepo(t)
	runGit(t, root, "branch", "feature/auth", "main")

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Branch:         "feature/auth",
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if got := plan.Worktrees[0].Name; got != "feature/auth" {
		t.Fatalf("unexpected worktree name: %q", got)
	}
	if got := plan.Worktrees[0].Branch; got != "feature/auth" {
		t.Fatalf("unexpected branch: %q", got)
	}
	if got := plan.Worktrees[0].BranchMode; got != "existing" {
		t.Fatalf("unexpected branch mode: %q", got)
	}
}

func TestBuildCreatePlanWithExistingBranchAllowsCustomWorktreeName(t *testing.T) {
	root := initRepo(t)
	runGit(t, root, "branch", "feature/auth", "main")

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Names:          []string{"review-auth"},
		Branch:         "feature/auth",
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if got := plan.Worktrees[0].Name; got != "review-auth" {
		t.Fatalf("unexpected worktree name: %q", got)
	}
	if got := plan.Worktrees[0].Branch; got != "feature/auth" {
		t.Fatalf("unexpected branch: %q", got)
	}
}

func TestBuildCreatePlanRejectsMissingExistingBranch(t *testing.T) {
	root := initRepo(t)

	_, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Branch:         "feature/auth",
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err == nil || !strings.Contains(err.Error(), "branch does not exist") {
		t.Fatalf("expected missing branch error, got %v", err)
	}
}

func TestBuildCreatePlanRejectsBranchFlagWithBranchTemplate(t *testing.T) {
	root := initRepo(t)
	runGit(t, root, "branch", "feature/auth", "main")

	_, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Branch:         "feature/auth",
		BranchTemplate: "feature/{{ .Name }}",
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err == nil || !strings.Contains(err.Error(), "--branch cannot be used with --branch-template") {
		t.Fatalf("expected branch/template conflict, got %v", err)
	}
}

func TestBuildCreatePlanRejectsBranchFlagWithBase(t *testing.T) {
	root := initRepo(t)
	runGit(t, root, "branch", "feature/auth", "main")

	_, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Branch:         "feature/auth",
		BaseRef:        "main",
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err == nil || !strings.Contains(err.Error(), "--branch cannot be used with --base") {
		t.Fatalf("expected branch/base conflict, got %v", err)
	}
}

func TestBuildCreatePlanRejectsExistingBranchAlreadyInWorktree(t *testing.T) {
	root := initRepo(t)
	worktreePath := filepath.Join(filepath.Dir(root), "repo-feature-auth")
	runGit(t, root, "worktree", "add", worktreePath, "-b", "feature/auth")

	_, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Branch:         "feature/auth",
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err == nil || !strings.Contains(err.Error(), "branch already has a worktree") {
		t.Fatalf("expected existing worktree error, got %v", err)
	}
}

func TestBuildCreatePlanRejectsMultipleNames(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".arbor.yaml"), `
defaults:
  base_ref: main
  open_app: cursor
  worktree_template: ../{{ .Repo }}-{{ .Name }}
presets:
  fast:
    auto_run: false
`)

	_, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Names:          []string{"api", "web"},
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err == nil || !strings.Contains(err.Error(), "exactly one worktree name is supported") {
		t.Fatalf("expected single-worktree error, got %v", err)
	}
}

func TestBuildCreatePlanRejectsExistingPath(t *testing.T) {
	root := initRepo(t)
	existingPath := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-a")
	if err := os.MkdirAll(existingPath, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, ".arbor.yaml"), `
defaults:
  worktree_template: ../{{ .Repo }}-{{ .Name }}
`)

	_, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Names:          []string{"a"},
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err == nil || !strings.Contains(err.Error(), "worktree path already exists") {
		t.Fatalf("expected existing path error, got %v", err)
	}
}

func TestBuildCreatePlanInteractivePrompts(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".env"), "A=1")
	writeFile(t, filepath.Join(root, "package.json"), `{"scripts":{"dev":"vite","build":"vite build","test":"vitest"}}`)

	input := bytes.NewBufferString("symlink\ny\ny\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:   root,
		Names: []string{"feature-auth"},
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Commands[0].Approved != true {
		t.Fatalf("expected command approved")
	}
	if len(plan.Worktrees[0].Commands) != 2 {
		t.Fatalf("expected install and build commands, got %#v", plan.Worktrees[0].Commands)
	}
}

func TestBuildCreatePlanInteractivePromptsCarryEnvActionForward(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".env"), "A=1")
	writeFile(t, filepath.Join(root, ".env.prod"), "A=1")
	writeFile(t, filepath.Join(root, ".env.test"), "A=1")

	input := bytes.NewBufferString("symlink\n\n\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:   root,
		Names: []string{"feature-auth"},
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if len(plan.Worktrees[0].EnvActions) != 3 {
		t.Fatalf("expected 3 env actions, got %#v", plan.Worktrees[0].EnvActions)
	}
	for _, env := range plan.Worktrees[0].EnvActions {
		if env.Action != "symlink" {
			t.Fatalf("expected symlink for all env actions, got %#v", plan.Worktrees[0].EnvActions)
		}
	}
}

func TestBuildCreatePlanInteractivePromptsUpdateCarriedEnvDefault(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".env"), "A=1")
	writeFile(t, filepath.Join(root, ".env.prod"), "A=1")
	writeFile(t, filepath.Join(root, ".env.test"), "A=1")

	input := bytes.NewBufferString("copy\nskip\n\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:   root,
		Names: []string{"feature-auth"},
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if got := plan.Worktrees[0].EnvActions[0].Action; got != "copy" {
		t.Fatalf("expected first env action copy, got %q", got)
	}
	if got := plan.Worktrees[0].EnvActions[1].Action; got != "skip" {
		t.Fatalf("expected second env action skip, got %q", got)
	}
	if got := plan.Worktrees[0].EnvActions[2].Action; got != "skip" {
		t.Fatalf("expected third env action to inherit skip, got %q", got)
	}
}

func TestBuildCreatePlanInteractivePrefixPromptUsesSelectedPrefixWithoutConfig(t *testing.T) {
	root := initRepo(t)

	input := bytes.NewBufferString("auth\n1\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD: root,
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Name != "auth" {
		t.Fatalf("unexpected worktree name: %#v", plan.Worktrees[0])
	}
	if plan.Worktrees[0].Branch != "feat/auth" {
		t.Fatalf("unexpected branch: %#v", plan.Worktrees[0])
	}
	wantPath := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-feat-auth")
	if plan.Worktrees[0].Path != wantPath {
		t.Fatalf("unexpected path: %q", plan.Worktrees[0].Path)
	}
}

func TestBuildCreatePlanInteractivePrefixPromptSupportsCustomPrefix(t *testing.T) {
	root := initRepo(t)

	input := bytes.NewBufferString("auth\n4\nbugfix\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD: root,
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Branch != "bugfix/auth" {
		t.Fatalf("unexpected branch: %#v", plan.Worktrees[0])
	}
	wantPath := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-bugfix-auth")
	if plan.Worktrees[0].Path != wantPath {
		t.Fatalf("unexpected path: %q", plan.Worktrees[0].Path)
	}
}

func TestBuildCreatePlanInteractivePrefixPromptSupportsEmptyPrefix(t *testing.T) {
	root := initRepo(t)

	input := bytes.NewBufferString("auth\n5\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD: root,
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Branch != "auth" {
		t.Fatalf("unexpected branch: %#v", plan.Worktrees[0])
	}
	wantPath := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-auth")
	if plan.Worktrees[0].Path != wantPath {
		t.Fatalf("unexpected path: %q", plan.Worktrees[0].Path)
	}
}

func TestBuildCreatePlanInteractivePrefixPromptStillRunsWhenConfigExists(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".arbor.yaml"), "templates:\n  branch: release/{{ .Name }}\n")

	input := bytes.NewBufferString("auth\n2\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD: root,
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Branch != "fix/auth" {
		t.Fatalf("unexpected branch: %#v", plan.Worktrees[0])
	}
}

func TestFormatEnvPrompt(t *testing.T) {
	got := formatEnvPrompt("test-arb", ".env", "symlink")
	if got != "[test-arb] env .env action [symlink/copy/skip] (symlink)" {
		t.Fatalf("unexpected prompt: %q", got)
	}
}

func TestFormatCommandPrompt(t *testing.T) {
	got := formatCommandPrompt("test-arb", "pnpm install", "n")
	if got != "[test-arb] run pnpm install? [y/N] (n)" {
		t.Fatalf("unexpected prompt: %q", got)
	}
}

func TestPromptBranchPrefix(t *testing.T) {
	input := bytes.NewBufferString("2\n")
	got, err := promptBranchPrefix(input, bufio.NewReader(input))
	if err != nil {
		t.Fatalf("promptBranchPrefix returned error: %v", err)
	}
	if got != "fix" {
		t.Fatalf("unexpected prefix: %q", got)
	}
}

func TestBuildCreatePlanOpenAppFlagOverridesConfig(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".arbor.yaml"), `
defaults:
  open_app: cursor
`)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Names:          []string{"feature-auth"},
		OpenApp:        "code",
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.OpenApp != "code" {
		t.Fatalf("unexpected open app: %q", plan.OpenApp)
	}
	if !strings.Contains(RenderSummary(plan), "open app: code") {
		t.Fatalf("expected open app in summary: %s", RenderSummary(plan))
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	root := t.TempDir()
	runGit(t, root, "init")
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
