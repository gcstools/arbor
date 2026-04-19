package planner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"arbor/internal/config"
	"arbor/internal/detect"
	"arbor/internal/gitutil"
	"arbor/internal/model"

	"github.com/manifoldco/promptui"
	"golang.org/x/term"
)

type Inputs struct {
	CWD            string
	Names          []string
	Branch         string
	BaseRef        string
	Preset         string
	OpenApp        string
	BranchTemplate string
	PathTemplate   string
	NonInteractive bool
}

type CreatePlan struct {
	RepoState gitutil.RepoState
	Worktrees []model.WorktreePlan
	OpenApp   string
	Warnings  []string
}

type nameResolution struct {
	Prefix string
	Name   string
	Branch string
}

func BuildCreatePlan(ctx context.Context, input Inputs, in io.Reader, cfgPath string) (CreatePlan, error) {
	if in == nil {
		in = os.Stdin
	}
	reader := bufio.NewReader(in)

	repoState, err := gitutil.DiscoverRepo(ctx, input.CWD)
	if err != nil {
		return CreatePlan{}, err
	}

	cfg, err := config.LoadOptional(resolveConfigPath(repoState.Root, cfgPath))
	if err != nil {
		return CreatePlan{}, err
	}

	result, err := detect.Scan(repoState.Root, cfg)
	if err != nil {
		return CreatePlan{}, err
	}

	resolvedName, presetName, promptBranchTemplate, promptPathTemplate, err := resolveNamesAndPreset(input, cfg, in, reader)
	if err != nil {
		return CreatePlan{}, err
	}
	if err := validateBranchInputs(input); err != nil {
		return CreatePlan{}, err
	}
	effectivePresetName := resolveEffectivePresetName(presetName, cfg)

	baseRef := firstNonEmpty(input.BaseRef, cfg.Defaults.BaseRef, repoState.CurrentRef, repoState.CurrentCommit)
	openApp := firstNonEmpty(input.OpenApp, cfg.Defaults.OpenApp)
	branchTemplate := firstNonEmpty(input.BranchTemplate, promptBranchTemplate, cfg.Templates.Branch)
	pathTemplate := firstNonEmpty(input.PathTemplate, promptPathTemplate, cfg.Templates.Worktree, cfg.Defaults.WorktreeTemplate)
	if pathTemplate == "" {
		pathTemplate = filepath.Join("..", "{{ .Repo }}-{{ .Name }}")
	}

	repoName := filepath.Base(repoState.Root)
	displayName := joinPrefixName(resolvedName.Prefix, resolvedName.Name)
	templateName := resolvedName.Name
	branchMode := model.BranchModeCreate
	branch := strings.TrimSpace(input.Branch)
	pathPrefix := resolvedName.Prefix
	pathName := templateName
	if branch != "" {
		branchMode = model.BranchModeExisting
		if len(trimNonEmpty(input.Names)) == 0 {
			displayName = sanitizePathPrefix(branch)
			if displayName == "" {
				displayName = branch
			}
		} else {
			pathName = displayName
			pathPrefix = ""
		}
		if !branchExists(repoState, branch) {
			return CreatePlan{}, fmt.Errorf("branch does not exist %q", branch)
		}
		if existingPath, inUse := branchInUse(repoState, branch); inUse {
			return CreatePlan{}, fmt.Errorf("branch already has a worktree %q at %q", branch, existingPath)
		}
	} else {
		branch = resolvedName.Branch
		if branchTemplate != "" {
			branch, err = config.RenderTemplate(branchTemplate, config.TemplateData{
				Prefix: resolvedName.Prefix,
				Name:   templateName,
				Index:  1,
				Base:   baseRef,
				Repo:   repoName,
				Branch: branch,
			})
			if err != nil {
				return CreatePlan{}, fmt.Errorf("branch template for %q: %w", displayName, err)
			}
		}
	}
	pathValue, err := config.RenderTemplate(pathTemplate, config.TemplateData{
		Prefix: pathPrefix,
		Name:   pathName,
		Index:  1,
		Base:   baseRef,
		Repo:   repoName,
		Branch: branch,
	})
	if err != nil {
		return CreatePlan{}, fmt.Errorf("path template for %q: %w", displayName, err)
	}
	path := config.CleanWorktreePath(filepath.Join(repoState.Root, pathValue))
	if _, err := os.Stat(path); err == nil {
		return CreatePlan{}, fmt.Errorf("worktree path already exists %q", path)
	} else if !os.IsNotExist(err) {
		return CreatePlan{}, fmt.Errorf("check worktree path %q: %w", path, err)
	}
	for _, worktree := range repoState.Worktrees {
		if filepath.Clean(worktree.Path) == path {
			return CreatePlan{}, fmt.Errorf("worktree path already exists %q", path)
		}
	}
	worktree := model.WorktreePlan{
		Name:       displayName,
		Branch:     branch,
		BranchMode: branchMode,
		BaseRef:    baseRef,
		Path:       path,
		Preset:     effectivePresetName,
		EnvActions: buildDefaultEnvPlans(result.EnvFiles, cfg, effectivePresetName),
		Commands:   buildDefaultCommandPlans(result.Commands, cfg, effectivePresetName),
	}

	if !input.NonInteractive {
		if err := promptForSelections(reader, &worktree, cfg, effectivePresetName); err != nil {
			return CreatePlan{}, err
		}
	}

	return CreatePlan{
		RepoState: repoState,
		Worktrees: []model.WorktreePlan{worktree},
		OpenApp:   openApp,
		Warnings:  result.Warnings,
	}, nil
}

