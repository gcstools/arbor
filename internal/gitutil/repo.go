package gitutil

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNotGitRepository = errors.New("not a git repository")

type Runner struct {
	Dir string
}

type RepoState struct {
	Root          string
	CurrentRef    string
	CurrentCommit string
	LocalBranches []string
	Worktrees     []Worktree
}

type Worktree struct {
	Path     string
	Head     string
	Branch   string
	Bare     bool
	Locked   bool
	Prunable bool
}

func DiscoverRepo(ctx context.Context, start string) (RepoState, error) {
	root, err := FindRepoRoot(start)
	if err != nil {
		return RepoState{}, err
	}

	runner := Runner{Dir: root}
	ref, err := runner.CurrentRef(ctx)
	if err != nil {
		return RepoState{}, err
	}
	commit, err := runner.CurrentCommit(ctx)
	if err != nil {
		return RepoState{}, err
	}
	branches, err := runner.LocalBranches(ctx)
	if err != nil {
		return RepoState{}, err
	}
	worktrees, err := runner.ListWorktrees(ctx)
	if err != nil {
		return RepoState{}, err
	}

	return RepoState{
		Root:          root,
		CurrentRef:    ref,
		CurrentCommit: commit,
		LocalBranches: branches,
		Worktrees:     worktrees,
	}, nil
}

func FindRepoRoot(start string) (string, error) {
	if start == "" {
		start = "."
	}

	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		dir = filepath.Dir(dir)
	}

	for {
		candidate := filepath.Join(dir, ".git")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotGitRepository
		}
		dir = parent
	}
}

func (r Runner) CurrentRef(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err == nil {
		return strings.TrimSpace(out), nil
	}

	out, detachedErr := r.run(ctx, "rev-parse", "--short", "HEAD")
	if detachedErr != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r Runner) CurrentCommit(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r Runner) ListWorktrees(ctx context.Context) ([]Worktree, error) {
	out, err := r.run(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return ParseWorktreeList(out)
}

func (r Runner) MainWorktreePath(ctx context.Context) (string, error) {
	worktrees, err := r.ListWorktrees(ctx)
	if err != nil {
		return "", err
	}
	if len(worktrees) == 0 {
		return "", fmt.Errorf("no worktrees found")
	}
	return worktrees[0].Path, nil
}

func (r Runner) BranchExists(ctx context.Context, branch string) (bool, error) {
	out, err := r.run(ctx, "branch", "--list", branch)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (r Runner) LocalBranches(ctx context.Context) ([]string, error) {
	out, err := r.run(ctx, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	branches := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		branches = append(branches, line)
	}
	return branches, nil
}

func (r Runner) CreateBranch(ctx context.Context, branch string, baseRef string) error {
	_, err := r.run(ctx, "branch", branch, baseRef)
	return err
}

func (r Runner) AddWorktree(ctx context.Context, path string, branch string) error {
	_, err := r.run(ctx, "worktree", "add", path, branch)
	return err
}

func (r Runner) IsDirty(ctx context.Context) (bool, error) {
	out, err := r.run(ctx, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (r Runner) Pull(ctx context.Context) error {
	_, err := r.run(ctx, "pull")
	return err
}

func ParseWorktreeList(raw string) ([]Worktree, error) {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	var worktrees []Worktree
	var current *Worktree

	flush := func() {
		if current != nil {
			worktrees = append(worktrees, *current)
			current = nil
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			flush()
			continue
		}

		key, value, found := strings.Cut(line, " ")
		if !found {
			key = line
			value = ""
		}

		switch key {
		case "worktree":
			flush()
			current = &Worktree{Path: value}
		case "HEAD":
			if current == nil {
				return nil, fmt.Errorf("HEAD before worktree entry")
			}
			current.Head = value
		case "branch":
			if current == nil {
				return nil, fmt.Errorf("branch before worktree entry")
			}
			current.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "bare":
			if current == nil {
				return nil, fmt.Errorf("bare before worktree entry")
			}
			current.Bare = true
		case "locked":
			if current == nil {
				return nil, fmt.Errorf("locked before worktree entry")
			}
			current.Locked = true
		case "prunable":
			if current == nil {
				return nil, fmt.Errorf("prunable before worktree entry")
			}
			current.Prunable = true
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	flush()
	return worktrees, nil
}

func (r Runner) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.Dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("git not installed: %w", err)
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}
