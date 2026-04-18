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

func TestNormalizeCreateNameKnownPrefix(t *testing.T) {
	name, branch, err := normalizeCreateName("feat shr-123 new admin page")
	if err != nil {
		t.Fatalf("normalizeCreateName returned error: %v", err)
	}
	if name != "feat-shr-123-new-admin-page" {
		t.Fatalf("unexpected worktree name: %q", name)
	}
	if branch != "feat/shr-123-new-admin-page" {
		t.Fatalf("unexpected branch: %q", branch)
	}
}

func TestNormalizeCreateNameFeaturePrefix(t *testing.T) {
	name, branch, err := normalizeCreateName("feature api v2")
	if err != nil {
		t.Fatalf("normalizeCreateName returned error: %v", err)
	}
	if name != "feature-api-v2" {
		t.Fatalf("unexpected worktree name: %q", name)
	}
	if branch != "feature/api-v2" {
		t.Fatalf("unexpected branch: %q", branch)
	}
}

func TestNormalizeCreateNameFixPrefix(t *testing.T) {
	name, branch, err := normalizeCreateName("fix login bug")
	if err != nil {
		t.Fatalf("normalizeCreateName returned error: %v", err)
	}
	if name != "fix-login-bug" {
		t.Fatalf("unexpected worktree name: %q", name)
	}
	if branch != "fix/login-bug" {
		t.Fatalf("unexpected branch: %q", branch)
	}
}

func TestNormalizeCreateNameChorePrefix(t *testing.T) {
	name, branch, err := normalizeCreateName("chore cleanup scripts")
	if err != nil {
		t.Fatalf("normalizeCreateName returned error: %v", err)
	}
	if name != "chore-cleanup-scripts" {
		t.Fatalf("unexpected worktree name: %q", name)
	}
	if branch != "chore/cleanup-scripts" {
		t.Fatalf("unexpected branch: %q", branch)
	}
}

func TestNormalizeCreateNameMixedPunctuation(t *testing.T) {
	name, branch, err := normalizeCreateName("feat api: retry?")
	if err != nil {
		t.Fatalf("normalizeCreateName returned error: %v", err)
	}
	if name != "feat-api-retry" {
		t.Fatalf("unexpected worktree name: %q", name)
	}
	if branch != "feat/api-retry" {
		t.Fatalf("unexpected branch: %q", branch)
	}
}

func TestNormalizeCreateNamePreservesSafeSymbols(t *testing.T) {
	name, branch, err := normalizeCreateName("docs c++ #1 & api")
	if err != nil {
		t.Fatalf("normalizeCreateName returned error: %v", err)
	}
	if name != "docs-c++-#1-&-api" {
		t.Fatalf("unexpected worktree name: %q", name)
	}
	if branch != "docs-c++-#1-&-api" {
		t.Fatalf("unexpected branch: %q", branch)
	}
}

func TestNormalizeCreateNameUnknownPrefix(t *testing.T) {
	name, branch, err := normalizeCreateName("docs new page")
	if err != nil {
		t.Fatalf("normalizeCreateName returned error: %v", err)
	}
	if name != "docs-new-page" {
		t.Fatalf("unexpected worktree name: %q", name)
	}
	if branch != "docs-new-page" {
		t.Fatalf("unexpected branch: %q", branch)
	}
}

func TestNormalizeCreateNamePrefixWithoutRemainder(t *testing.T) {
	name, branch, err := normalizeCreateName("feat")
	if err != nil {
		t.Fatalf("normalizeCreateName returned error: %v", err)
	}
	if name != "feat" {
		t.Fatalf("unexpected worktree name: %q", name)
	}
	if branch != "feat" {
		t.Fatalf("unexpected branch: %q", branch)
	}
}

func TestNormalizeCreateNameRejectsEmptyInput(t *testing.T) {
	_, _, err := normalizeCreateName("   ")
	if err == nil || !strings.Contains(err.Error(), "worktree name is required") {
		t.Fatalf("expected missing name error, got %v", err)
	}
}

func TestNormalizeCreateNameRejectsEmptyString(t *testing.T) {
	_, _, err := normalizeCreateName("")
	if err == nil || !strings.Contains(err.Error(), "worktree name is required") {
		t.Fatalf("expected missing name error, got %v", err)
	}
}

