# Arbor

Arbor is a Go CLI for creating a Git worktree and bootstrapping it with env files and setup commands.

## Features
- Detect env files from common project patterns and repo config.
- Detect setup commands from `package.json`, `Makefile`, `justfile`, and Arbor config.
- Create one worktree with one branch per invocation.
- Create a worktree from an existing local branch.
- Apply env-file actions with `symlink`, `copy`, or `skip`.
- Run trusted preset commands automatically or prompt for command approval.

## Quick Start
Install dependencies and inspect the CLI:

```bash
go mod tidy
go run ./cmd/arbor --help
```

Create a starter config:

```bash
go run ./cmd/arbor init
go run ./cmd/arbor config validate
```

Preview a worktree plan:

```bash
go run ./cmd/arbor create feature-auth --plan
```

Interactive `create` prompts for a branch prefix before resolving the branch name and worktree path.

Create and execute the worktree setup:

```bash
go run ./cmd/arbor create feature-auth --non-interactive
```

Create a worktree from an existing local branch:

```bash
go run ./cmd/arbor create --branch feature-auth --non-interactive
```

Create, run setup, and open the resulting worktree in a specific app:

```bash
go run ./cmd/arbor create feature-auth --non-interactive --open-app cursor
```

## Core Commands
- `arbor detect`: preview env files and commands without changing Git state.
- `arbor create`: plan or execute worktree creation and setup.
- `arbor init`: write a starter `.arbor.yaml`.
- `arbor config`: print the effective config.
- `arbor config validate`: validate the current config file.

## Config
Arbor uses `.arbor.yaml` at the repo root by default.

Example:

```yaml
defaults:
  base_ref: main
  env_action: symlink
  command_scope: per_worktree
  trusted_auto_run: true
  open_app: cursor
  worktree_template: ../{{ .Repo }}-{{ .Name }}

env_files:
  - id: env
    source_path: .env
    target_path: .env

commands:
  - id: bootstrap
    label: Bootstrap
    command: echo bootstrap
    scope: per_worktree
    trusted: true

presets:
  fast:
    env_selection: [env]
    commands: [bootstrap]
    auto_run: true
```

### Config Sections

#### `defaults`
Defines fallback behavior Arbor uses when the command line or a preset does not override it.

- `base_ref`: branch or commit used as the default source for new branches.
- `env_action`: default action for env files. Valid values are `symlink`, `copy`, and `skip`.
- `command_scope`: default execution scope for commands. Current value is `per_worktree`.
- `trusted_auto_run`: allows trusted preset commands to execute automatically.
- `open_app`: executable Arbor runs after setup to open the created worktree folder.
- `worktree_template`: default template for generated worktree paths.

Example:

```yaml
defaults:
  base_ref: main
  env_action: symlink
  command_scope: per_worktree
  trusted_auto_run: true
  open_app: cursor
  worktree_template: ../{{ .Repo }}-{{ .Name }}
```

When `open_app` is set, Arbor waits for env actions and approved commands to finish, then runs `<open_app> <worktree-path>` for the created worktree.

#### `env_files`
Declares env files that Arbor should offer during worktree setup. These entries can override or replace automatically detected env-file candidates.

- `id`: stable identifier used by presets.
- `label`: human-readable name shown in prompts and output.
- `source_path`: source file in the main repo.
- `target_path`: destination path inside the new worktree.
- `default_action`: default env action for this file.

Example:

```yaml
env_files:
  - id: env
    label: Shared env
    source_path: .env.shared
    target_path: .env
    default_action: symlink
```

What it does:
- Offers `.env.shared` as a candidate during `arbor create`.
- Places it at `.env` inside the new worktree.
- Uses `symlink` unless the user or preset changes it.

#### `commands`
Defines runnable setup commands that Arbor can show or execute after creating a worktree.

- `id`: stable identifier used by presets.
- `label`: display name for prompts and summaries.
- `command`: shell command Arbor runs.
- `scope`: `per_worktree`.
- `trusted`: marks a command as eligible for preset auto-run when `trusted_auto_run` is enabled.

