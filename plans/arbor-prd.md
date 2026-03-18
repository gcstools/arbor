# Arbor PRD

## Summary
Arbor is a Go CLI for creating and bootstrapping Git worktrees with minimal manual setup. Its v1 goal is to detect environment files and runnable setup commands, present clear choices to the user, and automate worktree creation, branch creation, env-file linking or copying, and post-create setup.

Primary audience: developers and teams that use Git worktrees regularly and need repeatable local setup for one worktree at a time.

Core product promise: `git worktree add`, plus guided environment bootstrapping and project-aware setup automation.

CLI foundation: Arbor uses Cobra for command structure, shared flags, help output, and shell completion support.

## Product Goals
- Reduce friction after creating a worktree by surfacing relevant env files and setup scripts automatically.
- Support both ad hoc use and team-standardized workflows via project config.
- Keep single-worktree creation fast and repeatable.
- Create a branch per worktree by default.
- Work across macOS, Linux, and Windows.

Success criteria for v1:
- User can create one worktree from a single command flow.
- User can choose which env files to symlink or copy into each worktree.
- User can review and run detected or configured setup commands after creation.
- Teams can codify defaults, presets, and templates in a YAML config.
- Trusted presets may auto-run commands; non-preset or inferred commands still require confirmation.

## V1 User Experience
Primary mode: interactive command suite.

Expected flow for `arbor create`:
1. Detect repository root and current Git state.
2. Resolve target worktree name and branch name.
3. Create one branch from a selected base branch or commit.
4. Detect candidate env files from repo rules and config.
5. Present env-file actions per file: `symlink`, `copy`, `skip`.
6. Detect candidate setup commands from known sources and config.
7. Present commands to run in the new worktree or via preset.
8. Require confirmation unless a trusted preset explicitly marks commands as auto-run.
9. Execute setup and show a worktree summary.

Non-interactive support:
- Flags may prefill decisions and suppress prompts where all required inputs are provided.
- Config presets may fully define env and script behavior for automation use.

Out of scope for v1:
- Full IDE or editor integration.
- Background daemons or watchers.
- Remote secret management.
- Arbitrary hook lifecycle beyond post-create setup.

## Key Requirements
### Worktree creation
- Support creating a single worktree per invocation.
- Support creating a new branch for that worktree.
- Allow branch names to be user-supplied or generated from templates.
- Validate branch and worktree name collisions before execution.

### Env file detection and handling
- Detect env-like files from common patterns and config overrides.
- Show source path, target path, and proposed action.
- Default behavior: symlink.
- Per file, user may choose `symlink`, `copy`, or `skip`.
- Must handle existing destination files safely with overwrite or skip prompts, or explicit non-interactive policy.
- Windows support must use behavior compatible with platform symlink constraints; when symlink creation is unavailable, Arbor should fall back to copy if configured or clearly prompt or fail.

### Script or command detection
- Detect commands from known files in v1, at minimum from common developer entrypoints such as `package.json`, `Makefile`, and `justfile`.
- Merge detected commands with commands defined in YAML config.
- Present command label, source, exact command, and execution scope.
- Allow commands to run in the created worktree.
- Inferred commands require confirmation. Preset-defined trusted commands may auto-run.

### Config
Standard config: YAML project config committed in repo.

Config must support:
- Detection overrides for env files.
- Script definitions and labels.
- Presets bundling env selections and post-create commands.
- Templates for branch names and worktree names or paths.
- Trust metadata for commands eligible for auto-run.

Config precedence:
1. Explicit CLI flags
2. Selected preset
3. Project config defaults
4. Built-in detection heuristics

## Public Interfaces
Primary commands:
- `arbor create`
- `arbor init`
- `arbor detect`
- `arbor config`

Expected behavior:
- `arbor create`: guided or flag-driven worktree creation and bootstrapping.
- `arbor init`: create starter YAML config in the repo.
- `arbor detect`: preview detected env files and commands without creating worktrees.
- `arbor config`: inspect or validate effective config and presets.

CLI behavior expectations:
- Cobra-backed command tree with consistent help, usage, and inherited root flags.
- Shared persistent flags for config path, non-interactive behavior, and future global output controls.
- Shell completion generation treated as a supported CLI capability, even if its install docs ship later.

Initial config concepts:
- `defaults`
- `env_files`
- `commands`
- `presets`
- `templates`
Initial user-visible result model:
- Worktree: branch created, path created, env actions taken, commands run or skipped, failures.

## Acceptance Scenarios
- Create one worktree interactively from current repo; choose two env files to symlink, skip one, run one detected setup command.
- Use project config preset to auto-select env files and auto-run trusted commands.
- Run in preview mode via `arbor detect` and confirm the detected env files and commands without changing Git state.
- Handle existing target env files without silent overwrite.
- Reject duplicate branch names or existing worktree paths before partial execution.
- On Windows, handle symlink permission limitations predictably and surface fallback behavior clearly.
- In non-interactive mode, fail fast if required choices are missing and no preset or default resolves them.

## Quality and Validation
- Cross-platform integration coverage for macOS, Linux, and Windows.
- CLI tests cover Cobra command registration, help rendering, flag inheritance, and invalid command handling.
- Parser and validation tests for YAML config, presets, templates, and precedence rules.
- Detection tests for supported command sources and env-file patterns.
- End-to-end tests for single worktree creation.
- Failure-path tests for branch conflicts, file conflicts, invalid commands, and Git command failures.

## Assumptions and Defaults
- Repo-local YAML config is the source of truth in v1; no separate global user config.
- Interactive flow is primary; flags and presets support automation.
- One branch per worktree is the default model.
- Symlink is the default env-file action.
- Trusted preset commands may auto-run; ad hoc detected commands require approval.
- v1 favors a Cobra-backed command suite over a single wizard entrypoint.