func TestNormalizeCreateNameRejectsPunctuationOnlyInput(t *testing.T) {
	_, _, err := normalizeCreateName(".")
	if err == nil || !strings.Contains(err.Error(), "worktree name is required") {
		t.Fatalf("expected missing name error, got %v", err)
	}
}

func TestNormalizeCreateNamePrefixOnlyWhenRemainderSanitizesEmpty(t *testing.T) {
	name, branch, err := normalizeCreateName("feat .")
	if err != nil {
		t.Fatalf("normalizeCreateName returned error: %v", err)
	}
	if name != "feat" {
		t.Fatalf("unexpected worktree name: %q", name)
	}
	if branch != "feat" {
		t.Fatalf("unexpected branch: %q", branch)
	}
}

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
	if got := plan.Worktrees[0].Name; got != "feature-auth" {
		t.Fatalf("unexpected worktree name: %q", got)
	}
	if got := plan.Worktrees[0].Branch; got != "feature/auth" {
		t.Fatalf("unexpected branch: %q", got)
	}
	if got := plan.Worktrees[0].BranchMode; got != "existing" {
		t.Fatalf("unexpected branch mode: %q", got)
	}
	wantPath := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-feature-auth")
	if got := plan.Worktrees[0].Path; got != wantPath {
		t.Fatalf("unexpected worktree path: %q", got)
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

func TestBuildCreatePlanWithExistingBranchDerivesSafeNameForPathOnly(t *testing.T) {
	root := initRepo(t)
	runGit(t, root, "branch", "feature/api.v2_auth", "main")

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Branch:         "feature/api.v2_auth",
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if got := plan.Worktrees[0].Branch; got != "feature/api.v2_auth" {
		t.Fatalf("unexpected branch: %q", got)
	}
	if got := plan.Worktrees[0].Name; got != "feature-api-v2-auth" {
		t.Fatalf("unexpected derived worktree name: %q", got)
	}
	wantPath := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-feature-api-v2-auth")
	if got := plan.Worktrees[0].Path; got != wantPath {
		t.Fatalf("unexpected worktree path: %q", got)
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

func TestBuildCreatePlanBranchTemplateUsesNormalizedNameButKeepsTemplateControl(t *testing.T) {
	root := initRepo(t)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Names:          []string{"Feat", "API", "Retry"},
		BranchTemplate: "release/{{ .Name }}--{{ .Repo }}",
		PathTemplate:   "../{{ .Branch }}",
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if got := plan.Worktrees[0].Name; got != "feat-api-retry" {
		t.Fatalf("unexpected worktree name: %q", got)
	}
	if got := plan.Worktrees[0].Branch; got != "release/feat-api-retry--"+filepath.Base(root) {
		t.Fatalf("unexpected branch: %q", got)
	}
	wantPath := filepath.Join(filepath.Dir(root), "release", "feat-api-retry--"+filepath.Base(root))
	if got := plan.Worktrees[0].Path; got != wantPath {
		t.Fatalf("unexpected worktree path: %q", got)
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

func TestBuildCreatePlanJoinsMultipleNamesAndNormalizes(t *testing.T) {
	root := initRepo(t)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Names:          []string{"feat", "shr-123", "new", "admin", "page"},
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if got := plan.Worktrees[0].Name; got != "feat-shr-123-new-admin-page" {
		t.Fatalf("unexpected worktree name: %q", got)
	}
	if got := plan.Worktrees[0].Branch; got != "feat/shr-123-new-admin-page" {
		t.Fatalf("unexpected branch: %q", got)
	}
}

func TestBuildCreatePlanJoinsDuplicatePositionalTokens(t *testing.T) {
	root := initRepo(t)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Names:          []string{"feat", "feat", "auth"},
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if got := plan.Worktrees[0].Name; got != "feat-feat-auth" {
		t.Fatalf("unexpected worktree name: %q", got)
	}
	if got := plan.Worktrees[0].Branch; got != "feat/feat-auth" {
		t.Fatalf("unexpected branch: %q", got)
	}
}

func TestBuildCreatePlanUnknownPrefixUsesPlainSlug(t *testing.T) {
	root := initRepo(t)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Names:          []string{"admin", "redesign"},
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if got := plan.Worktrees[0].Name; got != "admin-redesign" {
		t.Fatalf("unexpected worktree name: %q", got)
	}
	if got := plan.Worktrees[0].Branch; got != "admin-redesign" {
		t.Fatalf("unexpected branch: %q", got)
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

func TestBuildCreatePlanInteractivePrefixPromptUsesKnownPrefixWithoutConfig(t *testing.T) {
	root := initRepo(t)

	input := bytes.NewBufferString("new admin page\n2\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD: root,
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Name != "feature-new-admin-page" {
		t.Fatalf("unexpected worktree name: %#v", plan.Worktrees[0])
	}
	if plan.Worktrees[0].Branch != "feature/new-admin-page" {
		t.Fatalf("unexpected branch: %#v", plan.Worktrees[0])
	}
	wantPath := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-feature-new-admin-page")
	if plan.Worktrees[0].Path != wantPath {
		t.Fatalf("unexpected path: %q", plan.Worktrees[0].Path)
	}
}

func TestBuildCreatePlanInteractivePrefixPromptSupportsCustomPrefix(t *testing.T) {
	root := initRepo(t)

	input := bytes.NewBufferString("new admin page\n5\nbug fix\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD: root,
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Name != "bug-fix-new-admin-page" {
		t.Fatalf("unexpected worktree name: %#v", plan.Worktrees[0])
	}
	if plan.Worktrees[0].Branch != "bug-fix/new-admin-page" {
		t.Fatalf("unexpected branch: %#v", plan.Worktrees[0])
	}
	wantPath := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-bug-fix-new-admin-page")
	if plan.Worktrees[0].Path != wantPath {
		t.Fatalf("unexpected path: %q", plan.Worktrees[0].Path)
	}
}

func TestBuildCreatePlanInteractivePrefixPromptRejectsInvalidCustomPrefixInTextMode(t *testing.T) {
	root := initRepo(t)

	input := bytes.NewBufferString("new admin page\n5\n!!!\nbug fix\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD: root,
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Name != "bug-fix-new-admin-page" {
		t.Fatalf("unexpected worktree name: %#v", plan.Worktrees[0])
	}
	if plan.Worktrees[0].Branch != "bug-fix/new-admin-page" {
		t.Fatalf("unexpected branch: %#v", plan.Worktrees[0])
	}
	wantPath := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-bug-fix-new-admin-page")
	if plan.Worktrees[0].Path != wantPath {
		t.Fatalf("unexpected path: %q", plan.Worktrees[0].Path)
	}
}

func TestBuildCreatePlanInteractivePrefixPromptSupportsEmptyPrefix(t *testing.T) {
	root := initRepo(t)

	input := bytes.NewBufferString("new admin page\n6\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD: root,
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Name != "new-admin-page" {
		t.Fatalf("unexpected worktree name: %#v", plan.Worktrees[0])
	}
	if plan.Worktrees[0].Branch != "new-admin-page" {
		t.Fatalf("unexpected branch: %#v", plan.Worktrees[0])
	}
	wantPath := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-new-admin-page")
	if plan.Worktrees[0].Path != wantPath {
		t.Fatalf("unexpected path: %q", plan.Worktrees[0].Path)
	}
}

func TestBuildCreatePlanInteractiveEmptyPrefixUsesCliNormalization(t *testing.T) {
	root := initRepo(t)

	input := bytes.NewBufferString("feat new admin page\n6\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD: root,
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Name != "feat-new-admin-page" {
		t.Fatalf("unexpected worktree name: %#v", plan.Worktrees[0])
	}
	if plan.Worktrees[0].Branch != "feat/new-admin-page" {
		t.Fatalf("unexpected branch: %#v", plan.Worktrees[0])
	}
	wantPath := filepath.Join(filepath.Dir(root), filepath.Base(root)+"-feat-new-admin-page")
	if plan.Worktrees[0].Path != wantPath {
		t.Fatalf("unexpected path: %q", plan.Worktrees[0].Path)
	}
}

func TestBuildCreatePlanInteractivePrefixPromptStillRunsWhenConfigExists(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".arbor.yaml"), "templates:\n  branch: release/{{ .Name }}\n")

	input := bytes.NewBufferString("new admin page\n3\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD: root,
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Name != "fix-new-admin-page" {
		t.Fatalf("unexpected worktree name: %#v", plan.Worktrees[0])
	}
	if plan.Worktrees[0].Branch != "release/fix-new-admin-page" {
		t.Fatalf("unexpected branch: %#v", plan.Worktrees[0])
	}
}

func TestBuildCreatePlanInteractiveBranchTemplateSkipsPrefixPrompt(t *testing.T) {
	root := initRepo(t)

	input := bytes.NewBufferString("New API & worker\n")
	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		BranchTemplate: "release/{{ .Name }}",
	}, input, ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}
	if plan.Worktrees[0].Name != "new-api-&-worker" {
		t.Fatalf("unexpected worktree name: %#v", plan.Worktrees[0])
	}
	if plan.Worktrees[0].Branch != "release/new-api-&-worker" {
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
	if got != "feature" {
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

func TestBuildCreatePlanLoadsFallbackDefaultConfigName(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, "arbor.yml"), `
defaults:
  base_ref: main
  open_app: cursor
commands:
  - id: bootstrap
    label: Bootstrap deps
    command: bun install
presets:
  default:
    commands: [bootstrap]
`)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:   root,
		Names: []string{"feature-auth"},
	}, bytes.NewBufferString(""), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}

	if got := plan.OpenApp; got != "cursor" {
		t.Fatalf("expected fallback config open app, got %q", got)
	}
	if got := plan.Worktrees[0].Preset; got != "default" {
		t.Fatalf("expected fallback config preset, got %q", got)
	}
	if len(plan.Worktrees[0].Commands) != 1 || !plan.Worktrees[0].Commands[0].Approved {
		t.Fatalf("expected approved command from fallback config, got %#v", plan.Worktrees[0].Commands)
	}
}

func TestRenderSummaryUsesPlannedBaseRef(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".arbor.yaml"), `
defaults:
  base_ref: main
`)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:   root,
		Names: []string{"feature-auth"},
	}, bytes.NewBufferString(""), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}

	summary := RenderSummary(plan)
	if !strings.Contains(summary, "base ref: main") {
		t.Fatalf("expected planned base ref in summary: %s", summary)
	}
}

func TestBuildCreatePlanUsesImplicitDefaultPresetCommandsWithoutPrompt(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".arbor.yaml"), `
commands:
  - id: install
    label: Install deps
    command: pnpm install
  - id: build
    label: Build app
    command: pnpm build
presets:
  default:
    commands: [install]
`)
	// writeFile(t, filepath.Join(root, "package.json"), `{"name":"demo"}`)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:   root,
		Names: []string{"feature-auth"},
	}, bytes.NewBufferString(""), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}

	if got := plan.Worktrees[0].Preset; got != "default" {
		t.Fatalf("expected implicit default preset, got %q", got)
	}
	if len(plan.Worktrees[0].Commands) != 2 {
		t.Fatalf("expected 2 commands, got %#v", plan.Worktrees[0].Commands)
	}
	if !plan.Worktrees[0].Commands[0].Approved {
		t.Fatalf("expected selected default preset command to be approved: %#v", plan.Worktrees[0].Commands)
	}
	if plan.Worktrees[0].Commands[1].Approved {
		t.Fatalf("expected unselected command to remain unapproved: %#v", plan.Worktrees[0].Commands)
	}
}

