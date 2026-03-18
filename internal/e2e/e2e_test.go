package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"arbor/internal/config"
	"arbor/internal/detect"
	"arbor/internal/execute"
	"arbor/internal/gitutil"
	"arbor/internal/planner"
)

func TestDetectAcrossFixtures(t *testing.T) {
	for _, fixture := range []struct {
		name         string
		fixture      string
		wantCommands int
		wantEnv      int
	}{
		{name: "node", fixture: "node", wantCommands: 2, wantEnv: 2},
		{name: "make", fixture: "make", wantCommands: 2, wantEnv: 1},
		{name: "just", fixture: "just", wantCommands: 2, wantEnv: 1},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			root := copyFixtureRepo(t, fixture.fixture)
			cfg, err := config.LoadOptional(filepath.Join(root, ".arbor.yaml"))
			if err != nil {
				t.Fatalf("LoadOptional returned error: %v", err)
			}
			result, err := detect.Scan(root, cfg)
			if err != nil {
				t.Fatalf("Scan returned error: %v", err)
			}
			if len(result.Commands) != fixture.wantCommands {
				t.Fatalf("got %d commands", len(result.Commands))
			}
			if len(result.EnvFiles) != fixture.wantEnv {
				t.Fatalf("got %d env files", len(result.EnvFiles))
			}
		})
	}
}

func TestSingleCreateEndToEnd(t *testing.T) {
	root := copyFixtureRepo(t, "config")
	initGitRepo(t, root)

	plan, err := planner.BuildCreatePlan(context.Background(), planner.Inputs{
		CWD:            root,
		Names:          []string{"feature-auth"},
		Preset:         "fast",
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err != nil {
		t.Fatalf("BuildCreatePlan returned error: %v", err)
	}

	summary, err := execute.Runner{}.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if summary.HasFailure {
		t.Fatalf("unexpected failure: %#v", summary)
	}
	worktreePath := summary.Worktrees[0].Path
	if _, err := os.Stat(filepath.Join(worktreePath, ".env")); err != nil {
		t.Fatalf("expected env file in worktree: %v", err)
	}
	if summary.Worktrees[0].CommandResult[0].Executed != true {
		t.Fatalf("expected trusted command to execute")
	}
}

func TestCreateRejectsMultipleNamesEndToEnd(t *testing.T) {
	root := copyFixtureRepo(t, "config")
	initGitRepo(t, root)

	_, err := planner.BuildCreatePlan(context.Background(), planner.Inputs{
		CWD:            root,
		Names:          []string{"api", "web"},
		NonInteractive: true,
	}, bytes.NewBuffer(nil), ".arbor.yaml")
	if err == nil || err.Error() != "exactly one worktree name is supported" {
		t.Fatalf("expected single-worktree error, got %v", err)
	}
}

func TestInitConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := config.StarterConfig()
	cfg.Path = filepath.Join(dir, ".arbor.yaml")

	data, err := cfg.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML returned error: %v", err)
	}
	if err := os.WriteFile(cfg.Path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(cfg.Path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded.Presets) == 0 || len(loaded.Commands) == 0 {
		t.Fatalf("expected starter config content")
	}
}

func TestWindowsAssumptionsOrUnixSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := symlinkCapableOrCopy(src, dst); err != nil {
		t.Fatalf("symlinkCapableOrCopy returned error: %v", err)
	}
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		if info.Mode()&os.ModeSymlink == 0 {
			data, err := os.ReadFile(dst)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "hello" {
				t.Fatalf("expected copied file contents, got %q", string(data))
			}
		}
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink on unix-like platforms")
	}
}

func copyFixtureRepo(t *testing.T, name string) string {
	t.Helper()
	dst := t.TempDir()
	src := filepath.Join("..", "..", "testdata", "fixtures", name)
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		copyFixtureEntry(t, filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()))
	}
	return dst
}

func copyFixtureEntry(t *testing.T, src, dst string) {
	t.Helper()
	info, err := os.Stat(src)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, 0o755); err != nil {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			copyFixtureEntry(t, filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()))
		}
		return
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func initGitRepo(t *testing.T, root string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	runGit(t, root, "init", "-b", "main")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "fixture")
	state, err := gitutil.DiscoverRepo(context.Background(), root)
	if err != nil || state.Root == "" {
		t.Fatalf("repo discovery failed: %v", err)
	}
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

func symlinkCapableOrCopy(src, dst string) error {
	if err := os.Symlink(src, dst); err == nil {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
