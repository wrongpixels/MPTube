package sanitize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const maxBaseRunes = 150

var reserved = map[string]struct{}{
	"CON": {}, "PRN": {}, "AUX": {}, "NUL": {},
	"COM1": {}, "COM2": {}, "COM3": {}, "COM4": {}, "COM5": {}, "COM6": {}, "COM7": {}, "COM8": {}, "COM9": {},
	"LPT1": {}, "LPT2": {}, "LPT3": {}, "LPT4": {}, "LPT5": {}, "LPT6": {}, "LPT7": {}, "LPT8": {}, "LPT9": {},
}

// Filename turns a video title into a safe Windows base name (no extension).
func Filename(title string) string {
	title = strings.TrimSpace(title)
	var b strings.Builder
	for _, r := range title {
		if r < 32 || r == 127 || strings.ContainsRune(`<>:"/\|?*`, r) {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}
	name := strings.TrimSpace(b.String())
	name = strings.TrimRight(name, " .")
	name = strings.TrimLeft(name, " ")
	if name == "" || name == "." || name == ".." {
		name = "youtube-audio"
	}
	runes := []rune(name)
	if len(runes) > maxBaseRunes {
		name = strings.TrimRight(string(runes[:maxBaseRunes]), " .")
		if name == "" {
			name = "youtube-audio"
		}
	}
	stem := name
	if i := strings.LastIndex(name, "."); i >= 0 {
		stem = name[:i]
	}
	if _, ok := reserved[strings.ToUpper(stem)]; ok {
		name = name + "_"
	}
	// reserved names with no extension
	if _, ok := reserved[strings.ToUpper(name)]; ok {
		name = name + "_"
	}
	return name
}

// UniqueMP3Path returns dir/base.mp3 or dir/base (n).mp3 if that name exists.
// It never overwrites an existing file and never creates the directory.
func UniqueMP3Path(dir, base string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", fmt.Errorf("output folder is empty")
	}
	base = Filename(base)
	candidate := filepath.Join(dir, base+".mp3")
	if _, err := os.Stat(candidate); err != nil {
		if os.IsNotExist(err) {
			return candidate, nil
		}
		return "", err
	}
	for n := 1; n < 1000; n++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s (%d).mp3", base, n))
		if _, err := os.Stat(candidate); err != nil {
			if os.IsNotExist(err) {
				return candidate, nil
			}
			return "", err
		}
	}
	return "", fmt.Errorf("could not find an unused filename for %s", base)
}

func IsMostlyPrintable(s string) bool {
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func isCJKLetter(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hangul, r)
}

func hasUsefulLatinOrDigit(s string) bool {
	n := 0
	for _, r := range s {
		if unicode.IsDigit(r) || (unicode.IsLetter(r) && !isCJKLetter(r)) {
			n++
			if n >= 2 {
				return true
			}
		}
	}
	return false
}

func collapseSpaces(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func stripEmptyBrackets(s string) string {
	pairs := []string{"()", "[]", "{}", "<>", "《》", "〈〉", "「」", "『』", "【】", "〔〕", "（）"}
	for {
		next := s
		for _, p := range pairs {
			open, close := string([]rune(p)[0]), string([]rune(p)[1])
			next = strings.ReplaceAll(next, open+close, "")
			next = strings.ReplaceAll(next, open+" "+close, "")
		}
		next = collapseSpaces(next)
		if next == s {
			return next
		}
		s = next
	}
}

func tidySeparators(s string) string {
	s = strings.ReplaceAll(s, "|", "-")
	s = collapseSpaces(s)
	s = stripEmptyBrackets(s)
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.ReplaceAll(s, " - - ", " - ")
	s = collapseSpaces(s)
	s = strings.Trim(s, " -_")
	return collapseSpaces(s)
}

// SuggestedFilename builds a default download stem from a video title.
// CJK letters are dropped only when enough Latin letters or digits remain.
// The preview title is not passed through this helper.
func SuggestedFilename(title string) string {
	title = strings.TrimSpace(title)
	var stripped strings.Builder
	for _, r := range title {
		if isCJKLetter(r) {
			continue
		}
		stripped.WriteRune(r)
	}
	cleaned := tidySeparators(stripped.String())
	if hasUsefulLatinOrDigit(cleaned) {
		return Filename(cleaned)
	}
	return Filename(title)
}

// OutputBase turns an optional user filename into a safe Windows stem (no .mp3).
// A blank value falls back to the sanitized video title.
func OutputBase(user, fallbackTitle string) string {
	s := strings.TrimSpace(user)
	for len(s) >= 4 && strings.EqualFold(s[len(s)-4:], ".mp3") {
		s = strings.TrimSpace(s[:len(s)-4])
	}
	if s == "" {
		return SuggestedFilename(fallbackTitle)
	}
	return Filename(s)
}