func RenderSummary(plan CreatePlan) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("repo: %s", plan.RepoState.Root))
	lines = append(lines, fmt.Sprintf("base ref: %s", summarizeBaseRef(plan)))
	lines = append(lines, fmt.Sprintf("open app: %s", firstNonEmpty(plan.OpenApp, "disabled")))
	for _, worktree := range plan.Worktrees {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("worktree %s", worktree.Name))
		lines = append(lines, fmt.Sprintf("  branch: %s", worktree.Branch))
		lines = append(lines, fmt.Sprintf("  preset: %s", worktree.Preset))
		lines = append(lines, fmt.Sprintf("  branch mode: %s", worktree.BranchMode))
		lines = append(lines, fmt.Sprintf("  path: %s", worktree.Path))
		lines = append(lines, fmt.Sprintf("  env actions: %s", summarizeEnvActions(worktree.EnvActions)))
		lines = append(lines, fmt.Sprintf("  commands: %s", summarizeCommands(worktree.Commands)))
	}
	if len(plan.Warnings) > 0 {
		lines = append(lines, "")
		lines = append(lines, "warnings:")
		for _, warning := range plan.Warnings {
			lines = append(lines, "  - "+warning)
		}
	}
	lines = append(lines, "")
	lines = append(lines, "planning only; execution not implemented yet")
	return strings.Join(lines, "\n")
}

func summarizeBaseRef(plan CreatePlan) string {
	if len(plan.Worktrees) == 0 {
		return firstNonEmpty(plan.RepoState.CurrentRef, plan.RepoState.CurrentCommit)
	}

	baseRef := strings.TrimSpace(plan.Worktrees[0].BaseRef)
	if baseRef == "" {
		return firstNonEmpty(plan.RepoState.CurrentRef, plan.RepoState.CurrentCommit)
	}

	for _, worktree := range plan.Worktrees[1:] {
		if strings.TrimSpace(worktree.BaseRef) != baseRef {
			return "multiple"
		}
	}

	return baseRef
}

