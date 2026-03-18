# Deploy to Homebrew

This repo publishes Arbor to Homebrew through GitHub Actions.

- Source repo: `gcstools/arbor`
- Tap repo: `gcstools/homebrew-tap`
- Install command: `brew install gcstools/tap/arbor`
- Release workflow: `.github/workflows/release-homebrew.yml`

## How the deploy works

Deploying to Homebrew is tag-driven, not branch-driven.

Pushing to `main` does not publish a Homebrew release by itself. It only updates the code that will be used the next time you create a version tag.

When you push a tag that matches `v*`, for example `v0.1.0`, GitHub Actions will:

1. Run `go test ./...`
2. Build release archives for:
   - macOS arm64
   - macOS amd64
   - Linux amd64
3. Create or update the GitHub Release in `gcstools/arbor`
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

The tag should point at the commit you want to release. In normal use, that means:

1. Merge your changes to `main`
2. Pull the latest `main` locally
3. Create the release tag from that commit

### 2. Sanity-check locally

Before tagging, run:

```bash
go test ./...
go run ./cmd/arbor --help
go run ./cmd/arbor version
```

### 3. Create and push a version tag

Example:

```bash
git checkout main
git pull origin main
git tag v0.1.0
git push origin v0.1.0
```

That tag push is the deploy trigger.

You do not need to push `main` again after tagging if the tag already points to the correct commit.

### 4. Watch the workflow

In GitHub Actions, open the `release-homebrew` workflow and confirm that these jobs pass:

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

- The release version comes from the Git tag, for example `v0.1.0`.
- The Homebrew formula version is written without the leading `v`.
- If you retag the same version after a failed release, clean up the bad tag/release state first instead of forcing over it blindly.
- The workflow updates the formula automatically. You do not need to edit `gcstools/homebrew-tap` by hand for normal releases.

## Troubleshooting

### The workflow did not run

Check that the pushed tag matches the workflow trigger:

- `v*`

Examples that will run:

- `v0.1.0`
- `v1.2.3`

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
