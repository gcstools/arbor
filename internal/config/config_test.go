package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"arbor/internal/model"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".arbor.yaml")
	content := `
defaults:
  env_action: symlink
  command_scope: per_worktree
  trusted_auto_run: true
  open_app: cursor
commands:
  - id: bootstrap
    label: Bootstrap
    command: bun install
presets:
  default:
    commands: [bootstrap]
    auto_run: true
env_files:
  - id: env
    source_path: .env
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Defaults.EnvAction != model.ActionSymlink {
		t.Fatalf("unexpected env action: %q", cfg.Defaults.EnvAction)
	}
	if got := cfg.EnvFiles[0].TargetPath; got != ".env" {
		t.Fatalf("unexpected target path: %q", got)
	}
	if !cfg.ResolveTrustedAutoRun("default") {
		t.Fatal("expected trusted auto run")
	}
	if cfg.Defaults.OpenApp != "cursor" {
		t.Fatalf("unexpected open app: %q", cfg.Defaults.OpenApp)
	}
}

func TestLoadConfigRejectsPerBatchCommandScope(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".arbor.yaml")
	content := `
commands:
  - id: bootstrap
    label: Bootstrap
    command: bun install
    scope: per_batch
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), `commands[0].scope must be "per_worktree"`) {
		t.Fatalf("expected per_worktree scope validation error, got %v", err)
	}
}

func TestLoadConfigRejectsUnknownField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".arbor.yaml")
	if err := os.WriteFile(path, []byte("wat: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "field wat not found") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestLoadConfigRejectsBatchesField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".arbor.yaml")
	content := `
batches:
  feature:
    names: [api, web]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "field batches not found") {
		t.Fatalf("expected removed batches field error, got %v", err)
	}
}

func TestResolvePresetMissing(t *testing.T) {
	cfg := &File{}
	_, err := cfg.ResolvePreset("missing")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStarterConfigTemplates(t *testing.T) {
	cfg := StarterConfig()
	if got := cfg.Defaults.WorktreeTemplate; got != "" {
		t.Fatalf("starter config should leave defaults.worktree_template empty, got %q", got)
	}
	if got := cfg.Templates.Branch; got != "{{ .Prefix }}/{{ .Name }}" {
		t.Fatalf("unexpected branch template: %q", got)
	}
	if got := cfg.Templates.Worktree; got != "../{{ .Repo }}-{{ .Prefix }}-{{ .Name }}" {
		t.Fatalf("unexpected worktree template: %q", got)
	}
	data, err := cfg.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML returned error: %v", err)
	}
	text := string(data)
	wantTemplatesBlock := "templates:\n    branch: '{{ .Prefix }}/{{ .Name }}'\n    worktree: ../{{ .Repo }}-{{ .Prefix }}-{{ .Name }}\n"
	if !strings.Contains(text, wantTemplatesBlock) {
		t.Fatalf("starter config should serialize expected templates block:\n%s", text)
	}
	if strings.Contains(text, "worktree_template:") {
		t.Fatalf("starter config should not serialize defaults.worktree_template when templates.worktree is the source of truth:\n%s", text)
	}
}