func resolveNamesAndPreset(input Inputs, cfg *config.File, in io.Reader, reader *bufio.Reader) (nameResolution, string, string, string, error) {
	parts := trimNonEmpty(input.Names)
	if len(parts) == 0 {
		if input.Branch != "" {
			parts = []string{strings.TrimSpace(input.Branch)}
		}
	}
	if len(parts) == 0 {
		if input.NonInteractive {
			return nameResolution{}, "", "", "", fmt.Errorf("worktree name is required in non-interactive mode")
		}
		name, err := promptValue(reader, "worktree name")
		if err != nil {
			return nameResolution{}, "", "", "", err
		}
		if name == "" {
			return nameResolution{}, "", "", "", fmt.Errorf("worktree name is required")
		}
		if strings.TrimSpace(input.BranchTemplate) != "" {
			normalizedName, err := normalizeCreateName(name)
			if err != nil {
				return nameResolution{}, "", "", "", err
			}
			if input.Preset != "" && cfg != nil {
				if _, err := cfg.ResolvePreset(input.Preset); err != nil {
					return nameResolution{}, "", "", "", err
				}
			}
			return normalizedName, input.Preset, "", "", nil
		}
		prefix, err := promptBranchPrefix(in, reader)
		if err != nil {
			return nameResolution{}, "", "", "", err
		}
		resolved, err := normalizeInteractiveCreateName(name, prefix)
		if err != nil {
			return nameResolution{}, "", "", "", err
		}
		if input.Preset != "" && cfg != nil {
			if _, err := cfg.ResolvePreset(input.Preset); err != nil {
				return nameResolution{}, "", "", "", err
			}
		}
		return resolved, input.Preset, "", "", nil
	}

	rawName := strings.Join(parts, " ")
	resolution, err := normalizeCreateName(rawName)
	if err != nil {
		return nameResolution{}, "", "", "", err
	}

	if input.Preset != "" && cfg != nil {
		if _, err := cfg.ResolvePreset(input.Preset); err != nil {
			return nameResolution{}, "", "", "", err
		}
	}

	return resolution, input.Preset, "", "", nil
}

func promptBranchPrefix(in io.Reader, reader *bufio.Reader) (string, error) {
	if supportsInteractiveSelect(in) {
		return promptBranchPrefixSelect(in)
	}

	for {
		fmt.Println("branch prefix:")
		fmt.Println("  1) feat")
		fmt.Println("  2) feature")
		fmt.Println("  3) fix")
		fmt.Println("  4) chore")
		fmt.Println("  5) custom")
		fmt.Println("  6) empty")

		choice, err := promptValue(reader, "select prefix [1-6]")
		if err != nil {
			return "", err
		}

		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "1", "feat":
			return "feat", nil
		case "2", "feature":
			return "feature", nil
		case "3", "fix":
			return "fix", nil
		case "4", "chore":
			return "chore", nil
		case "5", "custom":
			for {
				value, err := promptValue(reader, "custom prefix")
				if err != nil {
					return "", err
				}
				value = sanitizePathPrefix(value)
				if value != "" {
					return value, nil
				}
				fmt.Println("custom prefix must contain at least one letter or number")
			}
		case "6", "empty", "":
			return "", nil
		default:
			fmt.Println("invalid prefix choice")
		}
	}
}

func promptBranchPrefixSelect(in io.Reader) (string, error) {
	options := []string{"feat", "feature", "fix", "chore", "custom", "empty"}
	selectPrompt := promptui.Select{
		Label:     "Branch prefix",
		Items:     options,
		Stdin:     io.NopCloser(in),
		Size:      len(options),
		Templates: branchPrefixSelectTemplates(),
	}

	_, choice, err := selectPrompt.Run()
	if err != nil {
		return "", err
	}

	switch choice {
	case "feat", "feature", "fix", "chore":
		return choice, nil
	case "custom":
		prompt := promptui.Prompt{
			Label: "custom prefix",
			Stdin: io.NopCloser(in),
			Validate: func(value string) error {
				if sanitizePathPrefix(value) == "" {
					return fmt.Errorf("enter a prefix or choose empty")
				}
				return nil
			},
		}
		value, err := prompt.Run()
		if err != nil {
			return "", err
		}
		return sanitizePathPrefix(value), nil
	default:
		return "", nil
	}
}

func branchPrefixSelectTemplates() *promptui.SelectTemplates {
	return &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "> {{ . | cyan }}",
		Inactive: "  {{ . }}",
		Selected: "Branch prefix: {{ . | cyan }}",
	}
}

