# Deploy for Dev Testing

This project is a Go CLI. For local development, "deploy" means compiling the `arbor` binary and running it against test repositories on your machine.

## Prerequisites
- Go `1.24.4` or newer
- Git installed and available on `PATH`

## Install dependencies
```bash
go mod tidy
```

## Run without building
Use this while iterating on command behavior:

```bash
go run ./cmd/arbor --help
go run ./cmd/arbor config --help
```

## Build local binary
Build the binary into a local `bin/` directory:

```bash
mkdir -p bin
go build -o ./bin/arbor ./cmd/arbor
```

Run it:

```bash
./bin/arbor --help
./bin/arbor version
./bin/arbor completion bash
```

## Run tests
```bash
go test ./...
```

If your environment blocks the default Go build cache, use:

```bash
GOCACHE=/tmp/arbor-go-build-cache go test ./...
```

## Cross-compile for manual testing
Build binaries for common targets:

```bash
mkdir -p bin
GOOS=darwin GOARCH=arm64 go build -o ./bin/arbor-darwin-arm64 ./cmd/arbor
GOOS=linux GOARCH=amd64 go build -o ./bin/arbor-linux-amd64 ./cmd/arbor
GOOS=windows GOARCH=amd64 go build -o ./bin/arbor-windows-amd64.exe ./cmd/arbor
```

## Suggested dev test flow
1. Build the binary.
2. Create or choose a disposable Git repository.
3. Run `./bin/arbor --help`.
4. Run `./bin/arbor config validate --config path/to/.arbor.yaml`.
5. When Phase 3+ lands, test `detect` and `create` against fixture repos.

## Notes
- The binary is self-contained; there is no separate runtime install step.
- Current config default path is `.arbor.yaml` in the repo root.
- Shell completion generation is available through Cobra:
- Homebrew releases are published from tags through `.github/workflows/release-homebrew.yml`.

```bash
./bin/arbor completion zsh
./bin/arbor completion bash
./bin/arbor completion fish
./bin/arbor completion powershell
```
