package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	got, err := RenderTemplate("feature/{{ .Name }}-{{ .Index }}", TemplateData{Name: "auth", Index: 2})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "feature/auth-2" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateMissingKey(t *testing.T) {
	_, err := RenderTemplate("{{ .Missing }}", TemplateData{Name: "auth"})
	if err == nil || !strings.Contains(err.Error(), "map has no entry") && !strings.Contains(err.Error(), "can't evaluate field Missing") {
		t.Fatalf("expected missing key error, got %v", err)
	}
}

func TestExpandNamesRejectsDuplicates(t *testing.T) {
	_, err := ExpandNames("same", []string{"a", "b"}, "repo")
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestExpandNames(t *testing.T) {
	got, err := ExpandNames("{{ .Repo }}/{{ .Name }}", []string{"a", "b"}, "arbor")
	if err != nil {
		t.Fatalf("ExpandNames returned error: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"arbor/a", "arbor/b"}) {
		t.Fatalf("unexpected values: %#v", got)
	}
}
