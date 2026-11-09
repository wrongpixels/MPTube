// Package diaglog writes a bounded local progress diagnostic file.
// It never sends data over the network and strips URLs, paths, and secrets.
package diaglog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"youtubemp3/internal/apppaths"
)

const (
	latestName   = "progress-latest.txt"
	previousName = "progress-previous.txt"
	maxBytes     = 256 * 1024
)

var (
	urlRe   = regexp.MustCompile(`(?i)\bhttps?://\S+`)
	uncRe   = regexp.MustCompile(`\\\\[^\s]+`)
	winRe   = regexp.MustCompile(`(?i)[A-Za-z]:\\[^\s]+`)
	nixRe   = regexp.MustCompile(`(?i)(?:/home/|/Users/|/tmp/|/var/|/usr/)[^\s]+`)
	authRe  = regexp.MustCompile(`(?i)(authorization|cookie|set-cookie|token|passwd|password|api[_-]?key)=[^\s]+`)
	queryRe = regexp.MustCompile(`\?[^\s]+`)
)

var (
	mu        sync.Mutex
	file      *os.File
	written   int
	stopped   bool
	startedAt time.Time
)

// Dir is %LOCALAPPDATA%\YouTubeMP3Saver\logs on Windows (UserCacheDir elsewhere).
func Dir() (string, error) {
	base, err := apppaths.CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "logs"), nil
}

// LatestPath is the current run's diagnostic file.
func LatestPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, latestName), nil
}

// Start rotates the previous log and opens a new one. Failures are ignored.
func Start(version string) {
	mu.Lock()
	defer mu.Unlock()
	closeLocked()
	dir, err := Dir()
	if err != nil {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	latest := filepath.Join(dir, latestName)
	prev := filepath.Join(dir, previousName)
	if _, err := os.Stat(latest); err == nil {
		_ = os.Remove(prev)
		_ = os.Rename(latest, prev)
	}
	f, err := os.OpenFile(latest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}
	file = f
	written = 0
	stopped = false
	startedAt = time.Now()
	writeLocked("event=start version=%s ts=%s\n", sanitize(version), startedAt.UTC().Format(time.RFC3339))
}

func Tool(id, version string) {
	Event("tool", "id=%s version=%s", sanitizeIdent(id), sanitize(version))
}

func Flags(hasTemplate, newline, colorNever, ignoreConfig bool) {
	Event("flags", "progress_template=%t newline=%t color_never=%t ignore_config=%t", hasTemplate, newline, colorNever, ignoreConfig)
}

func Event(name, format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()
	if file == nil || stopped {
		return
	}
	body := fmt.Sprintf(format, args...)
	writeLocked("event=%s %s\n", sanitizeIdent(name), sanitize(body))
}

// Progress records one parser/UI step. raw is sanitized; only prefixes and numeric fields are kept.
func Progress(raw string, kind string, determinate bool, percent int, speed, eta string, downloaded, total int64, hasPercent, hasTotal bool, statusMode string) {
	mu.Lock()
	defer mu.Unlock()
	if file == nil || stopped {
		return
	}
	prefix := progressPrefix(raw)
	writeLocked("event=progress prefix=%s kind=%s det=%t pct=%d has_pct=%t speed=%s eta=%s down=%d total=%d has_total=%t status=%s\n",
		prefix, sanitizeIdent(kind), determinate, percent, hasPercent, sanitizeIdent(speed), sanitizeIdent(eta), downloaded, total, hasTotal, sanitizeIdent(statusMode))
}

func End(ok bool, reason string) {
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return
	}
	writeLocked("event=end ok=%t reason=%s elapsed_ms=%d\n", ok, sanitizeIdent(reason), time.Since(startedAt).Milliseconds())
	closeLocked()
}

func closeLocked() {
	if file != nil {
		_ = file.Close()
		file = nil
	}
	stopped = true
}

func writeLocked(format string, args ...any) {
	if file == nil || stopped {
		return
	}
	line := fmt.Sprintf(format, args...)
	if written+len(line) > maxBytes {
		if !stopped {
			_, _ = file.WriteString("event=truncated\n")
			stopped = true
		}
		return
	}
	line = stampTMS(line)
	n, err := file.WriteString(line)
	written += n
	if err != nil {
		stopped = true
	}
}

func stampTMS(line string) string {
	ms := int64(0)
	if !startedAt.IsZero() {
		ms = time.Since(startedAt).Milliseconds()
	}
	hadNL := strings.HasSuffix(line, "\n")
	trimmed := strings.TrimSuffix(line, "\n")
	if strings.HasPrefix(trimmed, "event=") {
		if i := strings.IndexByte(trimmed, ' '); i > 0 {
			trimmed = trimmed[:i] + fmt.Sprintf(" t_ms=%d", ms) + trimmed[i:]
		} else {
			trimmed += fmt.Sprintf(" t_ms=%d", ms)
		}
	}
	if hadNL {
		return trimmed + "\n"
	}
	return trimmed
}

// UIApply records what the UI thread wrote after applying a footer model.
func UIApply(stage string, det bool, pct int, readDet bool, readPct int, statusTok string) {
	mu.Lock()
	defer mu.Unlock()
	if file == nil || stopped {
		return
	}
	writeLocked("event=ui_apply stage=%s det=%t pct=%d read_det=%t read_pct=%d status=%s\n",
		sanitizeIdent(stage), det, pct, readDet, readPct, sanitizeIdent(statusTok))
}

// Paint records a sampled progress-bar paint (not every pulse tick).
func Paint(det bool, pct int, active bool) {
	Event("paint", "det=%t pct=%d active=%t", det, pct, active)
}

func progressPrefix(raw string) string {
	s := sanitize(raw)
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	if strings.HasPrefix(s, "YTMP3") {
		fields := strings.Split(s, "\t")
		if len(fields) == 1 {
			fields = strings.Fields(s)
		}
		out := make([]string, 0, len(fields))
		for _, f := range fields {
			out = append(out, sanitizeIdent(f))
		}
		return strings.Join(out, ",")
	}
	if len(s) > 48 {
		s = s[:48]
	}
	return strings.ReplaceAll(s, " ", "_")
}

func sanitize(s string) string {
	s = urlRe.ReplaceAllString(s, "URL")
	s = queryRe.ReplaceAllString(s, "")
	s = authRe.ReplaceAllString(s, "${1}=REDACTED")
	s = uncRe.ReplaceAllString(s, "PATH")
	s = winRe.ReplaceAllString(s, "PATH")
	s = nixRe.ReplaceAllString(s, "PATH")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func sanitizeIdent(s string) string {
	s = sanitize(s)
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	var b strings.Builder
	for _, r := range s {
		if r > 127 || r < 32 {
			continue
		}
		switch r {
		case ' ', '\t', '\n', '\r':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	out := b.String()
	if out == "" {
		return "-"
	}
	if len(out) > 64 {
		return out[:64]
	}
	return out
}
