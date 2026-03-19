package model

type Action string

const (
	ActionSymlink Action = "symlink"
	ActionCopy    Action = "copy"
	ActionSkip    Action = "skip"
)

type BranchMode string

const (
	BranchModeCreate   BranchMode = "create"
	BranchModeExisting BranchMode = "existing"
)

type WorktreePlan struct {
	Name       string             `yaml:"name" json:"name"`
	Branch     string             `yaml:"branch" json:"branch"`
	BranchMode BranchMode         `yaml:"branch_mode,omitempty" json:"branch_mode,omitempty"`
	BaseRef    string             `yaml:"base_ref,omitempty" json:"base_ref,omitempty"`
	Path       string             `yaml:"path" json:"path"`
	Preset     string             `yaml:"preset,omitempty" json:"preset,omitempty"`
	EnvActions []EnvPlan          `yaml:"env_actions,omitempty" json:"env_actions,omitempty"`
	Commands   []CommandExecution `yaml:"commands,omitempty" json:"commands,omitempty"`
}

type EnvCandidate struct {
	ID            string `yaml:"id" json:"id"`
	Label         string `yaml:"label,omitempty" json:"label,omitempty"`
	SourcePath    string `yaml:"source_path" json:"source_path"`
	TargetPath    string `yaml:"target_path" json:"target_path"`
	DefaultAction Action `yaml:"default_action" json:"default_action"`
	Source        string `yaml:"source,omitempty" json:"source,omitempty"`
}

type EnvPlan struct {
	Candidate EnvCandidate `yaml:"candidate" json:"candidate"`
	Action    Action       `yaml:"action" json:"action"`
}

type CommandScope string

const (
	CommandScopePerWorktree CommandScope = "per_worktree"
)

type CommandCandidate struct {
	ID       string       `yaml:"id" json:"id"`
	Label    string       `yaml:"label" json:"label"`
	Command  string       `yaml:"command" json:"command"`
	Scope    CommandScope `yaml:"scope" json:"scope"`
	Trusted  bool         `yaml:"trusted,omitempty" json:"trusted,omitempty"`
	Source   string       `yaml:"source,omitempty" json:"source,omitempty"`
	Selected bool         `yaml:"selected,omitempty" json:"selected,omitempty"`
}

type CommandExecution struct {
	Candidate CommandCandidate `yaml:"candidate" json:"candidate"`
	Approved  bool             `yaml:"approved" json:"approved"`
}

type Preset struct {
	Name         string   `yaml:"name" json:"name"`
	Description  string   `yaml:"description,omitempty" json:"description,omitempty"`
	EnvSelection []string `yaml:"env_selection,omitempty" json:"env_selection,omitempty"`
	CommandIDs   []string `yaml:"commands,omitempty" json:"commands,omitempty"`
	AutoRun      bool     `yaml:"auto_run,omitempty" json:"auto_run,omitempty"`
}

type TemplateOutput struct {
	Value  string `yaml:"value" json:"value"`
	Source string `yaml:"source,omitempty" json:"source,omitempty"`
}

type WorktreeResult struct {
	Path          string          `yaml:"path" json:"path"`
	Branch        string          `yaml:"branch" json:"branch"`
	EnvResults    []EnvResult     `yaml:"env_results,omitempty" json:"env_results,omitempty"`
	CommandResult []CommandResult `yaml:"command_results,omitempty" json:"command_results,omitempty"`
	Created       bool            `yaml:"created" json:"created"`
	Error         string          `yaml:"error,omitempty" json:"error,omitempty"`
}

type EnvResult struct {
	ID      string `yaml:"id" json:"id"`
	Action  Action `yaml:"action" json:"action"`
	Applied bool   `yaml:"applied" json:"applied"`
	Error   string `yaml:"error,omitempty" json:"error,omitempty"`
}

type CommandResult struct {
	ID       string `yaml:"id" json:"id"`
	Command  string `yaml:"command" json:"command"`
	Scope    string `yaml:"scope" json:"scope"`
	Executed bool   `yaml:"executed" json:"executed"`
	ExitCode int    `yaml:"exit_code,omitempty" json:"exit_code,omitempty"`
	Error    string `yaml:"error,omitempty" json:"error,omitempty"`
}

type ExecutionSummary struct {
	RepoRoot   string           `yaml:"repo_root" json:"repo_root"`
	Worktrees  []WorktreeResult `yaml:"worktrees" json:"worktrees"`
	Warnings   []string         `yaml:"warnings,omitempty" json:"warnings,omitempty"`
	OpenApp    string           `yaml:"open_app,omitempty" json:"open_app,omitempty"`
	OpenedPath string           `yaml:"opened_path,omitempty" json:"opened_path,omitempty"`
	OpenError  string           `yaml:"open_error,omitempty" json:"open_error,omitempty"`
	HasFailure bool             `yaml:"has_failure" json:"has_failure"`
}
