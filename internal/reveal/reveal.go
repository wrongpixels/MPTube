package reveal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrUnavailable = errors.New("downloaded file is not available")

// CheckFile reports whether path is an existing regular file.
func CheckFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return ErrUnavailable
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return ErrUnavailable
	}
	return nil
}

// ExplorerSelectArgs is the no-shell fallback argv that selects path in Explorer.
// The path is part of a single /select, argument so spaces need no extra quotes.
func ExplorerSelectArgs(path string) ([]string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "" || cleaned == "." {
		return nil, fmt.Errorf("empty path")
	}
	return []string{"explorer.exe", "/select," + cleaned}, nil
}
