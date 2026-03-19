# Deploy to Homebrew

This repo publishes Arbor to Homebrew through GitHub Actions.

- Source repo: `gcstools/arbor`
- Tap repo: `gcstools/homebrew-tap`
- Install command: `brew install gcstools/tap/arbor`
- Release PR workflow: `.github/workflows/release-please.yml`
- Publish workflow: `.github/workflows/release-homebrew.yml`

## How the deploy works

Deploying to Homebrew is release-driven.

Pushing to `main` does not publish a Homebrew release by itself. It updates or opens a release PR through `release-please`.

When you merge that release PR, `release-please` creates the GitHub release and version tag. The publish workflow then:

1. Run `go test ./...`
2. Build release archives for:
   - macOS arm64
   - macOS amd64
   - Linux amd64
3. Upload release archives to the GitHub Release in `gcstools/arbor`
4. Compute archive SHA256 checksums
5. Update `Formula/arbor.rb` in `gcstools/homebrew-tap`
6. Push the formula commit to the tap repo

## One-time setup

### 1. Create the tap repo

Make sure this repository exists:

- `gcstools/homebrew-tap`

It should contain a `Formula/` directory, or allow the workflow to create one.

### 2. Create a token that can push to the tap repo

Create a GitHub personal access token with permission to write repository contents for `gcstools/homebrew-tap`.

Recommended:

- Use a fine-grained personal access token
- Limit it to the `gcstools/homebrew-tap` repo
- Grant repository `Contents` permission with write access

### 3. Add the token as a GitHub Actions secret

Store the token in the source repo, not the tap repo.

- Repo: `gcstools/arbor`
- Secret name: `HOMEBREW_TAP_TOKEN`

Path in GitHub UI:

`gcstools/arbor` -> `Settings` -> `Secrets and variables` -> `Actions`

## Release steps

### 1. Merge the release-ready code to `main`

In normal use:

1. Merge your changes to `main`
2. Wait for `release-please` to open or update the release PR
3. Review the proposed version and release notes

### 2. Sanity-check locally

Before merging the release PR, run:

```bash
go test ./...
go run ./cmd/arbor --help
go run ./cmd/arbor version
```

### 3. Merge the release PR

Merging the release PR is the deploy trigger. `release-please` creates the version tag and GitHub release automatically.

You do not need to create or push tags locally for normal releases.

### 4. Watch the workflow

In GitHub Actions, confirm:

- `release-please` updated or created the release PR after the `main` merge
- `release-homebrew` ran after the release was published

In `release-homebrew`, confirm these jobs pass:

- `test`
- `build`
- `release`
- `homebrew`

### 5. Verify the outputs

After the workflow completes, confirm:

- A GitHub Release exists in `gcstools/arbor` for the tag
- The release contains the platform tarballs and `.sha256` files
- `gcstools/homebrew-tap` contains an updated `Formula/arbor.rb`

### 6. Verify installation

On a machine with Homebrew:

```bash
brew install gcstools/tap/arbor
arbor version
```

If the tap is not already present, Homebrew should auto-tap it. The explicit flow is:

```bash
brew tap gcstools/tap
brew install gcstools/tap/arbor
```

## Operational notes

- The release version comes from `release-please`, based on Conventional Commits.
- The published tag is still in `v0.1.0` form.
- The Homebrew formula version is written without the leading `v`.
- If you retag the same version after a failed release, clean up the bad tag/release state first instead of forcing over it blindly.
- The workflow updates the formula automatically. You do not need to edit `gcstools/homebrew-tap` by hand for normal releases.
- Commit messages on merged PRs should follow Conventional Commits so version bumps are accurate.

## Troubleshooting

### The workflow did not run

Check which stage failed:

- `release-please.yml` runs on pushes to `main`
- `release-homebrew.yml` runs on `release.published`

If the release PR was merged but the publish workflow did not run, confirm a GitHub Release was actually published.

### The workflow cannot push to the tap repo

Check:

- `HOMEBREW_TAP_TOKEN` exists in `gcstools/arbor`
- The token has write access to `gcstools/homebrew-tap`
- The token belongs to a user or bot that can push to that repo

### Homebrew install does not pick up the new version

Try:

```bash
brew update
brew reinstall gcstools/tap/arbor
```

Then verify:

```bash
arbor version
```