Example:

```yaml
commands:
  - id: bootstrap
    label: Bootstrap
    command: pnpm install
    scope: per_worktree
    trusted: true
```

What it does:
- Shows `Bootstrap` as a selectable setup step.
- Runs `pnpm install` inside the new worktree.
- Allows auto-run only when a preset selects it and trust rules permit it.

Node projects also get built-in package detection from `package.json`. Arbor now offers package-manager-aware setup steps like `pnpm install` and, when present, `pnpm build` instead of prompting for every script in the file.

#### `presets`
Presets are saved setup profiles. They do not define new env files or new commands by themselves. Instead, they select from the `env_files` and `commands` you already defined and bundle those choices under a reusable name.

Use a preset when you want Arbor to answer the same setup questions the same way every time for a certain workflow, such as `fast`, `full`, `backend`, or `review`.

- preset name: the key under `presets`, such as `fast` or `full`.
- `description`: short explanation of the preset.
- `env_selection`: list of `env_files` IDs to preselect.
- `commands`: list of command IDs to preselect.
- `auto_run`: requests automatic execution for selected commands, subject to trust rules.

Example:

```yaml
presets:
  fast:
    description: Fast local setup
    env_selection: [env]
    commands: [bootstrap]
    auto_run: true
```

What it does:
- Preselects the env file with ID `env`.
- Preselects the command with ID `bootstrap`.
- If `bootstrap` is marked `trusted: true` and `defaults.trusted_auto_run` is also `true`, Arbor runs it automatically.
- If trust rules do not allow auto-run, Arbor still keeps the command selected and asks for confirmation.

Example usage:

```bash
arbor create feature-auth --preset fast
```

In that example, Arbor uses the `fast` profile instead of asking you to choose everything from scratch.

Think of the relationship like this:
- `env_files` says which env-file options exist.
- `commands` says which setup commands exist.
- `presets` says which of those options should be selected together for a named workflow.

#### `templates`
Provides reusable templates for naming branches and worktree paths.

- `branch`: template used to generate branch names.
- `worktree`: template used to generate worktree paths.

Template variables currently used by Arbor:
- `.Name`: worktree input name.
- `.Repo`: repo directory name.
- `.Base`: selected base ref.
- `.Branch`: resolved branch name when rendering worktree paths.

Example:

```yaml
templates:
  branch: feature/{{ .Name }}
  worktree: ../{{ .Repo }}-{{ .Name }}
```

What it does:
- Turns a worktree named `auth` into branch `feature/auth`.
- Places the worktree next to the repo as `../myrepo-auth`.

### Config Precedence
Arbor resolves config in this order:

1. CLI flags
2. Selected preset
3. Config defaults
4. Built-in detection

### Notes
- Config-defined env files override duplicate auto-detected env targets.
- If no config file exists, Arbor still works using detection and command-line flags.
- In interactive mode, Arbor asks for an optional branch prefix and applies it to both the branch name (`<prefix>/<name>`) and the default worktree folder name.
- `--branch` reuses an existing local branch and cannot be combined with `--base` or `--branch-template`.
- `arbor init` writes a starter `.arbor.yaml` you can edit for your repo.

## Examples
Preview detection only:

```bash
go run ./cmd/arbor detect
```

Validate a specific config:

```bash
go run ./cmd/arbor --config .arbor.yaml config validate
```

Create with a preset:

```bash
go run ./cmd/arbor create feature-auth --preset fast --non-interactive
```

Create a review worktree from an existing branch:

```bash
go run ./cmd/arbor create review-auth --branch feature-auth --non-interactive
```

## Failure Modes
- Existing branch name: Arbor reports the branch conflict and skips creation.
- Existing target env path: Arbor reports the file conflict and leaves the target untouched.
- Invalid config: Arbor stops before execution and returns the YAML validation error.
- Command failure: Arbor records the failing command and exit code in the execution summary.

## Docs
- [Deploy for dev testing](docs/deploy.md)
- [Release checklist](docs/release.md)