func TestBuildCreatePlanUsesImplicitDefaultPresetWhenPresent(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".arbor.yaml"), `
presets:
  default:
    description: Default local setup
`)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Names:          []string{"feature-auth"},
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}

	if got := plan.Worktrees[0].Preset; got != "default" {
		t.Fatalf("expected implicit default preset, got %q", got)
	}
}

func TestBuildCreatePlanSkipsEnvPromptsWhenConfigEnvFilesExist(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".env"), "A=1")
	writeFile(t, filepath.Join(root, ".arbor.yaml"), `
env_files:
  - id: env
    label: Primary env
    source_path: .env
    target_path: .env
    default_action: copy
`)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:   root,
		Names: []string{"feature-auth"},
	}, bytes.NewBufferString(""), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}

	if len(plan.Worktrees[0].EnvActions) != 1 {
		t.Fatalf("expected 1 env action, got %#v", plan.Worktrees[0].EnvActions)
	}
	if got := plan.Worktrees[0].EnvActions[0].Action; got != "copy" {
		t.Fatalf("expected config env action to remain copy, got %q", got)
	}
}

func TestBuildCreatePlanStillPromptsCommandsWhenNoDefaultPresetCommands(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".arbor.yaml"), `
commands:
  - id: install
    label: Install deps
    command: pnpm install
presets:
  default:
    description: Default local setup
`)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:   root,
		Names: []string{"feature-auth"},
	}, bytes.NewBufferString("y\n"), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}

	if len(plan.Worktrees[0].Commands) != 1 {
		t.Fatalf("expected 1 command, got %#v", plan.Worktrees[0].Commands)
	}
	if !plan.Worktrees[0].Commands[0].Approved {
		t.Fatalf("expected command prompt approval to apply, got %#v", plan.Worktrees[0].Commands)
	}
}

