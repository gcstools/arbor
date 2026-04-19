package detect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"arbor/internal/config"
	"arbor/internal/model"
)

func TestDetectEnvFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".env"), "A=1")
	writeFile(t, filepath.Join(root, ".env.local"), "A=1")
	writeFile(t, filepath.Join(root, ".env.example"), "A=1")
	writeFile(t, filepath.Join(root, "apps", "web", ".env.local"), "A=1")

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

	candidates, warnings, err := DetectEnvFiles(root, cfg)
	if err != nil {
		t.Fatalf("DetectEnvFiles returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(candidates) != 5 {
		t.Fatalf("got %d candidates", len(candidates))
	}
	if candidates[0].ID != "custom-env" {
		t.Fatalf("expected config candidate first, got %#v", candidates[0])
	}
}

func TestDetectEnvFilesFindsNestedEnvFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".env"), "A=1")
	writeFile(t, filepath.Join(root, "apps", "web", ".env.local"), "A=1")
	writeFile(t, filepath.Join(root, "packages", "api", ".env.test"), "A=1")
	writeFile(t, filepath.Join(root, "packages", "api", ".env.example"), "A=1")

	candidates, warnings, err := DetectEnvFiles(root, nil)
	if err != nil {
		t.Fatalf("DetectEnvFiles returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(candidates) != 3 {
		t.Fatalf("got %d candidates", len(candidates))
	}

	targets := make(map[string]model.EnvCandidate, len(candidates))
	for _, candidate := range candidates {
		targets[candidate.TargetPath] = candidate
	}

	for _, target := range []string{".env", filepath.Join("apps", "web", ".env.local"), filepath.Join("packages", "api", ".env.test")} {
		candidate, ok := targets[target]
		if !ok {
			t.Fatalf("missing target %q in %#v", target, candidates)
		}
		if candidate.Label != target {
			t.Fatalf("expected label %q, got %#v", target, candidate)
		}
	}
}

func TestDetectEnvFilesConfigOverridesNestedTarget(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "apps", "web", ".env.local"), "A=1")
	writeFile(t, filepath.Join(root, "shared", ".env.web"), "A=1")

	cfg := &config.File{
		Defaults: config.Defaults{EnvAction: model.ActionCopy},
		EnvFiles: []config.EnvFileRule{
			{
				ID:         "web-env",
				Label:      "Web env",
				SourcePath: "shared/.env.web",
				TargetPath: filepath.Join("apps", "web", ".env.local"),
			},
		},
	}

	candidates, warnings, err := DetectEnvFiles(root, cfg)
	if err != nil {
		t.Fatalf("DetectEnvFiles returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(candidates) != 2 {
		t.Fatalf("got %d candidates", len(candidates))
	}

	targets := make(map[string]model.EnvCandidate, len(candidates))
	for _, candidate := range candidates {
		targets[candidate.TargetPath] = candidate
	}

	overrideTarget := filepath.Join("apps", "web", ".env.local")
	override, ok := targets[overrideTarget]
	if !ok {
		t.Fatalf("missing override target %q in %#v", overrideTarget, candidates)
	}
	if override.ID != "web-env" {
		t.Fatalf("expected config override, got %#v", override)
	}

	sharedTarget := filepath.Join("shared", ".env.web")
	if _, ok := targets[sharedTarget]; !ok {
		t.Fatalf("missing shared target %q in %#v", sharedTarget, candidates)
	}
}

func TestDetectEnvFilesWarnsWhenConfiguredSourceMissing(t *testing.T) {
	root := t.TempDir()

	cfg := &config.File{
		EnvFiles: []config.EnvFileRule{
			{
				ID:         "missing-env",
				Label:      "Missing env",
				SourcePath: "config/.env.shared",
				TargetPath: ".env.shared",
			},
		},
	}

	candidates, warnings, err := DetectEnvFiles(root, cfg)
	if err != nil {
		t.Fatalf("DetectEnvFiles returned error: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no candidates, got %#v", candidates)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %#v", warnings)
	}
	if got := warnings[0]; !strings.Contains(got, `env_files[missing-env]: source path "config/.env.shared" not found`) {
		t.Fatalf("unexpected warning: %q", got)
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
	gotIDs := make([]string, 0, len(commands))
	for _, command := range commands {
		gotIDs = append(gotIDs, command.ID)
	}
	wantIDs := []string{"node-install", "node-build", "make-build", "just-lint", "bootstrap"}
	if strings.Join(gotIDs, ",") != strings.Join(wantIDs, ",") {
		t.Fatalf("unexpected command order: got %#v want %#v", gotIDs, wantIDs)
	}
}

func TestDetectCommandsConfigOverridePreservesOriginalPosition(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), `{"scripts":{"build":"vite build"}}`)

	cfg := &config.File{
		Commands: []model.CommandCandidate{
			{ID: "node-build", Label: "Build override", Command: "pnpm build:ci", Scope: model.CommandScopePerWorktree},
		},
	}

	commands, warnings, err := DetectCommands(root, cfg)
	if err != nil {
		t.Fatalf("DetectCommands returned error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
	if len(commands) != 2 {
		t.Fatalf("got %d commands", len(commands))
	}
	if commands[0].ID != "node-install" || commands[1].ID != "node-build" {
		t.Fatalf("unexpected command order: %#v", commands)
	}
	if commands[1].Label != "Build override" || commands[1].Command != "pnpm build:ci" || commands[1].Source != "config" {
		t.Fatalf("expected config override in original position, got %#v", commands[1])
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
