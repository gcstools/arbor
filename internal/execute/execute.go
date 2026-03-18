package execute

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"arbor/internal/gitutil"
	"arbor/internal/model"
	"arbor/internal/planner"
)

type Runner struct {
	Stdout io.Writer
	Stderr io.Writer
}

var runOpenCommand = func(ctx context.Context, app, path string) error {
	cmd := exec.CommandContext(ctx, app, path)
	return cmd.Run()
}

func (r Runner) Apply(ctx context.Context, plan planner.CreatePlan) (model.ExecutionSummary, error) {
	git := gitutil.Runner{Dir: plan.RepoState.Root}
	summary := model.ExecutionSummary{
		RepoRoot: plan.RepoState.Root,
		Warnings: append([]string(nil), plan.Warnings...),
	}

	for _, worktree := range plan.Worktrees {
		result := model.WorktreeResult{
			Path:   worktree.Path,
			Branch: worktree.Branch,
		}

		exists, err := git.BranchExists(ctx, worktree.Branch)
		if err != nil {
			result.Error = err.Error()
			summary.HasFailure = true
			summary.Worktrees = append(summary.Worktrees, result)
			continue
		}
		if exists {
			result.Error = fmt.Sprintf("branch already exists %q", worktree.Branch)
			summary.HasFailure = true
			summary.Worktrees = append(summary.Worktrees, result)
			continue
		}
		if err := git.CreateBranch(ctx, worktree.Branch, worktree.BaseRef); err != nil {
			result.Error = err.Error()
			summary.HasFailure = true
			summary.Worktrees = append(summary.Worktrees, result)
			continue
		}
		if err := git.AddWorktree(ctx, worktree.Path, worktree.Branch); err != nil {
			result.Error = err.Error()
			summary.HasFailure = true
			summary.Worktrees = append(summary.Worktrees, result)
			continue
		}

		result.Created = true
		result.EnvResults = applyEnvActions(worktree)
		result.CommandResult = r.applyCommands(ctx, worktree)
		for _, env := range result.EnvResults {
			if env.Error != "" {
				result.Error = firstError(result.Error, env.Error)
				summary.HasFailure = true
			}
		}
		for _, command := range result.CommandResult {
			if command.Error != "" {
				result.Error = firstError(result.Error, command.Error)
				summary.HasFailure = true
			}
		}

		summary.Worktrees = append(summary.Worktrees, result)
	}

	summary.OpenApp = plan.OpenApp
	if plan.OpenApp != "" && len(summary.Worktrees) > 0 && isSuccessfulWorktree(summary.Worktrees[0]) {
		summary.OpenedPath = summary.Worktrees[0].Path
		if err := runOpenCommand(ctx, plan.OpenApp, summary.Worktrees[0].Path); err != nil {
			summary.OpenError = err.Error()
		}
	}

	return summary, nil
}

func applyEnvActions(worktree model.WorktreePlan) []model.EnvResult {
	results := make([]model.EnvResult, 0, len(worktree.EnvActions))
	for _, action := range worktree.EnvActions {
		result := model.EnvResult{ID: action.Candidate.ID, Action: action.Action}
		if action.Action == model.ActionSkip {
			results = append(results, result)
			continue
		}

		targetPath := filepath.Join(worktree.Path, action.Candidate.TargetPath)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}
		if _, err := os.Lstat(targetPath); err == nil {
			result.Error = fmt.Sprintf("target already exists %q", targetPath)
			results = append(results, result)
			continue
		}

		var err error
		switch action.Action {
		case model.ActionCopy:
			err = copyFile(action.Candidate.SourcePath, targetPath)
		case model.ActionSymlink:
			err = symlinkOrCopy(action.Candidate.SourcePath, targetPath)
		default:
			err = fmt.Errorf("unsupported env action %q", action.Action)
		}
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Applied = true
		}
		results = append(results, result)
	}
	return results
}

func (r Runner) applyCommands(ctx context.Context, worktree model.WorktreePlan) []model.CommandResult {
	results := make([]model.CommandResult, 0, len(worktree.Commands))
	for _, execution := range worktree.Commands {
		result := model.CommandResult{
			ID:      execution.Candidate.ID,
			Command: execution.Candidate.Command,
			Scope:   string(execution.Candidate.Scope),
		}
		if !execution.Approved {
			results = append(results, result)
			continue
		}
		exitCode, err := runShellCommand(ctx, worktree.Path, execution.Candidate.Command, r.Stdout, r.Stderr)
		result.Executed = true
		result.ExitCode = exitCode
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results
}

func runShellCommand(ctx context.Context, dir, command string, stdout io.Writer, stderr io.Writer) (int, error) {
	shell, flag := shellForPlatform()
	cmd := exec.CommandContext(ctx, shell, flag, command)
	cmd.Dir = dir
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	if stdout != nil {
		cmd.Stdout = io.MultiWriter(stdout, &outBuf)
	} else {
		cmd.Stdout = &outBuf
	}
	if stderr != nil {
		cmd.Stderr = io.MultiWriter(stderr, &errBuf)
	} else {
		cmd.Stderr = &errBuf
	}
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), fmt.Errorf("command %q failed: %s", command, string(bytes.TrimSpace(errBuf.Bytes())))
	}
	return -1, err
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func symlinkOrCopy(src, dst string) error {
	if err := os.Symlink(src, dst); err == nil {
		return nil
	} else if runtime.GOOS == "windows" {
		return copyFile(src, dst)
	} else {
		return err
	}
}

func shellForPlatform() (string, string) {
	if runtime.GOOS == "windows" {
		return "cmd", "/C"
	}
	return "sh", "-c"
}

func firstError(current, next string) string {
	if current != "" {
		return current
	}
	return next
}

func isSuccessfulWorktree(worktree model.WorktreeResult) bool {
	if !worktree.Created {
		return false
	}
	for _, env := range worktree.EnvResults {
		if env.Error != "" {
			return false
		}
	}
	for _, command := range worktree.CommandResult {
		if command.Error != "" {
			return false
		}
	}
	return true
}