func TestExplicitPresetOverridesImplicitDefaultPreset(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".arbor.yaml"), `
commands:
  - id: install
    label: Install deps
    command: pnpm install
  - id: build
    label: Build app
    command: pnpm build
presets:
  default:
    commands: [install]
  fast:
    commands: [build]
`)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:    root,
		Names:  []string{"feature-auth"},
		Preset: "fast",
	}, bytes.NewBufferString(""), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}

	if got := plan.Worktrees[0].Preset; got != "fast" {
		t.Fatalf("expected explicit preset to win, got %q", got)
	}
	if plan.Worktrees[0].Commands[0].Approved {
		t.Fatalf("expected default preset command to stay unapproved: %#v", plan.Worktrees[0].Commands)
	}
	if !plan.Worktrees[0].Commands[1].Approved {
		t.Fatalf("expected explicit preset command to be approved: %#v", plan.Worktrees[0].Commands)
	}
}

func TestBuildCreatePlanImplicitDefaultTrustedAutoRunApprovesTrustedAndPromptsUntrusted(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".arbor.yaml"), `
defaults:
  trusted_auto_run: true
commands:
  - id: trusted
    label: Trusted bootstrap
    command: pnpm install
    trusted: true
  - id: untrusted
    label: Untrusted build
    command: pnpm build
presets:
  default:
    commands: [trusted, untrusted]
    auto_run: true
`)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:   root,
		Names: []string{"feature-auth"},
	}, bytes.NewBufferString(""), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}

	if len(plan.Worktrees[0].Commands) != 2 {
		t.Fatalf("expected 2 commands, got %#v", plan.Worktrees[0].Commands)
	}
	if !plan.Worktrees[0].Commands[0].Approved {
		t.Fatalf("expected trusted command approved, got %#v", plan.Worktrees[0].Commands)
	}
	if !plan.Worktrees[0].Commands[1].Approved {
		t.Fatalf("expected selected untrusted command approved without prompt, got %#v", plan.Worktrees[0].Commands)
	}
}

func TestBuildCreatePlanWarnsWhenConfiguredEnvFileMissing(t *testing.T) {
	root := initRepo(t)
	writeFile(t, filepath.Join(root, ".arbor.yaml"), `
env_files:
  - id: env
    label: Primary env
    source_path: .env
    target_path: .env
    default_action: copy
`)

	plan, err := BuildCreatePlan(context.Background(), Inputs{
		CWD:            root,
		Names:          []string{"feature-auth"},
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}

	if len(plan.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %#v", plan.Warnings)
	}
	if got := plan.Warnings[0]; !strings.Contains(got, `env_files[env]: source path ".env" not found`) {
		t.Fatalf("unexpected warning: %q", got)
	}
	if got := RenderSummary(plan); !strings.Contains(got, `env_files[env]: source path ".env" not found`) {
		t.Fatalf("expected warning in summary: %s", got)
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
