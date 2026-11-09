package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const MaxRecent = 15

var (
	ErrEmptyFolder       = errors.New("no output folder is set")
	ErrFolderUnavailable = errors.New("output folder is not available")
	ErrNotAFolder        = errors.New("output path is not a folder")
)

// Settings holds the last chosen destination and recent folders.
// Unavailable paths are retained exactly as stored.
type Settings struct {
	OutputFolder  string   `json:"output_folder"`
	RecentFolders []string `json:"recent_folders"`
	Language      string   `json:"language,omitempty"`
}

type Store struct {
	Path string
}

func Default() Settings {
	return Settings{
		OutputFolder:  "",
		RecentFolders: nil,
	}
}

func (s *Store) Load() (Settings, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Settings{}, err
	}
	var out Settings
	if err := json.Unmarshal(data, &out); err != nil {
		return Settings{}, err
	}
	out.RecentFolders = sanitizeRecentList(out.RecentFolders)
	if strings.TrimSpace(out.OutputFolder) != "" {
		out.RecentFolders = EnsureContained(out.RecentFolders, out.OutputFolder)
	}
	return out, nil
}

func (s *Store) Save(st Settings) error {
	if s.Path == "" {
		return errors.New("settings path is empty")
	}
	st.RecentFolders = sanitizeRecentList(st.RecentFolders)
	if strings.TrimSpace(st.OutputFolder) != "" {
		st.RecentFolders = EnsureContained(st.RecentFolders, st.OutputFolder)
	}
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}

func sanitizeRecentList(in []string) []string {
	var out []string
	for _, p := range in {
		if strings.TrimSpace(p) == "" {
			continue
		}
		// Keep the first spelling of each logical path.
		dup := false
		for _, existing := range out {
			if PathEqual(existing, p) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, p)
		}
	}
	if len(out) > MaxRecent {
		out = out[:MaxRecent]
	}
	return out
}

// ApplyChosenFolder records the user's chosen destination.
// The exact path string is always stored as OutputFolder.
// Valid (existing) folders are promoted to the front of recents.
// Unavailable folders are retained or added; they are never dropped, reset, or rewritten.
func (st *Settings) ApplyChosenFolder(path string, available bool) {
	st.OutputFolder = path
	if strings.TrimSpace(path) == "" {
		return
	}
	if available {
		st.RecentFolders = Promote(st.RecentFolders, path)
		return
	}
	st.RecentFolders = EnsureContained(st.RecentFolders, path)
}

// Promote moves path to the front, de-duplicates, and caps the list.
func Promote(recents []string, path string) []string {
	if strings.TrimSpace(path) == "" {
		return sanitizeRecentList(recents)
	}
	out := []string{path}
	for _, r := range recents {
		if !PathEqual(r, path) {
			out = append(out, r)
		}
	}
	if len(out) > MaxRecent {
		out = out[:MaxRecent]
	}
	return out
}

// EnsureContained adds path if it is not already present.
// Existing entries keep their stored spelling and position.
// A brand-new path is inserted at the front so a disconnected drive is not lost.
func EnsureContained(recents []string, path string) []string {
	if strings.TrimSpace(path) == "" {
		return sanitizeRecentList(recents)
	}
	for _, r := range recents {
		if PathEqual(r, path) {
			return sanitizeRecentList(recents)
		}
	}
	out := append([]string{path}, recents...)
	return sanitizeRecentList(out)
}

// PathEqual compares Windows-style paths without changing stored values.
func PathEqual(a, b string) bool {
	return normalizeForCompare(a) == normalizeForCompare(b)
}

func normalizeForCompare(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "/", "\\")
	// Collapse repeated slashes but keep a leading UNC prefix.
	for strings.Contains(p, "\\\\") && !strings.HasPrefix(p, "\\\\") {
		p = strings.ReplaceAll(p, "\\\\", "\\")
	}
	if strings.HasPrefix(p, "\\\\") {
		rest := strings.TrimLeft(p[2:], "\\")
		p = "\\\\" + rest
	}
	if len(p) == 2 && p[1] == ':' {
		p += "\\"
	} else if len(p) > 3 || (len(p) == 3 && !(unicode.IsLetter(rune(p[0])) && p[1] == ':' && (p[2] == '\\'))) {
		p = strings.TrimRight(p, "\\")
		if len(p) == 2 && p[1] == ':' {
			p += "\\"
		}
	}
	return strings.ToLower(p)
}

// FolderAvailable reports whether path currently exists as a directory.
// It never creates, deletes, or rewrites the path.
func FolderAvailable(path string) error {
	if strings.TrimSpace(path) == "" {
		return ErrEmptyFolder
	}
	info, err := os.Stat(path)
	if err != nil {
		return unavailableError(path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotAFolder, path)
	}
	return nil
}

func unavailableError(path string, err error) error {
	return fmt.Errorf("%w: %s (%v)", ErrFolderUnavailable, path, err)
}

// UserFolderError formats a clear message and emphasizes that the saved path was kept.
func UserFolderError(path string, err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrEmptyFolder) {
		return "No output folder is set. Choose a folder to save MP3 files."
	}
	if errors.Is(err, ErrNotAFolder) {
		return "The selected path is not a folder:\n" + path + "\n\nThe saved path was not changed."
	}
	return "The selected folder is not available:\n" + path + "\n\nIt may be on a disconnected drive or unreachable network location.\nThe saved path was not changed."
}
