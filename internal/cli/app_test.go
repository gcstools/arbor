package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootHelpListsCommands(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	if err := Execute([]string{"help"}, IOStreams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := out.String()
	for _, want := range []string{"create", "init", "detect", "pull", "config", "version", "completion"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	if err := Execute([]string{"version"}, IOStreams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := out.String()
	if !strings.HasPrefix(got, "arbor ") {
		t.Fatalf("version output missing version prefix:\n%s", got)
	}
	for _, unwanted := range []string{"commit:", "built:"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("version output should not include %q:\n%s", unwanted, got)
		}
	}
}

func TestCompletionCommandStdout(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	if err := Execute([]string{"completion", "zsh", "--stdout"}, IOStreams{
		In:     bytes.NewBuffer(nil),
		Out:    &out,
		ErrOut: &errOut,
	}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(out.String(), "#compdef arbor") {
		t.Fatalf("unexpected completion output: %s", out.String())
	}
}

func TestZshCompletionInstallerUpdatesZshrc(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := NewRootCommand(IOStreams{In: bytes.NewBuffer(nil), Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}})
	installer, err := buildCompletionInstaller(root, completionShellZsh, "", false)
	if err != nil {
		t.Fatalf("buildCompletionInstaller returned error: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(installer.scriptPath), 0o755); err != nil {
		t.Fatal(err)
	}
	scriptFile, err := os.OpenFile(installer.scriptPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := installer.scriptGenerator(scriptFile); err != nil {
		t.Fatal(err)
	}
	if err := scriptFile.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := installer.postInstall(home, installer.scriptPath); err != nil {
		t.Fatalf("postInstall returned error: %v", err)
	}
	if _, err := installer.postInstall(home, installer.scriptPath); err != nil {
		t.Fatalf("second postInstall returned error: %v", err)
	}

	scriptContent, err := os.ReadFile(installer.scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(scriptContent), "#compdef arbor") {
		t.Fatalf("unexpected zsh completion script: %s", string(scriptContent))
	}

	zshrcContent, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(zshrcContent)
	if strings.Count(got, "# arbor completion") != 1 {
		t.Fatalf("expected exactly one Arbor block in .zshrc:\n%s", got)
	}
	if !strings.Contains(got, "autoload -Uz compinit") {
		t.Fatalf("expected compinit setup in .zshrc:\n%s", got)
	}
	if !strings.Contains(got, "fpath=(") {
		t.Fatalf("expected fpath setup in .zshrc:\n%s", got)
	}
	if !strings.Contains(filepath.ToSlash(got), filepath.ToSlash(filepath.Base(filepath.Dir(installer.scriptPath)))) {
		t.Fatalf("expected completions directory reference in .zshrc:\n%s", got)
	}
}

func TestBashCompletionInstallerUsesCustomPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	customPath := filepath.Join(home, "completions", "arbor.bash")
	root := NewRootCommand(IOStreams{In: bytes.NewBuffer(nil), Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}})
	installer, err := buildCompletionInstaller(root, completionShellBash, customPath, false)
	if err != nil {
		t.Fatalf("buildCompletionInstaller returned error: %v", err)
	}

	if _, err := installer.postInstall(home, installer.scriptPath); err != nil {
		t.Fatalf("postInstall returned error: %v", err)
	}

	bashrcContent, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(bashrcContent)
	if !strings.Contains(got, "source ") {
		t.Fatalf("expected source line in .bashrc:\n%s", got)
	}
	if !strings.Contains(filepath.ToSlash(got), "arbor.bash") {
		t.Fatalf("expected custom completion path in .bashrc:\n%s", string(bashrcContent))
	}
}

func TestUnknownCommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	err := Execute([]string{"wat"}, IOStreams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut})
	if err == nil || !strings.Contains(err.Error(), `unknown command "wat"`) {
		t.Fatalf("expected unknown command error, got %v", err)
	}
}

