package cli

import (
	"errors"
	"strings"
	"testing"

	"arbor/internal/gitutil"
	"arbor/internal/model"
)

func TestFormatCreatePlanErrorSingleWorktreeMessage(t *testing.T) {
	err := formatCreatePlanError(errors.New("exactly one worktree name is supported"))
	if !strings.Contains(err.Error(), "can only create one worktree at a time") {
		t.Fatalf("unexpected message: %s", err.Error())
	}
}

func TestFormatCreatePlanErrorWrapsNotGitRepository(t *testing.T) {
	err := formatCreatePlanError(gitutil.ErrNotGitRepository)
	if !strings.Contains(err.Error(), "could not find a Git repository") {
		t.Fatalf("unexpected message: %s", err.Error())
	}
	if !errors.Is(err, gitutil.ErrNotGitRepository) {
		t.Fatalf("expected wrapped cause, got %v", err)
	}
}

func TestFormatCreatePlanErrorMissingBranch(t *testing.T) {
	err := formatCreatePlanError(errors.New(`branch does not exist "feature-auth"`))
	if !strings.Contains(err.Error(), "could not find the requested branch") {
		t.Fatalf("unexpected message: %s", err.Error())
	}
}

func TestFormatCreatePlanErrorBranchAlreadyInWorktree(t *testing.T) {
	err := formatCreatePlanError(errors.New(`branch already has a worktree "feature-auth" at "/tmp/repo-feature-auth"`))
	if !strings.Contains(err.Error(), "already attached to a worktree") {
		t.Fatalf("unexpected message: %s", err.Error())
	}
}

func TestRenderExecutionSummaryBranchExists(t *testing.T) {
	summary := model.ExecutionSummary{
		RepoRoot:   "/repo",
		HasFailure: true,
		Worktrees: []model.WorktreeResult{
			{
				Branch: "feature-auth",
				Path:   "/repo-feature-auth",
				Error:  `branch already exists "feature-auth"`,
			},
		},
	}

	got := renderExecutionSummary(summary)
	if !strings.Contains(got, "status: completed with errors") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, `branch "feature-auth" already exists`) {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestRenderExecutionSummaryCommandFailure(t *testing.T) {
	summary := model.ExecutionSummary{
		RepoRoot:   "/repo",
		HasFailure: true,
		Worktrees: []model.WorktreeResult{
			{
				Branch:  "feature-auth",
				Path:    "/repo-feature-auth",
				Created: true,
				Error:   `command "bun install" failed: missing lockfile`,
				CommandResult: []model.CommandResult{
					{
						ID:       "bootstrap",
						Command:  "bun install",
						Executed: true,
						ExitCode: 1,
						Error:    `command "bun install" failed: missing lockfile`,
					},
				},
			},
		},
	}

	got := renderExecutionSummary(summary)
	if !strings.Contains(got, "created the worktree, but one or more setup steps failed") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, `setup command "bun install" exited with status 1`) {
		t.Fatalf("unexpected output: %s", got)
	}
}

func TestRenderExecutionSummaryOpenFailure(t *testing.T) {
	summary := model.ExecutionSummary{
		RepoRoot:   "/repo",
		OpenApp:    "cursor",
		OpenedPath: "/repo-feature-auth",
		OpenError:  "executable file not found",
		Worktrees: []model.WorktreeResult{
			{
				Branch:  "feature-auth",
				Path:    "/repo-feature-auth",
				Created: true,
			},
		},
	}

	got := renderExecutionSummary(summary)
	if !strings.Contains(got, "open status: worktree setup finished, but opening the folder failed") {
		t.Fatalf("unexpected output: %s", got)
	}
	if !strings.Contains(got, "verify that the app executable exists") {
		t.Fatalf("unexpected output: %s", got)
	}
}
