package gitutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindRepoRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindRepoRoot(nested)
	if err != nil {
		t.Fatalf("FindRepoRoot returned error: %v", err)
	}
	if got != root {
		t.Fatalf("got %q want %q", got, root)
	}
}

func TestFindRepoRootNotRepo(t *testing.T) {
	_, err := FindRepoRoot(t.TempDir())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseWorktreeList(t *testing.T) {
	raw := strings.Join([]string{
		"worktree /tmp/repo",
		"HEAD abc123",
		"branch refs/heads/main",
		"",
		"worktree /tmp/repo-feature",
		"HEAD def456",
		"branch refs/heads/feature/test",
		"locked reason",
		"",
	}, "\n")

	worktrees, err := ParseWorktreeList(raw)
	if err != nil {
		t.Fatalf("ParseWorktreeList returned error: %v", err)
	}
	if len(worktrees) != 2 {
		t.Fatalf("got %d worktrees", len(worktrees))
	}
	if worktrees[1].Branch != "feature/test" {
		t.Fatalf("unexpected branch: %#v", worktrees[1])
	}
	if !worktrees[1].Locked {
		t.Fatalf("expected second worktree locked")
	}
}

func TestDiscoverRepo(t *testing.T) {
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

	state, err := DiscoverRepo(context.Background(), root)
	if err != nil {
		t.Fatalf("DiscoverRepo returned error: %v", err)
	}
	if state.Root != root {
		t.Fatalf("got root %q want %q", state.Root, root)
	}
	if state.CurrentCommit == "" {
		t.Fatal("expected current commit")
	}
	if len(state.Worktrees) == 0 {
		t.Fatal("expected at least one worktree")
	}
}

func TestMainWorktreePath(t *testing.T) {
	runner := Runner{Dir: t.TempDir()}
	path, err := runner.MainWorktreePath(context.Background())
	if err == nil || !strings.Contains(err.Error(), "git worktree list --porcelain failed") {
		t.Fatalf("expected git command error, got path=%q err=%v", path, err)
	}
}

func TestMainWorktreePathReturnsFirstEntry(t *testing.T) {
	worktrees, err := ParseWorktreeList(strings.Join([]string{
		"worktree /tmp/repo",
		"HEAD abc123",
		"branch refs/heads/main",
		"",
		"worktree /tmp/repo-feature",
		"HEAD def456",
		"branch refs/heads/feature/test",
		"",
	}, "\n"))
	if err != nil {
		t.Fatalf("ParseWorktreeList returned error: %v", err)
	}
	if got := worktrees[0].Path; got != "/tmp/repo" {
		t.Fatalf("got %q want %q", got, "/tmp/repo")
	}
}

func TestParseWorktreeListEmpty(t *testing.T) {
	worktrees, err := ParseWorktreeList("")
	if err != nil {
		t.Fatalf("ParseWorktreeList returned error: %v", err)
	}
	if len(worktrees) != 0 {
		t.Fatalf("got %d worktrees", len(worktrees))
	}
}

func TestRunnerIsDirty(t *testing.T) {
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

	runner := Runner{Dir: root}
	dirty, err := runner.IsDirty(context.Background())
	if err != nil {
		t.Fatalf("IsDirty returned error: %v", err)
	}
	if dirty {
		t.Fatal("expected clean repo")
	}

	if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirty, err = runner.IsDirty(context.Background())
	if err != nil {
		t.Fatalf("IsDirty returned error: %v", err)
	}
	if !dirty {
		t.Fatal("expected repo with untracked file to be dirty")
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
