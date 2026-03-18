package model

import (
	"encoding/json"
	"testing"
)

func TestExecutionSummaryJSON(t *testing.T) {
	summary := ExecutionSummary{
		RepoRoot:   "/tmp/repo",
		OpenApp:    "cursor",
		OpenedPath: "/tmp/repo-feature",
		Worktrees: []WorktreeResult{
			{
				Path:    "/tmp/repo-feature",
				Branch:  "feature/test",
				Created: true,
				EnvResults: []EnvResult{
					{ID: "env", Action: ActionSymlink, Applied: true},
				},
			},
		},
	}

	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected JSON output")
	}
}

func TestActionConstants(t *testing.T) {
	if ActionSymlink == ActionCopy || ActionCopy == ActionSkip {
		t.Fatal("expected distinct actions")
	}
}
