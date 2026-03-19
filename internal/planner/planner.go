package planner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

	names, presetName, promptBranchTemplate, promptPathTemplate, err := resolveNamesAndPreset(input, cfg, in, reader)
	if err != nil {
		return CreatePlan{}, err
	}
	baseRef := firstNonEmpty(input.BaseRef, cfg.Defaults.BaseRef, repoState.CurrentRef, repoState.CurrentCommit)
	openApp := firstNonEmpty(input.OpenApp, cfg.Defaults.OpenApp)
	branchTemplate := firstNonEmpty(input.BranchTemplate, promptBranchTemplate, cfg.Templates.Branch)
	pathTemplate := firstNonEmpty(input.PathTemplate, promptPathTemplate, cfg.Defaults.WorktreeTemplate, cfg.Templates.Worktree)

	if branchTemplate == "" {
		branchTemplate = "{{ .Name }}"
	}
	if pathTemplate == "" {
		pathTemplate = filepath.Join("..", "{{ .Repo }}-{{ .Name }}")
	}

	repoName := filepath.Base(repoState.Root)
	name := names[0]
	branch, err := config.RenderTemplate(branchTemplate, config.TemplateData{
		Name:  name,
		Index: 1,
		Base:  baseRef,
		Repo:  repoName,
	})
	if err != nil {
		return CreatePlan{}, fmt.Errorf("branch template for %q: %w", name, err)
	}
	pathValue, err := config.RenderTemplate(pathTemplate, config.TemplateData{
		Name:   name,
		Index:  1,
		Base:   baseRef,
		Repo:   repoName,
		Branch: branch,
	})
	if err != nil {
		return CreatePlan{}, fmt.Errorf("path template for %q: %w", name, err)
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
		Name:       name,
		Branch:     branch,
		BaseRef:    baseRef,
		Path:       path,
		Preset:     presetName,
		EnvActions: buildDefaultEnvPlans(result.EnvFiles, cfg, presetName),
		Commands:   buildDefaultCommandPlans(result.Commands, cfg, presetName),
	}

	if !input.NonInteractive {
		if err := promptForSelections(reader, &worktree, cfg, presetName); err != nil {
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
	lines = append(lines, fmt.Sprintf("base ref: %s", firstNonEmpty(plan.RepoState.CurrentRef, plan.RepoState.CurrentCommit)))
	lines = append(lines, fmt.Sprintf("open app: %s", firstNonEmpty(plan.OpenApp, "disabled")))
	for _, worktree := range plan.Worktrees {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("worktree %s", worktree.Name))
		lines = append(lines, fmt.Sprintf("  branch: %s", worktree.Branch))
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

func resolveNamesAndPreset(input Inputs, cfg *config.File, in io.Reader, reader *bufio.Reader) ([]string, string, string, string, error) {
	names := dedupeNonEmpty(input.Names)
	promptBranchTemplate := ""
	promptPathTemplate := ""
	if len(names) > 1 {
		return nil, "", "", "", fmt.Errorf("exactly one worktree name is supported")
	}
	if len(names) == 0 {
		if input.NonInteractive {
			return nil, "", "", "", fmt.Errorf("worktree name is required in non-interactive mode")
		}
		name, err := promptValue(reader, "worktree name")
		if err != nil {
			return nil, "", "", "", err
		}
		if name == "" {
			return nil, "", "", "", fmt.Errorf("worktree name is required")
		}
		if input.BranchTemplate == "" {
			prefix, err := promptBranchPrefix(in, reader)
			if err != nil {
				return nil, "", "", "", err
			}
			if prefix != "" {
				promptBranchTemplate = prefix + "/{{ .Name }}"
				promptPathTemplate = filepath.Join("..", "{{ .Repo }}-"+sanitizePathPrefix(prefix)+"-{{ .Name }}")
			}
		}
		names = []string{name}
	}

	if input.Preset != "" && cfg != nil {
		if _, err := cfg.ResolvePreset(input.Preset); err != nil {
			return nil, "", "", "", err
		}
	}

	return names, input.Preset, promptBranchTemplate, promptPathTemplate, nil
}

func promptBranchPrefix(in io.Reader, reader *bufio.Reader) (string, error) {
	if supportsInteractiveSelect(in) {
		return promptBranchPrefixSelect(in)
	}

	for {
		fmt.Println("branch prefix:")
		fmt.Println("  1) feat")
		fmt.Println("  2) fix")
		fmt.Println("  3) chore")
		fmt.Println("  4) custom")
		fmt.Println("  5) empty")

		choice, err := promptValue(reader, "select prefix [1-5]")
		if err != nil {
			return "", err
		}

		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "1", "feat":
			return "feat", nil
		case "2", "fix":
			return "fix", nil
		case "3", "chore":
			return "chore", nil
		case "4", "custom":
			value, err := promptValue(reader, "custom prefix")
			if err != nil {
				return "", err
			}
			return strings.Trim(value, "/ "), nil
		case "5", "empty", "":
			return "", nil
		default:
			fmt.Println("invalid prefix choice")
		}
	}
}

func promptBranchPrefixSelect(in io.Reader) (string, error) {
	options := []string{"feat", "fix", "chore", "custom", "empty"}
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
	case "feat", "fix", "chore":
		return choice, nil
	case "custom":
		prompt := promptui.Prompt{
			Label: "custom prefix",
			Stdin: io.NopCloser(in),
			Validate: func(value string) error {
				if strings.Trim(value, "/ ") == "" {
					return fmt.Errorf("enter a prefix or choose empty")
				}
				return nil
			},
		}
		value, err := prompt.Run()
		if err != nil {
			return "", err
		}
		return strings.Trim(value, "/ "), nil
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
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", "_", "-", ".", "-")
	value = replacer.Replace(value)
	return strings.Trim(value, "-")
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
	autoRun := false
	if cfg != nil && presetName != "" {
		if preset, ok := cfg.Presets[presetName]; ok {
			for _, id := range preset.Commands {
				selected[id] = struct{}{}
			}
		}
		autoRun = cfg.ResolveTrustedAutoRun(presetName)
	}

	plans := make([]model.CommandExecution, 0, len(candidates))
	for _, candidate := range candidates {
		_, isSelected := selected[candidate.ID]
		approved := autoRun && isSelected && candidate.Trusted
		plans = append(plans, model.CommandExecution{
			Candidate: candidate,
			Approved:  approved,
		})
	}
	return plans
}

func promptForSelections(reader *bufio.Reader, worktree *model.WorktreePlan, cfg *config.File, presetName string) error {
	lastEnvAction := model.Action("")
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

	autoRun := cfg != nil && cfg.ResolveTrustedAutoRun(presetName)
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