func TestConfigValidateCommand(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".arbor.yaml")
	if err := os.WriteFile(configPath, []byte(`
defaults:
  trusted_auto_run: true
commands:
  - id: bootstrap
    label: Bootstrap
    command: bun install
presets:
  default:
    commands: [bootstrap]
    auto_run: true
`), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := Execute([]string{"--config", configPath, "config", "validate"}, IOStreams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(out.String(), "config valid") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestPersistentFlagInheritedBySubcommandHelp(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	if err := Execute([]string{"create", "--help"}, IOStreams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "--config") {
		t.Fatalf("expected inherited --config flag in help output:\n%s", got)
	}
	if strings.Contains(got, "--batch") {
		t.Fatalf("did not expect removed --batch flag in help output:\n%s", got)
	}
}

func TestDetectCommandOutput(t *testing.T) {
	root := initRepo(t)
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("A=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"dev":"vite"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	runInDir(t, root, func() {
		if err := Execute([]string{"detect"}, IOStreams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})
	if !strings.Contains(out.String(), "env files:") || !strings.Contains(out.String(), "commands:") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestCreateCommandPlanningOutput(t *testing.T) {
	root := initRepo(t)
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("A=1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"dev":"vite"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	runInDir(t, root, func() {
		if err := Execute([]string{"create", "feature-auth", "--non-interactive", "--plan"}, IOStreams{
			In:     bytes.NewBuffer(nil),
			Out:    &out,
			ErrOut: &errOut,
		}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})
	if !strings.Contains(out.String(), "planning only") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestCreateCommandPlanningOutputWithExistingBranch(t *testing.T) {
	root := initRepo(t)
	runGit(t, root, "branch", "feature/auth", "HEAD")

	var out bytes.Buffer
	var errOut bytes.Buffer
	runInDir(t, root, func() {
		if err := Execute([]string{"create", "--branch", "feature/auth", "--non-interactive", "--plan"}, IOStreams{
			In:     bytes.NewBuffer(nil),
			Out:    &out,
			ErrOut: &errOut,
		}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})
	if !strings.Contains(out.String(), "branch mode: existing") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestCreateCommandPlanningOutputUsesConfigDefaultsWithoutInteractiveAnswers(t *testing.T) {
	root := initRepo(t)
	if err := os.WriteFile(filepath.Join(root, ".arbor.yaml"), []byte(`
env_files:
  - id: env
    label: Primary env
    source_path: .env
    target_path: .env
    default_action: copy
commands:
  - id: install
    label: Install deps
    command: pnpm install
presets:
  default:
    commands: [install]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("A=1"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	runInDir(t, root, func() {
		if err := Execute([]string{"create", "feature-auth", "--plan"}, IOStreams{
			In:     bytes.NewBufferString(""),
			Out:    &out,
			ErrOut: &errOut,
		}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	got := out.String()
	if !strings.Contains(got, "env actions: .env=copy") {
		t.Fatalf("expected config-driven env action in output: %s", got)
	}
	if !strings.Contains(got, "commands: Install deps=run") {
		t.Fatalf("expected config-driven command approval in output: %s", got)
	}
}

func TestCreateCommandExecutes(t *testing.T) {
	root := initRepo(t)
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("A=1"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	runInDir(t, root, func() {
		if err := Execute([]string{"create", "feature-live", "--non-interactive"}, IOStreams{
			In:     bytes.NewBuffer(nil),
			Out:    &out,
			ErrOut: &errOut,
		}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})
	if !strings.Contains(out.String(), "created: true") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestCreateCommandExecutesWithExistingBranch(t *testing.T) {
	root := initRepo(t)
	runGit(t, root, "branch", "feature/auth", "HEAD")

	var out bytes.Buffer
	var errOut bytes.Buffer
	runInDir(t, root, func() {
		if err := Execute([]string{"create", "review-auth", "--branch", "feature/auth", "--non-interactive"}, IOStreams{
			In:     bytes.NewBuffer(nil),
			Out:    &out,
			ErrOut: &errOut,
		}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})
	if !strings.Contains(out.String(), "created: true") {
		t.Fatalf("unexpected output: %s", out.String())
	}
	if !strings.Contains(out.String(), "worktree feature/auth") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestCreateCommandRejectsMultipleNames(t *testing.T) {
	root := initRepo(t)

	var out bytes.Buffer
	var errOut bytes.Buffer
	var runErr error
	runInDir(t, root, func() {
		runErr = Execute([]string{"create", "a", "b", "--non-interactive"}, IOStreams{
			In:     bytes.NewBuffer(nil),
			Out:    &out,
			ErrOut: &errOut,
		})
	})
	if runErr == nil || !strings.Contains(runErr.Error(), "can only create one worktree at a time") {
		t.Fatalf("expected single-worktree error, got %v", runErr)
	}
}

func TestInitCommandCreatesConfig(t *testing.T) {
	root := t.TempDir()
	var out bytes.Buffer
	var errOut bytes.Buffer
	runInDir(t, root, func() {
		if err := Execute([]string{"init"}, IOStreams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})
	if _, err := os.Stat(filepath.Join(root, ".arbor.yaml")); err != nil {
		t.Fatalf("expected config file: %v", err)
	}
}

func TestConfigRootCommandPrintsConfig(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, ".arbor.yaml")
	if err := os.WriteFile(configPath, []byte("defaults:\n  env_action: symlink\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	if err := Execute([]string{"--config", configPath, "config"}, IOStreams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !strings.Contains(out.String(), "defaults:") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestPullCommandFromLinkedWorktree(t *testing.T) {
	mainRoot, remoteRoot := initRepoWithRemote(t)
	worktreePath := filepath.Join(filepath.Dir(mainRoot), "repo-feature")
	runGit(t, mainRoot, "worktree", "add", worktreePath, "-b", "feature/test")
	pushCommitToRemote(t, remoteRoot, "remote update\n")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	runInDir(t, worktreePath, func() {
		if err := Execute([]string{"pull"}, IOStreams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	gotCWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if gotCWD != cwd {
		t.Fatalf("got cwd %q want %q", gotCWD, cwd)
	}
	if got := pullPathFromOutput(t, out.String(), "pulled main worktree: "); got != normalizedPath(t, mainRoot) {
		t.Fatalf("unexpected output: %s", out.String())
	}
	if got := readFile(t, filepath.Join(mainRoot, "README.md")); !strings.Contains(got, "remote update") {
		t.Fatalf("expected main worktree to receive pulled change, got: %s", got)
	}
}

func TestPullCommandSkipsDirtyMainWorktree(t *testing.T) {
	mainRoot, _ := initRepoWithRemote(t)
	worktreePath := filepath.Join(filepath.Dir(mainRoot), "repo-feature")
	runGit(t, mainRoot, "worktree", "add", worktreePath, "-b", "feature/test")
	if err := os.WriteFile(filepath.Join(mainRoot, "dirty.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	runInDir(t, worktreePath, func() {
		if err := Execute([]string{"pull"}, IOStreams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	gotCWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if gotCWD != cwd {
		t.Fatalf("got cwd %q want %q", gotCWD, cwd)
	}
	if got := pullPathFromOutput(t, out.String(), "skipped pull: main worktree has local changes at "); got != normalizedPath(t, mainRoot) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestPullCommandFromMainWorktree(t *testing.T) {
	mainRoot, remoteRoot := initRepoWithRemote(t)
	pushCommitToRemote(t, remoteRoot, "remote update\n")

	var out bytes.Buffer
	var errOut bytes.Buffer
	runInDir(t, mainRoot, func() {
		if err := Execute([]string{"pull"}, IOStreams{In: bytes.NewBuffer(nil), Out: &out, ErrOut: &errOut}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	if got := pullPathFromOutput(t, out.String(), "pulled main worktree: "); got != normalizedPath(t, mainRoot) {
		t.Fatalf("unexpected output: %s", out.String())
	}
	if got := readFile(t, filepath.Join(mainRoot, "README.md")); !strings.Contains(got, "remote update") {
		t.Fatalf("expected main worktree to receive pulled change, got: %s", got)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "init")
	return root
}

func initRepoWithRemote(t *testing.T) (string, string) {
	t.Helper()

	remoteRoot := t.TempDir()
	runGit(t, remoteRoot, "init", "--bare")

	mainRoot := initRepo(t)
	runGit(t, mainRoot, "remote", "add", "origin", remoteRoot)
	runGit(t, mainRoot, "branch", "-M", "main")
	runGit(t, mainRoot, "push", "-u", "origin", "main")
	runGit(t, remoteRoot, "symbolic-ref", "HEAD", "refs/heads/main")

	return mainRoot, remoteRoot
}

func pushCommitToRemote(t *testing.T, remoteRoot string, content string) {
	t.Helper()

	cloneRoot := t.TempDir()
	runGit(t, cloneRoot, "clone", remoteRoot, ".")
	runGit(t, cloneRoot, "config", "user.email", "test@example.com")
	runGit(t, cloneRoot, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(cloneRoot, "README.md"), []byte("hello\n"+content), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, cloneRoot, "add", "README.md")
	runGit(t, cloneRoot, "commit", "-m", "remote update")
	runGit(t, cloneRoot, "push", "origin", "main")
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func normalizedPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.ToSlash(filepath.Clean(resolved))
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func pullPathFromOutput(t *testing.T, output, prefix string) string {
	t.Helper()
	normalized := filepath.ToSlash(strings.TrimSpace(output))
	idx := strings.Index(normalized, prefix)
	if idx == -1 {
		t.Fatalf("output missing prefix %q: %s", prefix, output)
	}
	return normalizedPath(t, normalized[idx+len(prefix):])
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func runInDir(t *testing.T, dir string, fn func()) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatal(err)
		}
	}()
	fn()
}
