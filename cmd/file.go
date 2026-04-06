package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/oobagi/notebook-cli/internal/editor"
	"github.com/oobagi/notebook-cli/internal/recents"
)

// textFileExtensions lists file extensions treated as directly openable text files.
var textFileExtensions = []string{".md", ".txt"}

// isFilePath returns true if arg looks like a file path rather than a notebook
// name. It checks for path separators or known text file extensions.
func isFilePath(arg string) bool {
	if strings.ContainsAny(arg, "/\\") {
		return true
	}
	ext := strings.ToLower(filepath.Ext(arg))
	for _, e := range textFileExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

// openFile reads the file at path and opens it in the editor. The editor's
// save function writes content back to the same path, preserving the original
// file permissions.
func openFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("read file: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}
	originalMode := info.Mode().Perm()

	editorCfg := editor.Config{
		Title:       filepath.Base(absPath),
		FilePath:    absPath,
		FileSize:    info.Size(),
		Content:     string(data),
		Save: func(content string) error {
			if err := os.WriteFile(absPath, []byte(content), originalMode); err != nil {
				return err
			}
			recents.RecordExternal(absPath)
			return nil
		},
		HideChecked: cfg.HideChecked,
		WordWrap:    cfg.WordWrap,
	}

	m := editor.New(editorCfg)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run editor: %w", err)
	}
	return nil
}
