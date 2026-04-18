package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"arbor/internal/model"

	"gopkg.in/yaml.v3"
)

const DefaultConfigPath = ".arbor.yaml"

var defaultConfigFallbackNames = []string{
	".arbor.yml",
	"arbor.yaml",
	"arbor.yml",
}

type File struct {
	Path      string
	Defaults  Defaults                 `yaml:"defaults"`
	EnvFiles  []EnvFileRule            `yaml:"env_files"`
	Commands  []model.CommandCandidate `yaml:"commands"`
	Presets   map[string]Preset        `yaml:"presets"`
	Templates Templates                `yaml:"templates"`
}

type Defaults struct {
	BaseRef          string             `yaml:"base_ref"`
	EnvAction        model.Action       `yaml:"env_action"`
	CommandScope     model.CommandScope `yaml:"command_scope"`
	TrustedAutoRun   bool               `yaml:"trusted_auto_run"`
	OpenApp          string             `yaml:"open_app"`
	WorktreeTemplate string             `yaml:"worktree_template"`
}

type EnvFileRule struct {
	ID            string       `yaml:"id"`
	Label         string       `yaml:"label"`
	SourcePath    string       `yaml:"source_path"`
	TargetPath    string       `yaml:"target_path"`
	DefaultAction model.Action `yaml:"default_action"`
}

type Preset struct {
	Description  string   `yaml:"description"`
	EnvSelection []string `yaml:"env_selection"`
	Commands     []string `yaml:"commands"`
	AutoRun      bool     `yaml:"auto_run"`
}

type Templates struct {
	Branch   string `yaml:"branch"`
	Worktree string `yaml:"worktree"`
}

func Load(path string) (*File, error) {
	if path == "" {
		path = DefaultConfigPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &File{Path: path}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadOptional(path string) (*File, error) {
	cfg, err := Load(path)
	if err == nil {
		return cfg, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		if fallback := resolveFallbackPath(path); fallback != "" {
			return Load(fallback)
		}
		empty := &File{Path: ""}
		empty.applyDefaults()
		return empty, nil
	}
	return nil, err
}

func resolveFallbackPath(path string) string {
	if filepath.Base(path) != DefaultConfigPath {
		return ""
	}

	dir := filepath.Dir(path)
	for _, name := range defaultConfigFallbackNames {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func (f *File) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(f)
}

func StarterConfig() *File {
	cfg := &File{
		Path: DefaultConfigPath,
		Defaults: Defaults{
			EnvAction:        model.ActionSymlink,
			CommandScope:     model.CommandScopePerWorktree,
			TrustedAutoRun:   false,
			WorktreeTemplate: "../{{ .Repo }}-{{ .Name }}",
		},
		EnvFiles: []EnvFileRule{
			{
				ID:            "env",
				Label:         "Primary env file",
				SourcePath:    ".env",
				TargetPath:    ".env",
				DefaultAction: model.ActionSymlink,
			},
		},
		Commands: []model.CommandCandidate{
			{
				ID:      "bootstrap",
				Label:   "Bootstrap deps",
				Command: "go test ./...",
				Scope:   model.CommandScopePerWorktree,
			},
		},
		Presets: map[string]Preset{
			"default": {
				Description:  "Default local setup",
				EnvSelection: []string{"env"},
				Commands:     []string{"bootstrap"},
			},
		},
		Templates: Templates{
			Branch:   "{{ .Name }}",
			Worktree: "../{{ .Repo }}-{{ .Name }}",
		},
	}
	cfg.applyDefaults()
	return cfg
}

func (f *File) Validate() error {
	if f.Defaults.EnvAction == "" {
		f.Defaults.EnvAction = model.ActionSymlink
	}
	if f.Defaults.CommandScope == "" {
		f.Defaults.CommandScope = model.CommandScopePerWorktree
	}

	for i, env := range f.EnvFiles {
		if env.ID == "" {
			return fmt.Errorf("env_files[%d].id is required", i)
		}
		if env.SourcePath == "" {
			return fmt.Errorf("env_files[%d].source_path is required", i)
		}
		if env.TargetPath == "" {
			f.EnvFiles[i].TargetPath = filepath.Base(env.SourcePath)
		}
		if f.EnvFiles[i].DefaultAction == "" {
			f.EnvFiles[i].DefaultAction = f.Defaults.EnvAction
		}
	}

	commandIDs := make([]string, 0, len(f.Commands))
	for i, cmd := range f.Commands {
		if cmd.ID == "" {
			return fmt.Errorf("commands[%d].id is required", i)
		}
		if cmd.Label == "" {
			return fmt.Errorf("commands[%d].label is required", i)
		}
		if cmd.Command == "" {
			return fmt.Errorf("commands[%d].command is required", i)
		}
		if cmd.Scope == "" {
			f.Commands[i].Scope = f.Defaults.CommandScope
		}
		if f.Commands[i].Scope != model.CommandScopePerWorktree {
			return fmt.Errorf("commands[%d].scope must be %q", i, model.CommandScopePerWorktree)
		}
		commandIDs = append(commandIDs, cmd.ID)
	}

	for name, preset := range f.Presets {
		for _, id := range preset.Commands {
			if !slices.Contains(commandIDs, id) {
				return fmt.Errorf("preset %q references unknown command %q", name, id)
			}
		}
	}

	return nil
}

func (f *File) applyDefaults() {
	if f.Defaults.EnvAction == "" {
		f.Defaults.EnvAction = model.ActionSymlink
	}
	if f.Defaults.CommandScope == "" {
		f.Defaults.CommandScope = model.CommandScopePerWorktree
	}
}

func (f *File) ResolvePreset(name string) (Preset, error) {
	preset, ok := f.Presets[name]
	if !ok {
		return Preset{}, fmt.Errorf("preset %q not found", name)
	}
	return preset, nil
}

func (f *File) ResolveTrustedAutoRun(presetName string) bool {
	if presetName == "" {
		return false
	}
	preset, ok := f.Presets[presetName]
	if !ok {
		return false
	}
	return preset.AutoRun && f.Defaults.TrustedAutoRun
}
