package yturl

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

type Kind int

const (
	KindNone Kind = iota
	KindOther
	KindIncomplete
	KindNotVideo
	KindVideo
)

// Result is the unified interpretation of text typed or pasted into the URL field.
type Result struct {
	Kind      Kind
	Canonical string
	VideoID   string
}

var allowedHosts = map[string]struct{}{
	"youtube.com":              {},
	"www.youtube.com":          {},
	"m.youtube.com":            {},
	"music.youtube.com":        {},
	"youtu.be":                 {},
	"www.youtu.be":             {},
	"youtube-nocookie.com":     {},
	"www.youtube-nocookie.com": {},
}

// Longer hosts first so www.youtube.com wins over youtube.com when searching text.
var searchHosts = []string{
	"www.youtube-nocookie.com",
	"youtube-nocookie.com",
	"music.youtube.com",
	"www.youtube.com",
	"m.youtube.com",
	"www.youtu.be",
	"youtube.com",
	"youtu.be",
}

var (
	videoIDRe      = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	incompleteIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,10}$`)
	httpURLRe      = regexp.MustCompile(`(?i)https?://[^\s<>"']+`)
)

// Analyze interprets typed or pasted text. It never treats a video id on a
// non-YouTube host as valid, and it only extracts a URL when the host itself
// is an exact allowed YouTube host.
func Analyze(raw string) Result {
	raw = strings.TrimPrefix(raw, "\ufeff")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Result{Kind: KindNone}
	}
	if r := analyzeCandidate(raw); r.Kind != KindOther && r.Kind != KindNone {
		return r
	}
	if cand := extractYouTubeURL(raw); cand != "" {
		if r := analyzeCandidate(cand); r.Kind != KindOther && r.Kind != KindNone {
			return r
		}
	}
	return Result{Kind: KindOther}
}

// Validate accepts a public YouTube video URL (or text containing one) and
// returns the canonical https://www.youtube.com/watch?v=ID form.
func Validate(raw string) (string, error) {
	switch r := Analyze(raw); r.Kind {
	case KindVideo:
		return r.Canonical, nil
	case KindNone:
		return "", fmt.Errorf("paste a YouTube link first")
	case KindNotVideo, KindIncomplete:
		return "", fmt.Errorf("that YouTube link does not point to a video")
	default:
		if looksLikeURL(raw) && !allowedHostIn(raw) {
			return "", fmt.Errorf("only public YouTube links are allowed")
		}
		if strings.ContainsAny(raw, "\r\n\x00") {
			return "", fmt.Errorf("the link contains invalid characters")
		}
		return "", fmt.Errorf("only public YouTube links are allowed")
	}
}

// VideoID extracts a standard 11-character YouTube video id.
func VideoID(raw string) (string, bool) {
	r := Analyze(raw)
	if r.Kind != KindVideo {
		return "", false
	}
	return r.VideoID, true
}

// ReadyForFetch reports whether raw identifies a complete public YouTube video
// and returns the canonical watch URL. Incomplete ids typed so far are rejected
// so auto-preview does not fire on every keystroke.
func ReadyForFetch(raw string) (string, bool) {
	r := Analyze(raw)
	if r.Kind != KindVideo {
		return "", false
	}
	return r.Canonical, true
}

func analyzeCandidate(raw string) Result {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "<>")
	if raw == "" {
		return Result{Kind: KindNone}
	}
	if strings.ContainsRune(raw, 0) {
		return Result{Kind: KindOther}
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return Result{Kind: KindOther}
	}
	if parsed.Scheme == "" && parsed.Host != "" {
		parsed.Scheme = "https"
	}
	if parsed.Scheme == "" {
		if !looksLikeKnownHost(raw) {
			return Result{Kind: KindOther}
		}
		parsed, err = url.Parse("https://" + raw)
		if err != nil {
			return Result{Kind: KindOther}
		}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return Result{Kind: KindOther}
	}
	if parsed.User != nil {
		return Result{Kind: KindOther}
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if _, ok := allowedHosts[host]; !ok {
		return Result{Kind: KindOther}
	}
	id, state := extractID(parsed, host)
	switch state {
	case idComplete:
		return Result{
			Kind:      KindVideo,
			Canonical: "https://www.youtube.com/watch?v=" + id,
			VideoID:   id,
		}
	case idIncomplete:
		return Result{Kind: KindIncomplete}
	default:
		return Result{Kind: KindNotVideo}
	}
}

type idState int

const (
	idNone idState = iota
	idIncomplete
	idComplete
)

