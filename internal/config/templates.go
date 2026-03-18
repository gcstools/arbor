package config

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
)

type TemplateData struct {
	Name   string
	Index  int
	Branch string
	Base   string
	Repo   string
}

func RenderTemplate(raw string, data TemplateData) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("template is empty")
	}

	tmpl, err := template.New("value").Option("missingkey=error").Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}

	value := strings.TrimSpace(out.String())
	if value == "" {
		return "", fmt.Errorf("template rendered empty value")
	}
	return value, nil
}

func ExpandNames(raw string, names []string, repo string) ([]string, error) {
	values := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for i, name := range names {
		value, err := RenderTemplate(raw, TemplateData{
			Name:  name,
			Index: i + 1,
			Repo:  repo,
		})
		if err != nil {
			return nil, err
		}
		if _, exists := seen[value]; exists {
			return nil, fmt.Errorf("template produced duplicate value %q", value)
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values, nil
}

func CleanWorktreePath(path string) string {
	return filepath.Clean(path)
}
