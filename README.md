# Arbor

Arbor is a CLI for creating Git worktrees and bootstrapping each worktree with env files and setup commands.

Use `README.md` as the end-user guide. See [DOCS.md](DOCS.md) for the preserved full reference.

## What Arbor Handles
- Detects likely env files from your repo and `.arbor.yaml`.
- Detects setup commands from project files such as `package.json`, `Makefile`, and `justfile`.
- Creates a worktree from a base branch or commit.
- Applies env-file actions with `symlink`, `copy`, or `skip`.
- Runs trusted setup commands automatically when a preset allows it.

## Run Arbor
If you are using Arbor from source in this repo:

```bash
go run ./cmd/arbor --help
```

If you want a local binary:

```bash
go build -o bin/arbor ./cmd/arbor
./bin/arbor --help
```

If you want the released binary through Homebrew:

```bash
brew install gcstools/tap/arbor
```

Install shell completions into your current shell setup:

```bash
arbor completion zsh
```

Print the raw completion script instead of installing it:

```bash
arbor completion zsh --stdout
```

## Quick Start
Initialize a starter config and validate it:

```bash
go run ./cmd/arbor init
go run ./cmd/arbor config validate
```

Preview what Arbor would do before creating anything:

```bash
go run ./cmd/arbor detect
go run ./cmd/arbor create feature-auth --plan
```

Create a worktree interactively:

```bash
go run ./cmd/arbor create feature-auth
```

Create a worktree without prompts:

```bash
go run ./cmd/arbor create feature-auth --non-interactive
```

Create a worktree and open it in an app after setup:

```bash
go run ./cmd/arbor create feature-auth --non-interactive --open-app cursor
```

## Common Commands
- `arbor init`: write a starter `.arbor.yaml` in the repo root.
- `arbor config validate`: validate the current config file before use.
- `arbor detect`: preview detected env files and setup commands without changing Git state.
- `arbor create [name...]`: create one or more worktrees and run setup actions.
- `arbor pull`: pull the main worktree when it has no local changes.
- `arbor version`: print the release version, commit, and build date.

## Minimal Config
Arbor reads `.arbor.yaml` from the repo root by default.

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

Practical defaults:
- Set `defaults.base_ref` to the branch you usually branch from.
- Use `env_files` to declare which repo env files should be copied or symlinked into each worktree.
- Use `commands` for repeatable setup steps such as dependency installation.
- Use `presets` to save common setups like `fast` or `full`.

## Typical Workflows
Use a preset for a repeatable setup:

```bash
go run ./cmd/arbor create feature-auth --preset fast
```

Validate a specific config file:

```bash
go run ./cmd/arbor --config .arbor.yaml config validate
```

Preview a non-interactive run before executing it:

```bash
go run ./cmd/arbor create feature-auth --preset fast --plan
```

## Notes
- In interactive mode, `arbor create` prompts for any unresolved choices before creating the worktree.
- In non-interactive mode, Arbor requires enough config and flags to resolve the run without prompts.
- If an env target already exists or a branch name already exists, Arbor reports the conflict and skips the unsafe action.

## More Docs
- Full reference: [DOCS.md](DOCS.md)
- Deploy for dev testing: [docs/deploy.md](docs/deploy.md)
- Deploy to Homebrew: [docs/deploy-brew.md](docs/deploy-brew.md)
- Release checklist: [docs/release.md](docs/release.md)
