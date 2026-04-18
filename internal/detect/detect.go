package detect

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"arbor/internal/config"
	"arbor/internal/model"
)

type Result struct {
	RepoRoot  string
	EnvFiles  []model.EnvCandidate
	Commands  []model.CommandCandidate
	Warnings  []string
	ConfigHit bool
}

func Scan(repoRoot string, cfg *config.File) (Result, error) {
	envFiles, envWarnings, err := DetectEnvFiles(repoRoot, cfg)
	if err != nil {
		return Result{}, err
	}
	commands, commandWarnings, err := DetectCommands(repoRoot, cfg)
	if err != nil {
		return Result{}, err
	}
	warnings := append(envWarnings, commandWarnings...)

	return Result{
		RepoRoot:  repoRoot,
		EnvFiles:  envFiles,
		Commands:  commands,
		Warnings:  warnings,
		ConfigHit: cfg != nil && cfg.Path != "",
	}, nil
}

func DetectEnvFiles(repoRoot string, cfg *config.File) ([]model.EnvCandidate, []string, error) {
	candidates := map[string]model.EnvCandidate{}
	targetIndex := map[string]string{}
	var warnings []string

	if err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipEnvDir(repoRoot, path) {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		if !looksLikeEnvFile(name) {
			return nil
		}

		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.Clean(relPath)

		id := sanitizeID("env", relPath)
		candidates[id] = model.EnvCandidate{
			ID:            id,
			Label:         relPath,
			SourcePath:    path,
			TargetPath:    relPath,
			DefaultAction: model.ActionSymlink,
			Source:        "detected",
		}
		targetIndex[relPath] = id
		return nil
	}); err != nil {
		return nil, nil, err
	}

	if cfg != nil {
		for _, rule := range cfg.EnvFiles {
			sourcePath := rule.SourcePath
			if !filepath.IsAbs(sourcePath) {
				sourcePath = filepath.Join(repoRoot, sourcePath)
			}
			info, err := os.Stat(sourcePath)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("env_files[%s]: source path %q not found", rule.ID, rule.SourcePath))
				continue
			}
			if info.IsDir() {
				warnings = append(warnings, fmt.Sprintf("env_files[%s]: source path %q is directory", rule.ID, rule.SourcePath))
				continue
			}

			id := rule.ID
			if id == "" {
				id = sanitizeID("env", filepath.Base(sourcePath))
			}
			if existingID, ok := targetIndex[rule.TargetPath]; ok {
				delete(candidates, existingID)
			}
			candidates[id] = model.EnvCandidate{
				ID:            id,
				Label:         firstNonEmpty(rule.Label, filepath.Base(sourcePath)),
				SourcePath:    sourcePath,
				TargetPath:    rule.TargetPath,
				DefaultAction: firstAction(rule.DefaultAction, cfg.Defaults.EnvAction, model.ActionSymlink),
				Source:        "config",
			}
			targetIndex[rule.TargetPath] = id
		}
	}

	return sortEnvCandidates(candidates), warnings, nil
}

func DetectCommands(repoRoot string, cfg *config.File) ([]model.CommandCandidate, []string, error) {
	collector := newCommandCollector()
	var warnings []string

	pkgJSONPath := filepath.Join(repoRoot, "package.json")
	if data, err := os.ReadFile(pkgJSONPath); err == nil {
		pkgCommands, warn, err := detectPackageJSON(repoRoot, data)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("package.json: %v", err))
		} else {
			for _, cmd := range pkgCommands {
				collector.Add(cmd)
			}
		}
		if warn != "" {
			warnings = append(warnings, warn)
		}
	}

	for _, name := range []string{"Makefile", "makefile"} {
		makePath := filepath.Join(repoRoot, name)
		if data, err := os.ReadFile(makePath); err == nil {
			for _, cmd := range detectMakeTargets(string(data)) {
				collector.Add(cmd)
			}
			break
		}
	}

	for _, name := range []string{"justfile", ".justfile"} {
		justPath := filepath.Join(repoRoot, name)
		if data, err := os.ReadFile(justPath); err == nil {
			for _, cmd := range detectJustTargets(string(data)) {
				collector.Add(cmd)
			}
			break
		}
	}

	if cfg != nil {
		for _, cmd := range cfg.Commands {
			merged := cmd
			if merged.Source == "" {
				merged.Source = "config"
			}
			collector.Add(merged)
		}
	}

	return collector.List(), warnings, nil
}

type commandCollector struct {
	order []string
	items map[string]model.CommandCandidate
}

func newCommandCollector() *commandCollector {
	return &commandCollector{
		order: make([]string, 0),
		items: make(map[string]model.CommandCandidate),
	}
}

func (c *commandCollector) Add(candidate model.CommandCandidate) {
	if _, exists := c.items[candidate.ID]; !exists {
		c.order = append(c.order, candidate.ID)
	}
	c.items[candidate.ID] = candidate
}

