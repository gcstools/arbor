package docs

import (
	"os"
	"strings"
	"testing"
)

func TestREADMEContainsCoreCommands(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	for _, snippet := range []string{
		"go run ./cmd/arbor --help",
		"arbor create feature-auth --plan",
		"arbor config validate",
	} {
		if !strings.Contains(string(data), snippet) {
			t.Fatalf("README missing snippet %q", snippet)
		}
	}
}
