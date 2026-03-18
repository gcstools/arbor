# Arbor Implementation Plan

<!-- AGENT INSTRUCTIONS: Mark each checkbox [x] when the task is completed. Do not proceed to dependent tasks until prerequisites are checked off. Before marking a task complete, verify it satisfies any related acceptance criteria in the Verification section. COMMIT after completing each task using the `git-commit` skill to format the message. -->

## Overview
Build Arbor as a Cobra-based Go CLI for single Git worktree creation, env-file setup, detected or configured post-create commands, and YAML-driven project presets.

## Prerequisites
- [ ] Initialize Go module, CLI package layout, test harness, fixture repos, cross-platform CI targets.
- [ ] Add Cobra dependency; define target config path, command names, package boundaries, completion strategy, and minimum supported Git version.

## Phase 1: CLI Foundation

### Task 1.1: Migrate command suite to Cobra
- [x] Implementation: Replace custom CLI parser with Cobra root command, `create`, `init`, `detect`, `config` subcommands, persistent flags, help, usage, exit-code mapping.
- [x] Unit Test: Add Cobra smoke tests covering command registration, help output, inherited flags, invalid subcommand handling.

### Task 1.2: Add Git repo discovery layer
- [x] Implementation: Add repo root detection, Git binary execution wrapper, current branch or commit lookup, worktree listing.
- [x] Unit Test: Add tests for repo detection, non-repo failure, Git command error mapping, worktree parsing.

### Task 1.3: Define core domain models
- [x] Implementation: Add types for worktree plan, env-file candidate, command candidate, preset, template output, execution summary.
- [x] Unit Test: Add serialization and validation tests for core models and summary states.

### Task 1.4: Add Cobra completion and command wiring hooks
- [x] Implementation: Add shell completion command or generation path, command constructors, shared pre-run config loading hooks, output stream injection for tests.
- [x] Unit Test: Add tests for completion command presence, command constructor wiring, and shared pre-run behavior.

## Phase 2: Config and Template Engine

### Task 2.1: Implement YAML config loader
- [x] Implementation: Add repo-local YAML parsing, schema validation, defaults, config discovery, effective-config assembly.
- [x] Unit Test: Add tests for valid config load, missing config, malformed YAML, unknown fields, precedence defaults.

### Task 2.2: Implement presets and trust rules
- [x] Implementation: Add preset resolution, trusted command policy, default env actions, and command scopes.
- [x] Unit Test: Add tests for preset merge order, trust gating, and invalid preset references.

### Task 2.3: Implement naming templates
- [x] Implementation: Add branch and worktree path templates, variable interpolation, collision precheck inputs, deterministic expansion.
- [x] Unit Test: Add tests for template rendering, missing variables, invalid output, duplicate generated names.

## Phase 3: Detection Engine

### Task 3.1: Detect env files
- [x] Implementation: Add built-in env-file heuristics, config overrides, candidate ranking, target-path mapping.
- [x] Unit Test: Add tests for common env patterns, override inclusion or exclusion, duplicate suppression, target mapping.

### Task 3.2: Detect runnable commands
- [x] Implementation: Add command extraction from `package.json`, `Makefile`, `justfile`, plus config-defined commands.
- [x] Unit Test: Add tests for each source parser, label extraction, parse failures, merged command ordering.

### Task 3.3: Build detect preview output
- [x] Implementation: Add Cobra `detect` handler and preview formatter for env candidates, command candidates, source attribution, effective defaults.
- [x] Unit Test: Add tests for detect command output shape, empty-state handling, config-plus-detection merge visibility.

## Phase 4: Interactive Planning Flow

### Task 4.1: Build create-plan resolver
- [x] Implementation: Add input resolution for base ref, worktree count, branch names, target paths, preset, non-interactive requirements.
- [x] Unit Test: Add tests for single planning, explicit flag precedence, missing required input failure, duplicate path rejection.

