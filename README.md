# Arbor

Arbor is a CLI for creating Git worktrees and setting env files, reusable setup commands, and optional app launching.

`README.md` is the end-user guide. For full config reference, source builds, and maintainer/developer docs, see [DOCS.md](/Users/simon/work/github/arbor/DOCS.md).

## Install

Install Arbor with Homebrew:

```bash
brew install gcstools/tap/arbor
```

Confirm the install:

```bash
arbor version
arbor --help
```

Running from source during development:

```bash
go run ./cmd/arbor --help
```

## What Arbor Does

Arbor helps you spin up a new worktree without repeating the same setup steps by hand.

- Detects likely env files from your repo and `.arbor.yaml`.
- Detects setup commands from project files such as `package.json`, `Makefile`, and `justfile`.
- Creates a worktree from a new branch or attaches to an existing local branch.
- Applies env-file actions with `symlink`, `copy`, or `skip`.
- Runs selected setup commands after the worktree is created.
- Opens the finished worktree in your editor or app if configured.

## Typical Flow

1. Run `arbor init` once in a repo to create a starter `.arbor.yaml`.
2. Adjust the config so Arbor knows which env files, commands, and presets you want.
3. Run `arbor detect` to preview what Arbor sees automatically.
4. Run `arbor create <name>` to preview or create a worktree.

## Configure Arbor

Create a starter config:

```bash
arbor init
```

Validate the config after editing it:

```bash
arbor config validate
```

Print the fully resolved config:

```bash
arbor config
```

Arbor reads `.arbor.yaml` from the repo root by default. A practical starter config looks like this:

```yaml
defaults:
  base_ref: main
  env_action: symlink
  trusted_auto_run: true
  open_app: cursor

env_files:
  - id: env
    source_path: .env
    target_path: .env

commands:
  - id: bootstrap
    label: Bootstrap
    command: pnpm install
    scope: per_worktree
    trusted: true

presets:
  fast:
    env_selection: [env]
    commands: [bootstrap]
    auto_run: true
```

Use config to answer the choices Arbor would otherwise need to ask interactively:

- `defaults` sets fallback behavior such as base branch, env-file action, and app opening.
- `env_files` declares which files Arbor can copy or symlink into the new worktree.
- `commands` defines reusable setup steps Arbor can run after creating the worktree.
- `presets` bundles env files and commands into named workflows such as `fast` or `full`.

## Preview What Arbor Detects

Use `detect` to see which env files and setup commands Arbor can infer from the current repo:

```bash
arbor detect
```

This is useful when you want to check auto-detection before writing config or before creating a worktree.

## Create Worktrees

### Preview the plan first

Show exactly what Arbor would do without changing Git state:

```bash
arbor create feature-auth --plan
```

### Create a new worktree interactively

```bash
arbor create feature-auth
```

In interactive mode, Arbor prompts for any unresolved choices, such as which detected env files or commands to use.

### Create a new worktree without prompts

```bash
arbor create feature-auth --non-interactive
```

Use this when your config and flags already provide everything Arbor needs.

### Create from an existing local branch

```bash
arbor create review-auth --branch feature-auth --non-interactive
```

Use `--branch` when the branch already exists locally and you want the worktree attached to it. `--branch` cannot be combined with `--base` or `--branch-template`.

### Create from a specific base

```bash
arbor create feature-auth --base main --plan
```

Use `--base` to choose the branch or commit Arbor should branch from when creating a new branch.

### Use a preset

```bash
arbor create feature-auth --preset fast
```

Presets preselect env files and commands so common setups stay consistent.

### Override naming and path templates

```bash
arbor create feature-auth \
  --branch-template 'feature/{{ .Name }}' \
  --path-template '../{{ .Repo }}-{{ .Name }}' \
  --plan
```

Use templates when you want command-line control over branch naming or the destination folder for a worktree.

### Open the worktree in an app after setup

```bash
arbor create feature-auth --non-interactive --open-app cursor
```

Arbor runs the open command after env-file actions and approved setup commands finish.

## How Arbor Handles Setup

### Env files

Arbor can prepare files such as `.env` inside the new worktree.

- `symlink` shares the source file with the worktree.
- `copy` duplicates the file into the worktree.
- `skip` leaves that target path alone.

If the target path already exists, Arbor reports the conflict and leaves the existing file untouched.

### Commands

Arbor can run setup commands after worktree creation, such as dependency installation or a bootstrap script.

- Commands come from your config and from supported project files that Arbor can detect.
- Presets can preselect commands.
- Trusted commands can auto-run when both the preset and config allow it.
- If trust or config does not resolve the choice, Arbor asks before running the command.

### Interactive vs non-interactive runs

- Interactive mode is best when you want Arbor to guide you through unresolved choices.
- Non-interactive mode is best for repeatable flows in a fully configured repo.
- `--plan` is useful in either mode when you want a dry run first.

## Other Commands

### Pull the main worktree

```bash
arbor pull
```

This pulls the main worktree only when it has no local changes. If the main worktree is dirty, Arbor skips the pull and tells you why.

### Install shell completions

Install completions into your shell setup:

```bash
arbor completion zsh
arbor completion bash
arbor completion fish
arbor completion powershell
```

Print the raw completion script instead of installing it:

```bash
arbor completion zsh --stdout
```

Use a custom path when you want to manage the script yourself:

```bash
arbor completion fish --path ~/.config/fish/completions/arbor.fish
```

Disable inline command descriptions if you prefer shorter completion menus:

```bash
arbor completion zsh --no-descriptions
```

### Print the version

```bash
arbor version
```

## Common Workflows

Preview a reusable setup before executing it:

```bash
arbor create feature-auth --preset fast --plan
```

Validate a non-default config file:

```bash
arbor --config path/to/.arbor.yaml config validate
```

Create a worktree, run trusted setup, and open it:

```bash
arbor create feature-auth --preset fast --non-interactive --open-app cursor
```

## When Arbor Stops for Safety

- If a branch already exists where Arbor expected to create a new one, it reports the conflict.
- If an env-file target already exists in the worktree, Arbor leaves it untouched.
- If the config file is invalid, Arbor stops before creating anything.
- If a setup command fails, Arbor reports the failing command in the execution summary.

## More Information

See [DOCS.md](/Users/simon/work/github/arbor/DOCS.md) for:

- full config reference
- advanced examples and precedence details
- build and run instructions for working from source
- project maintenance and developer-oriented docs
