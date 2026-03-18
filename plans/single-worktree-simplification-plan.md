# Single Worktree Simplification Implementation Plan

<!-- AGENT INSTRUCTIONS: Mark each checkbox [x] when the task is completed. Do not proceed to dependent tasks until prerequisites are checked off. Before marking a task complete, verify it satisfies any related acceptance criteria in the Verification section. COMMIT after completing each task using the `git-commit` skill to format the message. -->

## Overview
Remove config-template default `branch_template`. Remove batch and multi-worktree flows. Keep Arbor focused on one worktree, one branch, one execution path per invocation. Require explicit branch prefix selection instead of deriving `feature/...` from starter config.

## Prerequisites
- [ ] Confirm desired non-interactive API for prefix input: reuse `--branch-template`, add `--branch-prefix`, or require full branch name arg

## Phase 1: Config And Model Cleanup

### Task 1.1: Drop Batch Schema
- [ ] Implementation: Remove `Batches`, `Batch`, `ExpandBatch`, `Inputs.Batch`, `--batch`, batch-only planner paths from [internal/config/config.go](/Users/simon/work/github/arbor/internal/config/config.go), [internal/planner/planner.go](/Users/simon/work/github/arbor/internal/planner/planner.go), [internal/cli/create.go](/Users/simon/work/github/arbor/internal/cli/create.go)
- [ ] Unit Test: Replace batch coverage in [internal/config/config_test.go](/Users/simon/work/github/arbor/internal/config/config_test.go) and [internal/planner/planner_test.go](/Users/simon/work/github/arbor/internal/planner/planner_test.go) with rejection/absence tests for removed batch fields and flags

### Task 1.2: Remove Per-Batch Command Scope
- [ ] Implementation: Delete `CommandScopePerBatch` and any branching that treats commands as batch-scoped in [internal/model/types.go](/Users/simon/work/github/arbor/internal/model/types.go), config validation, planner defaults, and executor result rendering
- [ ] Unit Test: Add/adjust tests in [internal/config/config_test.go](/Users/simon/work/github/arbor/internal/config/config_test.go) and [internal/execute/execute_test.go](/Users/simon/work/github/arbor/internal/execute/execute_test.go) to verify only `per_worktree` commands are accepted and executed

### Task 1.3: Fix Starter Config Defaults
- [ ] Implementation: Remove `defaults.branch_template` from `StarterConfig` while keeping path templating intact in [internal/config/config.go](/Users/simon/work/github/arbor/internal/config/config.go)
- [ ] Unit Test: Add starter-config assertions in [internal/config/config_test.go](/Users/simon/work/github/arbor/internal/config/config_test.go) verifying generated config omits `defaults.branch_template`

## Phase 2: Planner And CLI Single-Worktree Flow

### Task 2.1: Enforce Single Worktree Input
- [ ] Implementation: Change [internal/planner/planner.go](/Users/simon/work/github/arbor/internal/planner/planner.go) and [internal/cli/create.go](/Users/simon/work/github/arbor/internal/cli/create.go) so `create` accepts exactly one worktree target, rejects multiple names, removes dedupe/list loops, and simplifies summary/output wording to singular semantics
- [ ] Unit Test: Update [internal/planner/planner_test.go](/Users/simon/work/github/arbor/internal/planner/planner_test.go) and [internal/cli/app_test.go](/Users/simon/work/github/arbor/internal/cli/app_test.go) to cover single-name success and multi-name rejection

### Task 2.2: Always Prompt For Prefix In Interactive Create
- [ ] Implementation: Refactor prefix resolution in [internal/planner/planner.go](/Users/simon/work/github/arbor/internal/planner/planner.go) so interactive create always asks for prefix before branch/path resolution, regardless of config presence; stop using config default branch template as implicit prefix source
- [ ] Unit Test: Update prefix prompt tests in [internal/planner/planner_test.go](/Users/simon/work/github/arbor/internal/planner/planner_test.go) to assert prompt always runs interactively and config no longer suppresses it

### Task 2.3: Simplify Branch And Open-App Execution
- [ ] Implementation: Collapse remaining multi-worktree assumptions in [internal/execute/execute.go](/Users/simon/work/github/arbor/internal/execute/execute.go) and summary formatting so open-app targets the single created worktree directly instead of “first successful in batch”
- [ ] Unit Test: Update [internal/execute/execute_test.go](/Users/simon/work/github/arbor/internal/execute/execute_test.go) to verify single-worktree open behavior and remove first-successful batch expectations

## Phase 3: Docs And User-Facing Contract

### Task 3.1: Rewrite Config Docs
- [ ] Implementation: Remove `branch_template` starter examples, `batches`, multi-worktree examples, `.Index`/batch language, and `per_batch` docs from [README.md](/Users/simon/work/github/arbor/README.md)
- [ ] Unit Test: Add/adjust CLI help assertions in [internal/cli/app_test.go](/Users/simon/work/github/arbor/internal/cli/app_test.go) so removed flags/options (`--batch`, batch wording) stay gone

### Task 3.2: Update Product Plans And Release Notes
- [ ] Implementation: Revise historical planning/docs references to batch support in [plans/arbor-prd.md](/Users/simon/work/github/arbor/plans/arbor-prd.md), [plans/arbor-implementation-plan.md](/Users/simon/work/github/arbor/plans/arbor-implementation-plan.md), and [docs/release.md](/Users/simon/work/github/arbor/docs/release.md) where current behavior claims would mislead contributors
- [ ] Unit Test: Add no code test; verify via doc review in Verification

## Verification
- [ ] `go test ./...` passes
- [ ] `arbor init` generated config omits `defaults.branch_template`
- [ ] `arbor create foo` interactive flow always prompts for branch prefix, even with `.arbor.yaml` present
- [ ] `arbor create a b --non-interactive` fails with a clear single-worktree error
- [ ] CLI help and README contain no `--batch`, `batches`, or `per_batch`
- [ ] Single-worktree create still handles env selection, command approval, execution summary, and open-app flow

## Unresolved Questions
- Non-interactive prefix contract is not specified. If interactive mode must always ask for prefix, what is the canonical non-interactive equivalent: `--branch-prefix`, retained `--branch-template`, or passing the final branch name directly?
