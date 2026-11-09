package infocache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"youtubemp3/internal/apppaths"
)

// TTL is how long a cached yt-dlp info JSON may be reused.
// YouTube media URLs often last longer; this keeps selection data fresh.
const TTL = 8 * time.Minute

var (
	idFileRe    = regexp.MustCompile(`^[A-Za-z0-9_-]{11}\.json$`)
	expireRe    = regexp.MustCompile(`expire=(\d{10})`)
	dirOverride string
	renameFile  = os.Rename

	ErrNotJSON = errors.New("not_json")
	ErrMkdir   = errors.New("mkdir")
	ErrWrite   = errors.New("write")
	ErrRename  = errors.New("rename")

	memMu    sync.Mutex
	memRaw   = map[string]kept{}
	memSaved = map[string]string{}
)

type kept struct {
	raw []byte
	at  time.Time
}

func UseDir(dir string) { dirOverride = dir }

func Dir() (string, error) {
	if dirOverride != "" {
		return dirOverride, nil
	}
	base, err := apppaths.CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cache", "info"), nil
}

func ValidID(id string) bool {
	if len(id) != 11 {
		return false
	}
	for _, r := range id {
		if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func filePath(id string) (string, error) {
	if !ValidID(id) {
		return "", fmt.Errorf("invalid video id")
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

// SaveClass is a path-free token for diagnostics: ok, rename_fallback, not_json, mkdir, write, rename.
func SaveClass(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrNotJSON):
		return "not_json"
	case errors.Is(err, ErrMkdir):
		return "mkdir"
	case errors.Is(err, ErrWrite):
		return "write"
	case errors.Is(err, ErrRename):
		return "rename"
	default:
		return "write"
	}
}

// Remember records the preview save result for id. result must be a class token.
func Remember(id, result string) {
	if !ValidID(id) || result == "" {
		return
	}
	memMu.Lock()
	memSaved[id] = result
	memMu.Unlock()
}

// SavedResult is the last preview save class for id, or "" if this process did not try.
func SavedResult(id string) string {
	memMu.Lock()
	defer memMu.Unlock()
	return memSaved[id]
}

// Keep holds the preview JSON in this process so a failed disk write can be retried.
func Keep(id string, raw []byte) { keepRaw(id, raw) }

func keepRaw(id string, raw []byte) {
	if !ValidID(id) || !json.Valid(raw) {
		return
	}
	cp := append([]byte(nil), raw...)
	memMu.Lock()
	memRaw[id] = kept{raw: cp, at: time.Now()}
	memMu.Unlock()
}

func dropRaw(id string) {
	memMu.Lock()
	delete(memRaw, id)
	memMu.Unlock()
}

// Save writes full yt-dlp JSON (formats and media URLs) with 0600 perms.
// note is ok or rename_fallback. A Windows rename failure is followed by a direct write.
func Save(id string, raw []byte) (path, note string, err error) {
	if !json.Valid(raw) {
		return "", "not_json", ErrNotJSON
	}
	p, err := filePath(id)
	if err != nil {
		return "", "invalid_id", err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return "", "mkdir", fmt.Errorf("%w", ErrMkdir)
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return "", "write", fmt.Errorf("%w", ErrWrite)
	}
	if err := renameFile(tmp, p); err != nil {
		_ = os.Remove(tmp)
		if werr := os.WriteFile(p, raw, 0o600); werr != nil {
			return "", "rename", fmt.Errorf("%w", ErrRename)
		}
		Cleanup()
		return p, "rename_fallback", nil
	}
	Cleanup()
	return p, "ok", nil
}

// Lookup returns a reusable info JSON path. reason is missing, invalid, expired_ttl, expired_url, or ok.
func Lookup(id string) (path string, age time.Duration, reason string, ok bool) {
	p, err := filePath(id)
	if err != nil {
		return "", 0, "invalid", false
	}
	st, err := os.Stat(p)
	if err != nil {
		if path, age, reason, ok := restoreKept(id); ok || reason != "" {
			return path, age, reason, ok
		}
		return "", 0, "missing", false
	}
	age = time.Since(st.ModTime())
	raw, err := os.ReadFile(p)
	if err != nil {
		return "", age, "unreadable", false
	}
	if !json.Valid(raw) {
		_ = os.Remove(p)
		return "", age, "invalid", false
	}
	until, why := validUntil(raw, st.ModTime())
	if time.Now().After(until) {
		_ = os.Remove(p)
		return "", age, why, false
	}
	return p, age, "ok", true
}

func Remove(id string) {
	dropRaw(id)
	p, err := filePath(id)
	if err != nil {
		return
	}
	_ = os.Remove(p)
}

// restoreKept writes the in-process copy when the file is absent.
// An expired copy is not written and is not reported as missing.
func restoreKept(id string) (path string, age time.Duration, reason string, ok bool) {
	memMu.Lock()
	k, found := memRaw[id]
	memMu.Unlock()
	if !found {
		return "", 0, "", false
	}
	age = time.Since(k.at)
	until, why := validUntil(k.raw, k.at)
	if time.Now().After(until) {
		dropRaw(id)
		return "", age, why, false
	}
	p, _, err := Save(id, k.raw)
	if err != nil {
		return "", age, "missing", false
	}
	return p, age, "ok", true
}

func validUntil(raw []byte, savedAt time.Time) (time.Time, string) {
	hard := savedAt.Add(TTL)
	if exp, found := earliestExpire(raw); found {
		exp = exp.Add(-30 * time.Second)
		if exp.Before(hard) {
			return exp, "expired_url"
		}
	}
	return hard, "expired_ttl"
}

func earliestExpire(raw []byte) (time.Time, bool) {
	matches := expireRe.FindAllSubmatch(raw, -1)
	var earliest time.Time
	found := false
	for _, m := range matches {
		sec, err := strconv.ParseInt(string(m[1]), 10, 64)
		if err != nil || sec <= 0 {
			continue
		}
		t := time.Unix(sec, 0)
		if !found || t.Before(earliest) {
			earliest = t
			found = true
		}
	}
	return earliest, found
}

// Cleanup deletes only this app's expired <id>.json files.
func Cleanup() {
	dir, err := Dir()
	if err != nil {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !idFileRe.MatchString(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > TTL {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// ValidatePath ensures p is a file we created inside the info cache directory.
func ValidatePath(p string) error {
	if strings.TrimSpace(p) == "" {
		return fmt.Errorf("empty info path")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return err
	}
	dir, err := Dir()
	if err != nil {
		return err
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	sep := string(os.PathSeparator)
	prefix := absDir
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}
	if !strings.HasPrefix(abs, prefix) {
		return fmt.Errorf("info path outside cache")
	}
	if !idFileRe.MatchString(filepath.Base(abs)) {
		return fmt.Errorf("unexpected info file name")
	}
	return nil
}
