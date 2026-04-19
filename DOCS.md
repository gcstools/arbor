# Arbor Docs

Arbor is a Go CLI for creating a Git worktree and bootstrapping it with env files and setup commands.

Use [README.md](README.md) as the end-user guide. This document keeps the full reference plus source/developer-oriented instructions.

## Install From Homebrew

For normal usage, install Arbor with Homebrew:

```bash
brew install gcstools/tap/arbor
```

Then inspect the CLI:

```bash
arbor --help
arbor create --help
```

## Build and Run From Source

This project is a Go CLI. If you are working in this repository, use the commands below.

Install dependencies:

```bash
go mod tidy
```

Run directly from the repo:

```bash
go run ./cmd/arbor --help
go run ./cmd/arbor config --help
```

Build a local binary:

```bash
mkdir -p bin
go build -o ./bin/arbor ./cmd/arbor
```

Run the built binary:

```bash
./bin/arbor --help
./bin/arbor version
./bin/arbor completion bash
```

Run tests:

```bash
go test ./...
```

If your environment blocks the default Go build cache, use:

```bash
GOCACHE=/tmp/arbor-go-build-cache go test ./...
```

Cross-compile for manual testing:

```bash
mkdir -p bin
GOOS=darwin GOARCH=arm64 go build -o ./bin/arbor-darwin-arm64 ./cmd/arbor
GOOS=linux GOARCH=amd64 go build -o ./bin/arbor-linux-amd64 ./cmd/arbor
GOOS=windows GOARCH=amd64 go build -o ./bin/arbor-windows-amd64.exe ./cmd/arbor
```

## Features

- Detect env files from common project patterns and repo config.
- Detect setup commands from `package.json`, `Makefile`, `justfile`, and Arbor config.
- Create one worktree with one branch per invocation.
- Create a worktree from an existing local branch.
- Apply env-file actions with `symlink`, `copy`, or `skip`.
- Run trusted preset commands automatically or prompt for command approval.
- Install or print shell completion scripts.

## Commands

### `arbor init`

Write a starter `.arbor.yaml` in the current repo.

```bash
arbor init
arbor init --force
```

`--force` overwrites an existing config file.

### `arbor config`

Print the resolved config file:

```bash
arbor config
```

Validate the current config:

```bash
arbor config validate
```

Validate a specific config file:

```bash
arbor --config .arbor.yaml config validate
```

### `arbor detect`

Preview detected env files and setup commands without changing Git state:

```bash
arbor detect
```

The output includes:

- repo root
- detected env files and their target paths
- detected setup commands and their source
- warnings when Arbor finds ambiguous or incomplete data

### `arbor create`

Preview a worktree plan:

```bash
arbor create feature-auth --plan
```

Create and execute the worktree setup:

```bash
arbor create feature-auth --non-interactive
```

Create from an existing local branch:

```bash
arbor create review-auth --branch feature-auth --non-interactive
```

Create with a preset:

```bash
arbor create feature-auth --preset fast
```

Create from a specific base branch or commit:

```bash
arbor create feature-auth --base main --plan
```

Override branch naming and path templates:

```bash
arbor create feature-auth \
  --branch-template '{{ .Prefix }}/{{ .Name }}' \
  --path-template '../{{ .Repo }}-{{ .Prefix }}-{{ .Name }}' \
  --plan
```

Open the created worktree in a specific app:

```bash
arbor create feature-auth --non-interactive --open-app cursor
```

Notes:

- Interactive mode prompts only for unresolved choices before creating the worktree.
- Config-defined `env_files` suppress env-action prompts during `create`.
- If `presets.default.commands` exists and `--preset` is omitted, Arbor implicitly uses `default` and skips command approval prompts for those selected commands.
- Non-interactive mode requires enough config and flags to resolve the run without prompts.
- `--branch` reuses an existing local branch and cannot be combined with `--base` or `--branch-template`.

### `arbor pull`

Pull the main worktree when it has no local changes:

```bash
arbor pull
```

If the main worktree is dirty, Arbor skips the pull and prints the main worktree path.

### `arbor completion`

Install shell completions:

```bash
arbor completion zsh
arbor completion bash
arbor completion fish
arbor completion powershell
```

Print the raw completion script:

```bash
arbor completion zsh --stdout
```

Write the script to a custom location:

```bash
arbor completion fish --path ~/.config/fish/completions/arbor.fish
```

Disable descriptions in completion entries:

```bash
arbor completion zsh --no-descriptions
```

When run in an interactive terminal without `--stdout`, Arbor installs the completion file and updates shell startup files where needed.

### `arbor version`

Print the Arbor version:

```bash
arbor version
```

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

templates:
  branch: "{{ .Prefix }}/{{ .Name }}"
  worktree: "../{{ .Repo }}-{{ .Prefix }}-{{ .Name }}"
