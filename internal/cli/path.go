package cli

import "path/filepath"

func filepathJoin(root, value string) string {
	if value == "" {
		return root
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(root, value)
}