func supportsInteractiveSelect(in io.Reader) bool {
	file, ok := in.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func sanitizePathPrefix(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case strings.ContainsRune("+#&", r):
			b.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r), strings.ContainsRune("/._-", r):
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		default:
			continue
		}
	}
	return strings.Trim(b.String(), "-")
}

func joinPrefixName(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "-" + name
}

func normalizeCreateName(raw string) (nameResolution, error) {
	tokens := strings.Fields(strings.TrimSpace(raw))
	if len(tokens) == 0 {
		return nameResolution{}, fmt.Errorf("worktree name is required")
	}

	name := sanitizePathPrefix(strings.Join(tokens, " "))
	if name == "" {
		return nameResolution{}, fmt.Errorf("worktree name is required")
	}
	if len(tokens) > 1 {
		prefix := strings.ToLower(tokens[0])
		switch prefix {
		case "feat", "feature", "fix", "chore":
			remainder := sanitizePathPrefix(strings.Join(tokens[1:], " "))
			if remainder != "" {
				return nameResolution{
					Prefix: prefix,
					Name:   remainder,
					Branch: prefix + "/" + remainder,
				}, nil
			}
		}
	}

	return nameResolution{Name: name, Branch: name}, nil
}

func normalizeInteractiveCreateName(rawName, prefix string) (nameResolution, error) {
	prefix = sanitizePathPrefix(prefix)
	switch prefix {
	case "":
		resolution, err := normalizeCreateName(rawName)
		if err != nil {
			return nameResolution{}, err
		}
		return resolution, nil
	}

	name := sanitizePathPrefix(rawName)
	if name == "" {
		return nameResolution{}, fmt.Errorf("worktree name is required")
	}
	return nameResolution{
		Prefix: prefix,
		Name:   name,
		Branch: prefix + "/" + name,
	}, nil
}

func buildDefaultEnvPlans(candidates []model.EnvCandidate, cfg *config.File, presetName string) []model.EnvPlan {
	selected := map[string]struct{}{}
	if cfg != nil && presetName != "" {
		if preset, ok := cfg.Presets[presetName]; ok {
			for _, id := range preset.EnvSelection {
				selected[id] = struct{}{}
			}
		}
	}

	plans := make([]model.EnvPlan, 0, len(candidates))
	for _, candidate := range candidates {
		action := candidate.DefaultAction
		if len(selected) > 0 {
			if _, ok := selected[candidate.ID]; !ok {
				action = model.ActionSkip
			}
		}
		plans = append(plans, model.EnvPlan{
			Candidate: candidate,
			Action:    action,
		})
	}
	return plans
}

func buildDefaultCommandPlans(candidates []model.CommandCandidate, cfg *config.File, presetName string) []model.CommandExecution {
	selected := map[string]struct{}{}
	if cfg != nil && presetName != "" {
		if preset, ok := cfg.Presets[presetName]; ok {
			for _, id := range preset.Commands {
				selected[id] = struct{}{}
			}
		}
	}

	plans := make([]model.CommandExecution, 0, len(candidates))
	for _, candidate := range candidates {
		_, isSelected := selected[candidate.ID]
		plans = append(plans, model.CommandExecution{
			Candidate: candidate,
			Approved:  isSelected,
		})
	}
	return plans
}

