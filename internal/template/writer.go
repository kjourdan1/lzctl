package template

import (
	"fmt"
	"os"
	"path/filepath"
)

// Writer writes rendered files to disk.
type Writer struct {
	DryRun bool
}

// WriteAll writes all files under targetDir, creating parent directories.
// In dry-run mode, no file is written but the output paths are returned.
func (w Writer) WriteAll(files []RenderedFile, targetDir string) ([]string, error) {
	paths := make([]string, 0, len(files))
	for _, f := range files {
		fullPath := filepath.Join(targetDir, f.Path)
		paths = append(paths, fullPath)
		if w.DryRun {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return nil, fmt.Errorf("creating parent directory for %s: %w", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(f.Content), 0o644); err != nil {
			return nil, fmt.Errorf("writing file %s: %w", fullPath, err)
		}
	}
	return paths, nil
}