```

### Config Sections

#### `defaults`

Defines fallback behavior Arbor uses when the command line or a preset does not override it.

- `base_ref`: branch or commit used as the default source for new branches.
- `env_action`: default env action for env files. Valid values are `symlink`, `copy`, and `skip`.
- `command_scope`: default execution scope for commands. Current value is `per_worktree`.
- `trusted_auto_run`: allows trusted preset commands to execute automatically.
- `open_app`: executable Arbor runs after setup to open the created worktree folder.

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
- Uses `symlink` unless config changes it.
- Suppresses env-action prompts during `create` because config already defines the env-file plan shape.
- Emits a warning if `source_path` does not exist on disk.

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
- Lets presets select the command as part of a repeatable setup plan.
- Executes whenever the final plan marks it approved.

Node projects also get built-in package detection from `package.json`. Arbor offers package-manager-aware setup steps like `pnpm install` and, when present, `pnpm build` instead of prompting for every script in the file.

#### `presets`

Presets are saved setup profiles. They do not define new env files or new commands by themselves. Instead, they select from the `env_files` and `commands` you already defined and bundle those choices under a reusable name.

Use a preset when you want Arbor to answer the same setup questions the same way every time for a certain workflow, such as `fast`, `full`, `backend`, or `review`.

- preset name: the key under `presets`, such as `fast` or `full`
- `description`: short explanation of the preset
- `env_selection`: list of `env_files` IDs to select
- `commands`: list of command IDs to select and approve in the plan
- `auto_run`: requests automatic execution for selected commands, subject to trust rules

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

- Selects the env file with ID `env`.
- Selects and approves the command with ID `bootstrap` in the plan.
- If you omit `--preset` and define `presets.default`, Arbor applies the same rule implicitly.
- `trusted_auto_run` and preset `auto_run` remain available config signals, but Arbor no longer needs an interactive confirmation step when preset command selection already resolves the choice.

Example usage:

```bash
arbor create feature-auth --preset fast
```

Think of the relationship like this:

- `env_files` says which env-file options exist.
- `commands` says which setup commands exist.
- `presets` says which of those options should be selected together for a named workflow.

No-prompt example:

```yaml
env_files:
  - id: env
    source_path: .env.shared
    target_path: .env
    default_action: copy

commands:
  - id: bootstrap
    label: Bootstrap
    command: pnpm install

presets:
  default:
    commands: [bootstrap]
```

With that config, `arbor create feature-auth` does not ask env-action questions and does not ask whether to run `bootstrap`; the plan already resolves both.

#### `templates`

Provides reusable templates for naming branches and worktree paths.

- `branch`: template used to generate branch names
- `worktree`: template used to generate worktree paths

Template variables currently used by Arbor:

- `.Prefix`: normalized branch prefix, empty when none is present.
- `.Name`: normalized slug after the prefix, or the full slug when no prefix exists.
- `.Repo`: repo directory name
- `.Base`: selected base ref
- `.Branch`: resolved branch name when rendering worktree paths

Example:

```yaml
templates:
  branch: "{{ .Prefix }}/{{ .Name }}"
  worktree: "../{{ .Repo }}-{{ .Prefix }}-{{ .Name }}"
```

What it does:

- Splits the branch namespace into `.Prefix` and `.Name` so branch templates can describe both the prefix and the slug.
- Uses `.Prefix` to build nested branch names like `feature/auth`.
- Places the worktree next to the repo as `../myrepo-feature-auth`.

Template variables:

- `.Prefix`: normalized branch prefix, empty when none is present.
- `.Name`: normalized slug after the prefix, or the full slug when no prefix exists.
- `.Repo`: repo directory name.
- `.Base`: selected base ref.
- `.Branch`: resolved branch name when rendering worktree paths.

When `.Prefix` is empty, Arbor removes one adjacent `/` or `-` separator for bare `{{ .Prefix }}` placeholders so templates can omit conditionals.

For `--branch` plans, Arbor keeps the attached-branch path behavior stable. If you attach to an existing branch, path template rendering may flatten the branch into a single display slug instead of preserving a split `.Prefix`/`.Name` pair.

### Config Precedence

Arbor resolves config in this order:

1. CLI flags
2. Selected preset
3. Config defaults
4. Built-in detection

### Notes

- Config-defined env files override duplicate auto-detected env targets.
- If no config file exists, Arbor still works using detection and command-line flags.
- In interactive mode, Arbor asks for unresolved input and applies the resolved choices to the worktree plan before execution.
- `arbor init` writes a starter `.arbor.yaml` you can edit for your repo.

## Failure Modes

- Existing branch name: Arbor reports the branch conflict and skips creation.
- Existing target env path: Arbor reports the file conflict and leaves the target untouched.
- Invalid config: Arbor stops before execution and returns the YAML validation error.
- Command failure: Arbor records the failing command and exit code in the execution summary.

## Developer Docs

- [README.md](/Users/simon/work/github/arbor/README.md): end-user install and usage guide
- [docs/deploy.md](/Users/simon/work/github/arbor/docs/deploy.md): local dev deployment and manual testing
- [docs/deploy-brew.md](/Users/simon/work/github/arbor/docs/deploy-brew.md): Homebrew deployment
- [docs/release.md](/Users/simon/work/github/arbor/docs/release.md): release checklist
