# Release Checklist

## Pre-release
1. Run `go test ./...`.
2. Run `go run ./cmd/arbor --help`.
3. Run `go run ./cmd/arbor version`.
4. Run `go run ./cmd/arbor detect` in a disposable fixture repo.
5. Run `go run ./cmd/arbor create feature-auth --plan`.
6. Run `go run ./cmd/arbor create feature-auth` in a disposable fixture repo and confirm the prefix prompt appears.
7. Run `go run ./cmd/arbor create feature-auth --non-interactive` in a disposable fixture repo.
8. Validate `.arbor.yaml` with `go run ./cmd/arbor config validate`.

## Cross-platform
1. Confirm CI passed on Linux, macOS, and Windows.
2. Confirm worktree creation succeeded on at least one real repo per OS.
3. Confirm env-file symlink or copy behavior is acceptable on Windows.

## Homebrew release
1. Confirm the `HOMEBREW_TAP_TOKEN` secret is configured in `gcstools/arbor`.
2. Confirm `gcstools/homebrew-tap` exists and allows the token to push commits.
3. Push a semver tag such as `v0.1.0` to trigger `.github/workflows/release-homebrew.yml`.
4. Confirm the workflow publishes release archives to `gcstools/arbor`.
5. Confirm the workflow updates `Formula/arbor.rb` in `gcstools/homebrew-tap`.
6. Verify install with `brew install gcstools/tap/arbor`.

## Docs
1. Review `README.md`.
2. Review [deploy.md](/Users/simon/work/github/arbor/docs/deploy.md).
3. Update examples if command output changed.