func promptForSelections(reader *bufio.Reader, worktree *model.WorktreePlan, cfg *config.File, presetName string) error {
	promptEnv := shouldPromptEnvSelections(cfg)
	promptCommands := shouldPromptCommandSelections(cfg, presetName)
	lastEnvAction := model.Action("")
	if promptEnv {
		for i := range worktree.EnvActions {
			defaultAction := worktree.EnvActions[i].Action
			if lastEnvAction != "" {
				defaultAction = lastEnvAction
				worktree.EnvActions[i].Action = defaultAction
			}
			prompt := formatEnvPrompt(worktree.Name, worktree.EnvActions[i].Candidate.TargetPath, defaultAction)
			value, err := promptValue(reader, prompt)
			if err != nil {
				return err
			}
			if value == "" {
				continue
			}
			action, err := parseAction(value)
			if err != nil {
				return err
			}
			worktree.EnvActions[i].Action = action
			lastEnvAction = action
		}
	}

	autoRun := cfg != nil && cfg.ResolveTrustedAutoRun(presetName)
	if promptCommands {
		for i := range worktree.Commands {
			if autoRun && worktree.Commands[i].Candidate.Trusted {
				worktree.Commands[i].Approved = true
				continue
			}
			defaultAnswer := "n"
			if worktree.Commands[i].Approved {
				defaultAnswer = "y"
			}
			value, err := promptValue(reader, formatCommandPrompt(worktree.Name, worktree.Commands[i].Candidate.Label, defaultAnswer))
			if err != nil {
				return err
			}
			if value == "" {
				worktree.Commands[i].Approved = defaultAnswer == "y"
				continue
			}
			worktree.Commands[i].Approved = parseYesNo(value)
		}
	}
	return nil
}

func promptValue(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func formatEnvPrompt(name, targetPath string, defaultAction model.Action) string {
	return fmt.Sprintf("[%s] env %s action [symlink/copy/skip] (%s)", name, targetPath, defaultAction)
}

func formatCommandPrompt(name, label, defaultAnswer string) string {
	return fmt.Sprintf("[%s] run %s? [y/N] (%s)", name, label, defaultAnswer)
}

func parseAction(value string) (model.Action, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "symlink", "s":
		return model.ActionSymlink, nil
	case "copy", "c":
		return model.ActionCopy, nil
	case "skip", "k", "n":
		return model.ActionSkip, nil
	default:
		return "", fmt.Errorf("unknown env action %q", value)
	}
}

func parseYesNo(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "y", "yes", "true":
		return true
	default:
		return false
	}
}

func summarizeEnvActions(actions []model.EnvPlan) string {
	if len(actions) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(actions))
	for _, action := range actions {
		parts = append(parts, fmt.Sprintf("%s=%s", action.Candidate.TargetPath, action.Action))
	}
	return strings.Join(parts, ", ")
}

func summarizeCommands(commands []model.CommandExecution) string {
	if len(commands) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(commands))
	for _, command := range commands {
		state := "skip"
		if command.Approved {
			state = "run"
		}
		parts = append(parts, fmt.Sprintf("%s=%s", command.Candidate.Label, state))
	}
	return strings.Join(parts, ", ")
}

func resolveConfigPath(repoRoot, cfgPath string) string {
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath
	}
	if filepath.IsAbs(cfgPath) {
		return cfgPath
	}
	return filepath.Join(repoRoot, cfgPath)
}

func dedupeNonEmpty(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func trimNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func validateBranchInputs(input Inputs) error {
	if input.Branch == "" {
		return nil
	}
	if input.BranchTemplate != "" {
		return fmt.Errorf("--branch cannot be used with --branch-template")
	}
	if input.BaseRef != "" {
		return fmt.Errorf("--branch cannot be used with --base")
	}
	return nil
}

func branchExists(repoState gitutil.RepoState, branch string) bool {
	return slices.Contains(repoState.LocalBranches, branch)
}

func branchInUse(repoState gitutil.RepoState, branch string) (string, bool) {
	for _, worktree := range repoState.Worktrees {
		if worktree.Branch == branch {
			return worktree.Path, true
		}
	}
	return "", false
}

func resolveEffectivePresetName(inputPreset string, cfg *config.File) string {
	if inputPreset != "" {
		return inputPreset
	}
	if cfg != nil {
		if _, ok := cfg.Presets["default"]; ok {
			return "default"
		}
	}
	return ""
}

func shouldPromptEnvSelections(cfg *config.File) bool {
	if cfg != nil && len(cfg.EnvFiles) > 0 {
		return false
	}
	return true
}

func shouldPromptCommandSelections(cfg *config.File, presetName string) bool {
	if cfg != nil && presetName != "" {
		if preset, ok := cfg.Presets[presetName]; ok && len(preset.Commands) > 0 {
			return false
		}
	}
	return true
}