func (c *commandCollector) List() []model.CommandCandidate {
	out := make([]model.CommandCandidate, 0, len(c.order))
	for _, id := range c.order {
		out = append(out, c.items[id])
	}
	return out
}

type packageJSON struct {
	PackageManager string            `json:"packageManager"`
	Scripts        map[string]string `json:"scripts"`
}

func detectPackageJSON(repoRoot string, data []byte) ([]model.CommandCandidate, string, error) {
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, "", err
	}

	manager := detectPackageManager(repoRoot, pkg.PackageManager)
	commands := []model.CommandCandidate{
		{
			ID:      "node-install",
			Label:   manager + " install",
			Command: manager + " install",
			Scope:   model.CommandScopePerWorktree,
			Source:  "package.json",
		},
	}
	if _, ok := pkg.Scripts["build"]; ok {
		commands = append(commands, model.CommandCandidate{
			ID:      "node-build",
			Label:   manager + " build",
			Command: buildCommandForManager(manager),
			Scope:   model.CommandScopePerWorktree,
			Source:  "package.json",
		})
	}
	return commands, "", nil
}

func detectPackageManager(repoRoot, packageManager string) string {
	packageManager = strings.TrimSpace(packageManager)
	for _, manager := range []string{"pnpm", "yarn", "bun", "npm"} {
		if packageManager == manager || strings.HasPrefix(packageManager, manager+"@") {
			return manager
		}
	}

	for _, candidate := range []struct {
		file    string
		manager string
	}{
		{file: "pnpm-lock.yaml", manager: "pnpm"},
		{file: "yarn.lock", manager: "yarn"},
		{file: "bun.lock", manager: "bun"},
		{file: "bun.lockb", manager: "bun"},
		{file: "package-lock.json", manager: "npm"},
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, candidate.file)); err == nil {
			return candidate.manager
		}
	}

	return "npm"
}

func buildCommandForManager(manager string) string {
	switch manager {
	case "npm":
		return "npm run build"
	case "bun":
		return "bun run build"
	default:
		return manager + " build"
	}
}

func detectMakeTargets(data string) []model.CommandCandidate {
	lines := strings.Split(data, "\n")
	var commands []model.CommandCandidate
	seen := map[string]struct{}{}
	for _, line := range lines {
		target, ok := parseRuleTarget(line)
		if !ok || target == "" {
			continue
		}
		id := "make-" + sanitizeID("", target)
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		commands = append(commands, model.CommandCandidate{
			ID:      id,
			Label:   "make:" + target,
			Command: "make " + target,
			Scope:   model.CommandScopePerWorktree,
			Source:  "Makefile",
		})
	}
	return commands
}

func detectJustTargets(data string) []model.CommandCandidate {
	lines := strings.Split(data, "\n")
	var commands []model.CommandCandidate
	seen := map[string]struct{}{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "set ") {
			continue
		}
		target, ok := parseRuleTarget(line)
		if !ok || strings.Contains(target, " ") || strings.HasPrefix(target, "_") {
			continue
		}
		id := "just-" + sanitizeID("", target)
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		commands = append(commands, model.CommandCandidate{
			ID:      id,
			Label:   "just:" + target,
			Command: "just " + target,
			Scope:   model.CommandScopePerWorktree,
			Source:  "justfile",
		})
	}
	return commands
}

func parseRuleTarget(line string) (string, bool) {
	if strings.HasPrefix(line, ".PHONY:") || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ") {
		return "", false
	}
	target, _, found := strings.Cut(line, ":")
	if !found {
		return "", false
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return "", false
	}
	if strings.ContainsAny(target, "$()") || strings.Contains(target, "=") {
		return "", false
	}
	fields := strings.Fields(target)
	if len(fields) != 1 {
		return "", false
	}
	return fields[0], true
}

func looksLikeEnvFile(name string) bool {
	if name == ".env" || name == ".envrc" {
		return true
	}
	if strings.HasPrefix(name, ".env.") {
		suffix := strings.TrimPrefix(name, ".env.")
		excluded := []string{"example", "sample", "template", "dist"}
		for _, v := range excluded {
			if suffix == v || strings.HasSuffix(suffix, "."+v) {
				return false
			}
		}
		return suffix != ""
	}
	return false
}

func shouldSkipEnvDir(repoRoot, path string) bool {
	if filepath.Clean(path) == filepath.Clean(repoRoot) {
		return false
	}
	name := filepath.Base(path)
	switch name {
	case ".git", "node_modules":
		return true
	default:
		return false
	}
}

func sanitizeID(prefix, value string) string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer(" ", "-", "/", "-", "_", "-", ".", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, "-")
	if prefix == "" {
		return value
	}
	if value == "" {
		return prefix
	}
	return prefix + "-" + value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstAction(values ...model.Action) model.Action {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return model.ActionSymlink
}

func sortEnvCandidates(in map[string]model.EnvCandidate) []model.EnvCandidate {
	out := make([]model.EnvCandidate, 0, len(in))
	for _, candidate := range in {
		out = append(out, candidate)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}
