package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	for _, want := range []string{"arbor ", "commit:", "built:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("version output missing %q:\n%s", want, got)
		}
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
	if !strings.Contains(normalizeForAssertion(out.String()), "pulled main worktree: "+normalizedPath(t, mainRoot)) {
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
	if !strings.Contains(normalizeForAssertion(out.String()), "skipped pull: main worktree has local changes at "+normalizedPath(t, mainRoot)) {
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

	if !strings.Contains(normalizeForAssertion(out.String()), "pulled main worktree: "+normalizedPath(t, mainRoot)) {
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
	cleaned := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return filepath.ToSlash(cleaned)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return cleaned
	}
	return filepath.Clean(resolved)
}

func normalizeForAssertion(value string) string {
	return filepath.ToSlash(value)
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
