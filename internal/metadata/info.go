package metadata

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"youtubemp3/internal/sanitize"
)

// PrintTemplate is the yt-dlp --print value used to fetch video details as JSON.
const PrintTemplate = "%(.{title,duration,thumbnail})j"

type Info struct {
	Title     string
	Duration  float64
	HasDur    bool
	Thumbnail string
	Thumbs    []Thumb
}

// Thumb is one entry from yt-dlp's thumbnails array. URL is untrusted until
// CanonicalThumbnailURL accepts it.
type Thumb struct {
	URL        string
	Width      int
	Height     int
	Preference int
	HasPref    bool
}

type rawThumb struct {
	URL        string          `json:"url"`
	Width      json.RawMessage `json:"width"`
	Height     json.RawMessage `json:"height"`
	Preference json.RawMessage `json:"preference"`
}

type rawInfo struct {
	Title      string          `json:"title"`
	Duration   json.RawMessage `json:"duration"`
	Thumbnail  string          `json:"thumbnail"`
	Thumbnails []rawThumb      `json:"thumbnails"`
}

func ExtractJSONObject(stdout string) ([]byte, bool) {
	// yt-dlp --dump-single-json is one JSON line. Screen or stderr text can
	// contain braces after that line; taking the first '{' through the last '}'
	// then fails json.Valid and the caller drops the raw bytes.
	var lineBest []byte
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || line[0] != '{' || !json.Valid([]byte(line)) {
			continue
		}
		if !strings.Contains(line, `"title"`) && !strings.Contains(line, `"formats"`) {
			continue
		}
		if len(line) > len(lineBest) {
			lineBest = []byte(line)
		}
	}
	var span []byte
	if i := strings.Index(stdout, "{"); i >= 0 {
		if j := strings.LastIndex(stdout, "}"); j > i {
			raw := []byte(stdout[i : j+1])
			if json.Valid(raw) {
				span = raw
			}
		}
	}
	if len(span) >= len(lineBest) && span != nil {
		return span, true
	}
	if lineBest != nil {
		return lineBest, true
	}
	return nil, false
}

// ParseDump extracts the full yt-dlp JSON object (for caching) and the display fields.
func ParseDump(stdout string) (raw []byte, info Info, err error) {
	if raw, ok := ExtractJSONObject(stdout); ok {
		info, err = ParsePrintJSON(string(raw))
		if err != nil {
			return raw, Info{}, err
		}
		if info.Title == "" || !sanitize.IsMostlyPrintable(info.Title) {
			info.Title = "youtube-audio"
		}
		return raw, info, nil
	}
	info, err = ParseOutput(stdout)
	return nil, info, err
}

func ParsePrintJSON(line string) (Info, error) {
	line = strings.TrimSpace(line)
	var raw rawInfo
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return Info{}, err
	}
	info := Info{Title: strings.TrimSpace(raw.Title), Thumbnail: strings.TrimSpace(raw.Thumbnail)}
	for _, th := range raw.Thumbnails {
		item := Thumb{URL: strings.TrimSpace(th.URL)}
		if item.URL == "" {
			continue
		}
		if w, ok := rawInt(th.Width); ok && w > 0 {
			item.Width = w
		}
		if h, ok := rawInt(th.Height); ok && h > 0 {
			item.Height = h
		}
		if p, ok := rawInt(th.Preference); ok {
			item.Preference = p
			item.HasPref = true
		}
		info.Thumbs = append(info.Thumbs, item)
	}
	if d, ok := parseDuration(raw.Duration); ok {
		info.Duration = d
		info.HasDur = true
	}
	return info, nil
}

func rawInt(raw json.RawMessage) (int, bool) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return 0, false
	}
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return n, true
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
		return int(f), true
	}
	return 0, false
}

func ParseOutput(stdout string) (Info, error) {
	var last Info
	found := false
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}
		info, err := ParsePrintJSON(line)
		if err != nil {
			continue
		}
		last = info
		found = true
	}
	if !found {
		return Info{}, fmt.Errorf("could not read the video title and details")
	}
	if last.Title == "" || !sanitize.IsMostlyPrintable(last.Title) {
		last.Title = "youtube-audio"
	}
	return last, nil
}

func parseDuration(raw json.RawMessage) (float64, bool) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return 0, false
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil && !math.IsNaN(f) && !math.IsInf(f, 0) && f >= 0 {
		return f, true
	}
	var str string
	if json.Unmarshal(raw, &str) == nil {
		f, err := strconv.ParseFloat(strings.TrimSpace(str), 64)
		if err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) && f >= 0 {
			return f, true
		}
	}
	return 0, false
}

func FormatDuration(sec float64) string {
	if sec < 0 || math.IsNaN(sec) || math.IsInf(sec, 0) {
		return ""
	}
	s := int(math.Round(sec))
	h := s / 3600
	m := (s % 3600) / 60
	ss := s % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, ss)
	}
	return fmt.Sprintf("%d:%02d", m, ss)
}

func DefaultFilename(title string) string {
	return sanitize.SuggestedFilename(title)
}
