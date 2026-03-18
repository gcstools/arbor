package detect

import (
	"os"
	"path/filepath"
	"testing"

	"arbor/internal/config"
	"arbor/internal/model"
)

func TestDetectEnvFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".env"), "A=1")
	writeFile(t, filepath.Join(root, ".env.local"), "A=1")
	writeFile(t, filepath.Join(root, ".env.example"), "A=1")

	cfg := &config.File{
		Defaults: config.Defaults{EnvAction: model.ActionCopy},
		EnvFiles: []config.EnvFileRule{
			{
				ID:         "custom-env",
				Label:      "Custom env",
				SourcePath: "config/.env.shared",
				TargetPath: ".env.shared",
			},
		},
	}
	writeFile(t, filepath.Join(root, "config", ".env.shared"), "A=1")

	candidates, err := DetectEnvFiles(root, cfg)
	if err != nil {
		t.Fatalf("DetectEnvFiles returned error: %v", err)
	}
	if len(candidates) != 3 {
		t.Fatalf("got %d candidates", len(candidates))
	}
	if candidates[0].ID != "custom-env" {
		t.Fatalf("expected config candidate first, got %#v", candidates[0])
	}
}

func TestDetectCommands(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), `{"scripts":{"dev":"vite","build":"vite build","test":"vitest"}}`)
	writeFile(t, filepath.Join(root, "Makefile"), "build:\n\tgo build\n")
	writeFile(t, filepath.Join(root, "justfile"), "lint:\n  golangci-lint run\n")

	cfg := &config.File{
		Commands: []model.CommandCandidate{
			{ID: "bootstrap", Label: "Bootstrap", Command: "bun install", Scope: model.CommandScopePerWorktree},
		},
	}

	commands, warnings, err := DetectCommands(root, cfg)
	if err != nil {
		t.Fatalf("DetectCommands returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(commands) != 5 {
		t.Fatalf("got %d commands", len(commands))
	}
	if commands[3].ID != "node-install" || commands[4].ID != "node-build" {
		t.Fatalf("expected node build/install commands, got %#v", commands)
	}
}

func TestDetectPackageJSONDefaultsToNPMInstall(t *testing.T) {
	commands, warning, err := detectPackageJSON(t.TempDir(), []byte(`{"name":"arbor"}`))
	if err != nil {
		t.Fatalf("detectPackageJSON returned error: %v", err)
	}
	if warning != "" {
		t.Fatalf("unexpected warning: %q", warning)
	}
	if len(commands) != 1 || commands[0].Command != "npm install" {
		t.Fatalf("unexpected result: %#v %q", commands, warning)
	}
}

func TestDetectPackageJSONUsesPackageManagerField(t *testing.T) {
	commands, _, err := detectPackageJSON(t.TempDir(), []byte(`{"packageManager":"pnpm@9.0.0","scripts":{"build":"vite build","test":"vitest"}}`))
	if err != nil {
		t.Fatalf("detectPackageJSON returned error: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("got %d commands", len(commands))
	}
	if commands[0].Command != "pnpm install" || commands[1].Command != "pnpm build" {
		t.Fatalf("unexpected commands: %#v", commands)
	}
}

func TestDetectPackageJSONUsesLockfileFallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "yarn.lock"), "# lockfile")

	commands, _, err := detectPackageJSON(root, []byte(`{"scripts":{"build":"vite build"}}`))
	if err != nil {
		t.Fatalf("detectPackageJSON returned error: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("got %d commands", len(commands))
	}
	if commands[0].Command != "yarn install" || commands[1].Command != "yarn build" {
		t.Fatalf("unexpected commands: %#v", commands)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