func extractID(parsed *url.URL, host string) (string, idState) {
	path := strings.Trim(parsed.Path, "/")
	var parts []string
	if path != "" {
		parts = strings.Split(path, "/")
	}
	query := parsed.Query()
	_, hasV := query["v"]
	v := query.Get("v")

	candidate := ""
	hasSlot := false

	if host == "youtu.be" || host == "www.youtu.be" {
		hasSlot = true
		if len(parts) > 0 {
			candidate = parts[0]
		}
	}
	if hasV {
		hasSlot = true
		candidate = v
	} else if candidate == "" && len(parts) >= 2 {
		switch strings.ToLower(parts[0]) {
		case "shorts", "embed", "live", "v":
			hasSlot = true
			candidate = parts[1]
		}
	} else if candidate == "" && len(parts) == 1 {
		switch strings.ToLower(parts[0]) {
		case "shorts", "embed", "live", "v", "watch":
			hasSlot = true
		}
	}

	if videoIDRe.MatchString(candidate) {
		return candidate, idComplete
	}
	if candidate != "" && incompleteIDRe.MatchString(candidate) {
		return "", idIncomplete
	}
	if candidate != "" {
		// Present but neither a full nor an in-progress video id.
		return "", idNone
	}
	if hasSlot && candidate == "" {
		if len(parts) > 0 && strings.EqualFold(parts[0], "watch") {
			if hasV {
				return "", idIncomplete
			}
			if len(query) > 0 {
				return "", idNone
			}
			return "", idIncomplete
		}
		if host == "youtu.be" || host == "www.youtu.be" {
			if len(query) > 0 && !hasV && path == "" {
				return "", idNone
			}
			return "", idIncomplete
		}
		return "", idIncomplete
	}
	if isNonVideoPath(parts) {
		return "", idNone
	}
	if len(parts) == 0 {
		return "", idIncomplete
	}
	p0 := strings.ToLower(parts[0])
	for _, known := range []string{"watch", "shorts", "embed", "live", "v"} {
		if strings.HasPrefix(known, p0) {
			return "", idIncomplete
		}
	}
	return "", idNone
}

func isNonVideoPath(parts []string) bool {
	if len(parts) == 0 {
		return false
	}
	switch strings.ToLower(parts[0]) {
	case "playlist", "channel", "c", "user", "results", "feed", "account", "hashtag":
		return true
	}
	return strings.HasPrefix(parts[0], "@")
}

func looksLikeKnownHost(raw string) bool {
	host := raw
	if i := strings.IndexAny(raw, "/?#"); i >= 0 {
		host = raw[:i]
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	_, ok := allowedHosts[host]
	return ok
}

func extractYouTubeURL(text string) string {
	for _, m := range httpURLRe.FindAllString(text, -1) {
		m = trimTrailingPunct(m)
		if r := analyzeCandidate(m); meaningful(r.Kind) {
			return m
		}
	}
	lower := strings.ToLower(text)
	bestPos := -1
	best := ""
	for _, host := range searchHosts {
		start := 0
		for {
			rel := strings.Index(lower[start:], host)
			if rel < 0 {
				break
			}
			i := start + rel
			if !hostBoundaryBefore(text, i) {
				start = i + 1
				continue
			}
			end := i + len(host)
			for end < len(text) && !isURLStop(text[end]) {
				end++
			}
			cand := trimTrailingPunct(text[i:end])
			if r := analyzeCandidate(cand); meaningful(r.Kind) {
				if bestPos < 0 || i < bestPos {
					bestPos = i
					best = cand
				}
			}
			start = i + 1
		}
	}
	return best
}

func meaningful(k Kind) bool {
	return k == KindVideo || k == KindNotVideo || k == KindIncomplete
}

func hostBoundaryBefore(text string, i int) bool {
	if i == 0 {
		return true
	}
	c := text[i-1]
	return !isHostChar(c) && c != '@'
}

func isHostChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_'
}

func isURLStop(c byte) bool {
	if c <= ' ' {
		return true
	}
	switch c {
	case '<', '>', '"', '\'':
		return true
	}
	return unicode.IsSpace(rune(c))
}

func trimTrailingPunct(s string) string {
	return strings.TrimRight(s, ".,!?;:)]}")
}

func looksLikeURL(raw string) bool {
	s := strings.TrimSpace(raw)
	return strings.Contains(s, "://") || looksLikeKnownHost(s)
}

func allowedHostIn(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	_, ok := allowedHosts[host]
	return ok
}
