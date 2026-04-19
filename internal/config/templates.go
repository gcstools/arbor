package config

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"
)

type TemplateData struct {
	Prefix string
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

	rewritten, err := rewriteTemplateSet(raw)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var out bytes.Buffer
	if err := rewritten.Execute(&out, data); err != nil {
		return "", fmt.Errorf("render template: %w", err)
	}

	value := strings.TrimSpace(out.String())
	if value == "" {
		return "", fmt.Errorf("template rendered empty value")
	}
	return value, nil
}

func rewriteTemplateSet(raw string) (*template.Template, error) {
	tmpl, err := template.New("value").Parse(raw)
	if err != nil {
		return nil, err
	}

	sources := make(map[string]string)
	for _, t := range tmpl.Templates() {
		if t == nil || t.Tree == nil || t.Tree.Root == nil {
			continue
		}
		sources[t.Name()] = serializeNodes(t.Tree.Root.Nodes)
	}

	rewritten := template.New(tmpl.Name()).Option("missingkey=error")
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		src := sources[name]
		if name == tmpl.Name() {
			if _, err := rewritten.Parse(src); err != nil {
				return nil, err
			}
			continue
		}
		if _, err := rewritten.New(name).Parse(src); err != nil {
			return nil, err
		}
	}
	return rewritten, nil
}

func serializeNodes(nodes []parse.Node) string {
	var out bytes.Buffer
	var skipLeadingSeparator byte

	for i := 0; i < len(nodes); i++ {
		switch n := nodes[i].(type) {
		case *parse.TextNode:
			text := n.Text
			if skipLeadingSeparator != 0 && len(text) > 0 && text[0] == skipLeadingSeparator {
				text = text[1:]
				skipLeadingSeparator = 0
			}
			out.Write(text)
		case *parse.ActionNode:
			if sep, ok := rewritePrefixAction(nodes, i, &out); ok {
				skipLeadingSeparator = sep
				continue
			}
			out.WriteString(n.String())
		case *parse.CommentNode:
			out.WriteString(n.String())
		case *parse.ListNode:
			out.WriteString(serializeNodes(n.Nodes))
		case *parse.IfNode:
			serializeBranch(&out, "if", &n.BranchNode)
		case *parse.RangeNode:
			serializeBranch(&out, "range", &n.BranchNode)
		case *parse.WithNode:
			serializeBranch(&out, "with", &n.BranchNode)
		case *parse.TemplateNode:
			out.WriteString("{{template ")
			out.WriteString(strconv.Quote(n.Name))
			if n.Pipe != nil {
				out.WriteByte(' ')
				out.WriteString(n.Pipe.String())
			}
			out.WriteString("}}")
		default:
			out.WriteString(n.String())
		}
	}

	return out.String()
}

func serializeBranch(out *bytes.Buffer, keyword string, node *parse.BranchNode) {
	out.WriteString("{{")
	out.WriteString(keyword)
	if node.Pipe != nil {
		out.WriteByte(' ')
		out.WriteString(node.Pipe.String())
	}
	out.WriteString("}}")
	if node.List != nil {
		out.WriteString(serializeNodes(node.List.Nodes))
	}
	if node.ElseList != nil {
		out.WriteString("{{else}}")
		out.WriteString(serializeNodes(node.ElseList.Nodes))
	}
	out.WriteString("{{end}}")
}

func rewritePrefixAction(nodes []parse.Node, index int, out *bytes.Buffer) (byte, bool) {
	action, ok := nodes[index].(*parse.ActionNode)
	if !ok || !isPrefixAction(action) {
		return 0, false
	}

	if index+1 < len(nodes) {
		if nextText, ok := nodes[index+1].(*parse.TextNode); ok && len(nextText.Text) > 0 && isSeparator(nextText.Text[0]) {
			nextSep := nextText.Text[0]
			if index > 0 {
				if prevText, ok := nodes[index-1].(*parse.TextNode); ok && len(prevText.Text) > 0 && isSeparator(prevText.Text[len(prevText.Text)-1]) {
					prevSep := prevText.Text[len(prevText.Text)-1]
					if out.Len() > 0 && out.Bytes()[out.Len()-1] == prevSep {
						out.Truncate(out.Len() - 1)
						out.WriteString("{{if .Prefix}}")
						out.WriteByte(prevSep)
						out.WriteString(action.String())
						out.WriteString("{{end}}")
						return 0, true
					}
				}
			}

			out.WriteString("{{if .Prefix}}")
			out.WriteString(action.String())
			out.WriteByte(nextSep)
			out.WriteString("{{end}}")
			return nextSep, true
		}
	}

	if index > 0 {
		if prevText, ok := nodes[index-1].(*parse.TextNode); ok && len(prevText.Text) > 0 && isSeparator(prevText.Text[len(prevText.Text)-1]) {
			sep := prevText.Text[len(prevText.Text)-1]
			if out.Len() > 0 && out.Bytes()[out.Len()-1] == sep {
				out.Truncate(out.Len() - 1)
				out.WriteString("{{if .Prefix}}")
				out.WriteByte(sep)
				out.WriteString(action.String())
				out.WriteString("{{end}}")
				return 0, true
			}
		}
	}

	return 0, false
}

func isPrefixAction(node *parse.ActionNode) bool {
	if node == nil || node.Pipe == nil || len(node.Pipe.Cmds) != 1 {
		return false
	}

	cmd := node.Pipe.Cmds[0]
	if len(cmd.Args) != 1 {
		return false
	}

	field, ok := cmd.Args[0].(*parse.FieldNode)
	if !ok || len(field.Ident) != 1 {
		return false
	}

	return field.Ident[0] == "Prefix"
}

func isSeparator(b byte) bool {
	return b == '/' || b == '-'
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
