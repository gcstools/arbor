package config

import (
	"reflect"
	"strings"
	"testing"
)

const oldEmptyPrefixMarker = "__TEMPLATE_EMPTY_PREFIX__"

func TestRenderTemplate(t *testing.T) {
	got, err := RenderTemplate("feature/{{ .Prefix }}/{{ .Name }}-{{ .Index }}", TemplateData{Prefix: "api", Name: "auth", Index: 2})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "feature/api/auth-2" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateCompactPrefixAction(t *testing.T) {
	got, err := RenderTemplate("{{.Prefix}}/{{.Name}}", TemplateData{Name: "name"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "name" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateTrimmedPrefixAction(t *testing.T) {
	got, err := RenderTemplate("{{- .Prefix -}}/{{ .Name }}", TemplateData{Name: "name"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "name" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateEmptyPrefixRemovesLeadingSlash(t *testing.T) {
	got, err := RenderTemplate("{{ .Prefix }}/{{ .Name }}", TemplateData{Name: "auth"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "auth" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateEmptyPrefixRemovesSingleDash(t *testing.T) {
	got, err := RenderTemplate("{{ .Prefix }}-{{ .Name }}", TemplateData{Name: "auth"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "auth" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateDoesNotCollapseUnrelatedSeparators(t *testing.T) {
	got, err := RenderTemplate("path//{{ .Prefix }}/{{ .Name }}", TemplateData{Name: "auth"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "path//auth" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateEmptyPrefixRemovesSeparatorBeforePrefix(t *testing.T) {
	got, err := RenderTemplate("{{ .Name }}/{{ .Prefix }}", TemplateData{Name: "auth"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "auth" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateEmptyPrefixKeepsNameLiteralMarker(t *testing.T) {
	got, err := RenderTemplate("{{ .Prefix }}/{{ .Name }}", TemplateData{Name: oldEmptyPrefixMarker})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != oldEmptyPrefixMarker {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateLiteralMarkerPrefixRendersWhenNonEmpty(t *testing.T) {
	got, err := RenderTemplate("{{ .Prefix }}/{{ .Name }}", TemplateData{Prefix: oldEmptyPrefixMarker, Name: "auth"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != oldEmptyPrefixMarker+"/auth" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateLiteralPrefixTextInsideAction(t *testing.T) {
	got, err := RenderTemplate(`{{ printf "%s" "{{ .Prefix }}" }}`, TemplateData{Name: "auth"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "{{ .Prefix }}" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplatePreservesDefinedHelperTemplate(t *testing.T) {
	got, err := RenderTemplate(`{{define "helper"}}{{.Prefix}}/{{.Name}}{{end}}{{template "helper" .}}`, TemplateData{Name: "auth"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "auth" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateRepeatedPrefixActionsEmptyPrefix(t *testing.T) {
	got, err := RenderTemplate("{{.Name}}/{{.Prefix}}/{{.Prefix}}/{{.Name}}", TemplateData{Name: "auth"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "auth/auth" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderTemplateRepeatedPrefixActionsNonEmptyPrefix(t *testing.T) {
	got, err := RenderTemplate("{{.Prefix}}/{{.Prefix}}/{{.Name}}", TemplateData{Prefix: "api", Name: "auth"})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if got != "api/api/auth" {
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
