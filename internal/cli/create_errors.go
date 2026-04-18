package cli

import (
	"errors"
	"fmt"
	"strings"

	"arbor/internal/gitutil"
	"arbor/internal/model"
)

type userError struct {
	message string
	cause   error
}

func (e userError) Error() string {
	return e.message
}

func (e userError) Unwrap() error {
	return e.cause
}

func formatCreatePlanError(err error) error {
	if err == nil {
		return nil
	}

	message := "Arbor could not prepare this worktree.\n" +
		"Details: " + err.Error()

	switch {
	case errors.Is(err, gitutil.ErrNotGitRepository):
		message = "Arbor could not find a Git repository from the current directory.\n" +
			"Next step: run this command inside a Git repo or pass a path inside one."
	case strings.Contains(err.Error(), "worktree name is required in non-interactive mode"):
		message = "Arbor could not choose a worktree name in non-interactive mode.\n" +
			"Details: no worktree name was provided and prompts are disabled.\n" +
			"Next step: pass a name like `arbor create feature-auth --non-interactive`."
	case strings.Contains(err.Error(), "worktree name is required"):
		message = "Arbor needs a worktree name before it can continue.\n" +
			"Next step: enter a name when prompted or pass one on the command line."
	case strings.Contains(err.Error(), "worktree path already exists"):
		message = "The target worktree path is already in use.\n" +
			"Details: " + err.Error() + ".\n" +
			"Next step: remove the existing directory or choose a different worktree name or path template."
	case strings.Contains(err.Error(), "--branch cannot be used with --branch-template"):
		message = "Arbor could not combine the requested branch options.\n" +
			"Details: `--branch` reuses an existing branch, so `--branch-template` is not allowed.\n" +
			"Next step: remove `--branch-template` or omit `--branch` to create a new branch."
	case strings.Contains(err.Error(), "--branch cannot be used with --base"):
		message = "Arbor could not combine the requested branch options.\n" +
			"Details: `--branch` reuses an existing branch, so `--base` is not allowed.\n" +
			"Next step: remove `--base` or omit `--branch` to create a new branch from a base ref."
	case strings.Contains(err.Error(), "branch does not exist"):
		message = "Arbor could not find the requested branch.\n" +
			"Details: " + err.Error() + ".\n" +
			"Next step: create or fetch that local branch first, then rerun the command."
	case strings.Contains(err.Error(), "branch already has a worktree"):
		message = "The requested branch is already attached to a worktree.\n" +
			"Details: " + err.Error() + ".\n" +
			"Next step: use a different branch or remove the existing worktree first."
	case strings.Contains(err.Error(), "branch template for"):
		message = "Arbor could not build the branch name for this worktree.\n" +
			"Details: " + err.Error() + ".\n" +
			"Next step: check your branch template values or run `arbor config validate`."
	case strings.Contains(err.Error(), "path template for"):
		message = "Arbor could not build the worktree path for this run.\n" +
			"Details: " + err.Error() + ".\n" +
			"Next step: check your path template values or run `arbor config validate`."
	case strings.Contains(err.Error(), "preset ") && strings.Contains(err.Error(), " not found"):
		message = "Arbor could not find the requested preset.\n" +
			"Details: " + err.Error() + ".\n" +
			"Next step: use a preset that exists in your Arbor config or run `arbor config` to inspect it."
	case strings.Contains(err.Error(), "parse config:"):
		message = "Arbor could not read the config file for this run.\n" +
			"Details: " + err.Error() + ".\n" +
			"Next step: fix the config syntax and rerun `arbor config validate`."
	case strings.Contains(err.Error(), "git "):
		message = "Arbor could not inspect the Git repository before creating the worktree.\n" +
			"Details: " + err.Error() + ".\n" +
			"Next step: make sure Git is installed and the repository is in a healthy state."
	}

	return userError{message: message, cause: err}
}

func renderExecutionSummary(summary model.ExecutionSummary) string {
	var lines []string

	lines = append(lines, fmt.Sprintf("repo: %s", summary.RepoRoot))
	if summary.HasFailure {
		lines = append(lines, "status: completed with errors")
	} else {
		lines = append(lines, "status: success")
	}
	if len(summary.Warnings) > 0 {
		lines = append(lines, "")
		lines = append(lines, "warnings:")
		for _, warning := range summary.Warnings {
			lines = append(lines, "  - "+warning)
		}
	}

	for _, worktree := range summary.Worktrees {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("worktree %s", worktree.Branch))
		lines = append(lines, fmt.Sprintf("  path: %s", worktree.Path))
		lines = append(lines, fmt.Sprintf("  branch: %s", worktree.Branch))
		lines = append(lines, fmt.Sprintf("  created: %t", worktree.Created))
		if worktree.Error != "" {
			lines = append(lines, "  summary: "+formatWorktreeErrorSummary(worktree))
		}
		for _, env := range worktree.EnvResults {
			if env.Error == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("  env %s: %s", env.ID, formatEnvResultError(env, worktree)))
		}
		for _, command := range worktree.CommandResult {
			if command.Error == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf("  command %s: %s", command.ID, formatCommandResultError(command)))
		}
	}

	if summary.OpenApp != "" {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("open app: %s", summary.OpenApp))
		if summary.OpenedPath != "" {
			lines = append(lines, fmt.Sprintf("opened path: %s", summary.OpenedPath))
		}
		if summary.OpenError != "" {
			lines = append(lines, "open status: worktree setup finished, but opening the folder failed")
			lines = append(lines, fmt.Sprintf("open details: %s", summary.OpenError))
			lines = append(lines, "next step: verify that the app executable exists and can open local folders.")
		}
	}

	return strings.Join(lines, "\n")
}

func formatWorktreeErrorSummary(worktree model.WorktreeResult) string {
	switch {
	case strings.Contains(worktree.Error, "branch already exists"):
		return fmt.Sprintf("Arbor did not create this worktree because the branch %q already exists. Choose a different name or branch.", worktree.Branch)
	case strings.Contains(worktree.Error, "already checked out"):
		return fmt.Sprintf("Arbor could not add the worktree because branch %q is already attached to another worktree. Details: %s", worktree.Branch, worktree.Error)
	case strings.Contains(worktree.Error, "git branch "):
		return fmt.Sprintf("Arbor could not create the Git branch %q before adding the worktree. Details: %s", worktree.Branch, worktree.Error)
	case strings.Contains(worktree.Error, "git worktree add "):
		return fmt.Sprintf("Arbor could not add the worktree for branch %q at %q. Details: %s", worktree.Branch, worktree.Path, worktree.Error)
	default:
		if worktree.Created {
			return "Arbor created the worktree, but one or more setup steps failed."
		}
		return "Arbor could not finish creating this worktree. Details: " + worktree.Error
	}
}

func formatEnvResultError(result model.EnvResult, worktree model.WorktreeResult) string {
	switch {
	case strings.Contains(result.Error, "target already exists"):
		return fmt.Sprintf("Arbor skipped this env file because the target path already exists in %q. Details: %s", worktree.Path, result.Error)
	default:
		return "Arbor could not apply this env file action. Details: " + result.Error
	}
}

func formatCommandResultError(result model.CommandResult) string {
	switch {
	case result.ExitCode > 0:
		return fmt.Sprintf("setup command %q exited with status %d. Details: %s", result.Command, result.ExitCode, result.Error)
	default:
		return fmt.Sprintf("setup command %q failed. Details: %s", result.Command, result.Error)
	}
}
