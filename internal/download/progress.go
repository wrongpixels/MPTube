package download

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Structured yt-dlp lines. The "download:" / "postprocess:" prefixes are
// selectors and are not printed on stdout.
//
// progress.percent is NOT a real field in yt-dlp 2026.10.26 (it prints "NA").
// progress._percent_str is the formatted value, e.g. " 15.9%".
const (
	DownloadProgressTemplate    = "download:YTMP3\t%(progress._percent_str)s\t%(progress.downloaded_bytes)s\t%(progress.total_bytes)s\t%(progress.total_bytes_estimate)s\t%(progress.speed)s\t%(progress.eta)s"
	PostprocessProgressTemplate = "postprocess:YTMP3POST"
)

// Progress is one parsed yt-dlp stdout line about download or conversion.
type Progress struct {
	Percent       int
	HasPercent    bool
	Speed         string
	ETA           string
	Downloaded    int64
	HasDownloaded bool
	Total         int64
	HasTotal      bool
	Post          bool
}

// View is the last applied download UI state. Unrecognized lines keep the previous view.
type View struct {
	Progress
	Kind string // downloading, bytes, converting, ""
}

var (
	progressRe   = regexp.MustCompile(`\[download\]\s+(\d+(?:\.\d+)?)%`)
	progressAtRe = regexp.MustCompile(`\bat\s+(\S+)`)
	progressETA  = regexp.MustCompile(`\bETA\s+(\S+)`)
	ansiRe       = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// ParseOutputLine understands the YTMP3 template, classic [download] lines,
// fragment/~estimate lines, and extract/convert markers.
func ParseOutputLine(line string) (Progress, bool) {
	line = stripANSI(line)
	line = strings.TrimRight(line, "\r")
	line = strings.TrimSpace(line)
	if line == "" {
		return Progress{}, false
	}
	if IsExtractLine(line) {
		return Progress{Post: true}, true
	}
	if strings.HasPrefix(line, "YTMP3\t") || strings.HasPrefix(line, "YTMP3 ") {
		return parseTemplateLine(line)
	}
	return parseClassicLine(line)
}

// UpdateView applies one subprocess line to the download UI state.
// Non-progress lines (metadata, destinations) leave the previous state alone.
func UpdateView(prev View, line string) View {
	upd, ok := ParseOutputLine(line)
	if !ok {
		return prev
	}
	if upd.Post {
		return View{Kind: "converting", Progress: Progress{Post: true}}
	}
	next := View{Progress: upd}
	if upd.HasPercent {
		next.Kind = "downloading"
		return next
	}
	next.Kind = "bytes"
	return next
}

// ParseProgressLine keeps the older percent-only API for callers and tests.
func ParseProgressLine(line string) (percent int, ok bool) {
	p, ok := ParseOutputLine(line)
	if !ok || p.Post || !p.HasPercent {
		return 0, false
	}
	return p.Percent, true
}

func IsExtractLine(line string) bool {
	s := strings.TrimSpace(line)
	if s == "YTMP3POST" || strings.HasPrefix(s, "YTMP3POST") {
		return true
	}
	if strings.Contains(s, "[ExtractAudio]") || strings.Contains(s, "Extracting audio") {
		return true
	}
	if strings.Contains(s, "[Merger]") || strings.Contains(s, "[Fixup") {
		return true
	}
	return false
}

func parseTemplateLine(line string) (Progress, bool) {
	body := strings.TrimPrefix(line, "YTMP3")
	body = strings.TrimLeft(body, "\t ")
	parts := strings.Split(body, "\t")
	if len(parts) < 6 {
		parts = strings.Fields(body)
	}
	if len(parts) < 6 {
		return Progress{}, false
	}
	var p Progress
	if pct, ok := parseNum(parts[0]); ok {
		p.Percent = clampPercent(pct)
		p.HasPercent = true
	}
	if down, ok := parseNum(parts[1]); ok && down >= 0 {
		p.Downloaded = int64(down)
		p.HasDownloaded = true
	}
	total, hasTotal := parseNum(parts[2])
	est, hasEst := parseNum(parts[3])
	if hasTotal && total > 0 {
		p.Total = int64(total)
		p.HasTotal = true
	} else if hasEst && est > 0 {
		p.Total = int64(est)
		p.HasTotal = true
	}
	if !p.HasPercent && p.HasDownloaded && p.HasTotal && p.Total > 0 {
		p.Percent = clampPercent(100 * float64(p.Downloaded) / float64(p.Total))
		p.HasPercent = true
	}
	if spd, ok := parseNum(parts[4]); ok && spd > 0 {
		p.Speed = formatSpeed(spd)
	}
	if eta, ok := parseNum(parts[5]); ok && eta >= 1 {
		p.ETA = formatETA(eta)
	}
	return p, p.HasPercent || p.HasDownloaded || p.Speed != "" || p.ETA != ""
}

func parseClassicLine(line string) (Progress, bool) {
	if !strings.Contains(line, "[download]") {
		return Progress{}, false
	}
	m := progressRe.FindStringSubmatch(line)
	if m == nil {
		return Progress{}, false
	}
	pct, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return Progress{}, false
	}
	p := Progress{Percent: clampPercent(pct), HasPercent: true}
	if at := progressAtRe.FindStringSubmatch(line); len(at) == 2 && looksLikeSpeed(at[1]) {
		p.Speed = at[1]
	}
	if eta := progressETA.FindStringSubmatch(line); len(eta) == 2 && looksLikeETA(eta[1]) {
		p.ETA = eta[1]
	}
	return p, true
}

func parseNum(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	if s == "" {
		return 0, false
	}
	switch strings.ToLower(s) {
	case "na", "n/a", "none", "nan", "unknown", "null":
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func clampPercent(f float64) int {
	if f <= 0 {
		return 0
	}
	if f >= 100 {
		return 100
	}
	p := int(math.Round(f))
	if p == 0 {
		return 1
	}
	return p
}

func formatSpeed(bps float64) string {
	switch {
	case bps >= 1024*1024:
		return fmt.Sprintf("%.1f MiB/s", bps/(1024*1024))
	case bps >= 1024:
		return fmt.Sprintf("%.0f KiB/s", bps/1024)
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}

func formatETA(sec float64) string {
	s := int(sec + 0.5)
	if s < 0 {
		return ""
	}
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func FormatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KiB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MiB", float64(n)/(1024*1024))
}

func looksLikeSpeed(s string) bool {
	if s == "" || strings.EqualFold(s, "Unknown") || strings.Contains(strings.ToLower(s), "unknown") {
		return false
	}
	return strings.Contains(strings.ToLower(s), "ib/s") || strings.Contains(strings.ToLower(s), "b/s")
}

func looksLikeETA(s string) bool {
	if s == "" || strings.EqualFold(s, "Unknown") || strings.Contains(strings.ToLower(s), "unknown") {
		return false
	}
	return true
}