### Task 4.2: Build interactive selection prompts
- [x] Implementation: Add prompt flow for env actions, command selection, confirmation gates, overwrite policy, preset review; integrate with Cobra flag state and non-interactive mode.
- [x] Unit Test: Add prompt-state tests for default selections, skip flows, overwrite prompts, trusted auto-run bypass, flag-driven prompt suppression.

### Task 4.3: Add execution plan summary
- [x] Implementation: Add preflight summary for branches, paths, env actions, command scopes, destructive-risk warnings.
- [x] Unit Test: Add tests for summary completeness, warning rendering, non-interactive summary generation.

## Phase 5: Worktree and Setup Execution

### Task 5.1: Execute branch and worktree creation
- [x] Implementation: Add branch creation, `git worktree add` orchestration, preflight conflict checks, partial-failure handling.
- [x] Unit Test: Add tests for branch collisions, existing path conflicts, and Git failure rollback policy.

### Task 5.2: Execute env-file actions
- [x] Implementation: Add `symlink`, `copy`, `skip` handlers, overwrite behavior, Windows-safe symlink fallback, result tracking.
- [x] Unit Test: Add tests for symlink success, copy success, fallback behavior, existing destination handling, permission failures.

### Task 5.3: Execute post-create commands
- [x] Implementation: Add worktree command runners, trusted preset auto-run, confirmation enforcement, stdout or stderr capture.
- [x] Unit Test: Add tests for trusted auto-run, rejected command skip, and command failure reporting.

## Phase 6: User-Facing Setup Commands

### Task 6.1: Implement `arbor init`
- [x] Implementation: Add starter YAML generation with commented defaults, presets scaffold, template examples, safe file creation.
- [x] Unit Test: Add tests for init output, existing file refusal, overwrite flag behavior, generated config validity.

### Task 6.2: Implement `arbor config`
- [x] Implementation: Add Cobra `config` subcommands for inspect and validate flows, effective-config rendering, config error diagnostics.
- [x] Unit Test: Add tests for config validation output, merged view rendering, missing-config messaging, subcommand routing.

### Task 6.3: Harden non-interactive automation
- [x] Implementation: Add flag-complete Cobra `create` path, machine-readable errors, deterministic exit codes, prompt suppression rules.
- [x] Unit Test: Add tests for fully non-interactive create, unresolved choice failure, preset-only automation, Cobra exit-code mapping.

## Phase 7: Verification and Release Readiness

### Task 7.1: Add end-to-end fixture coverage
- [x] Implementation: Add fixture repos covering Node, Make, Just, env variants, and config presets.
- [x] Unit Test: Add end-to-end tests for single create, detect preview, init plus config roundtrip.

### Task 7.2: Add cross-platform validation
- [x] Implementation: Add CI matrix for macOS, Linux, Windows; verify symlink or copy behavior and Git integration.
- [x] Unit Test: Add OS-conditional tests for path handling, symlink fallback, shell command execution assumptions.

### Task 7.3: Final docs and release checks
- [x] Implementation: Add README usage, config reference, examples, failure-mode docs, release checklist.
- [x] Unit Test: Add doc-snippet validation or smoke checks for sample commands and generated config examples.

## Verification
- [x] All unit tests pass.
- [x] Cobra command tree exposes `create`, `init`, `detect`, `config`, and completion support with consistent inherited flags.
- [x] `arbor create` handles branch, path, env-file, and command conflicts without silent failure.
- [x] Trusted preset commands auto-run only when config marks them trusted.
- [x] `arbor init` and `arbor config` produce valid YAML-driven workflows.
- [x] `arbor detect` shows env and command candidates without mutating Git state.
- [x] End-to-end tests pass for single worktree flows.
- [x] Windows symlink limitations handled via clear fallback or failure behavior.

## Unresolved Questions
- [ ] Confirm minimum supported Git version and whether Arbor should enforce it at startup.
- [ ] Confirm exact default YAML filename and path within the repo.
- [ ] Confirm whether command execution should use platform shell defaults or explicit shell selection per config.
