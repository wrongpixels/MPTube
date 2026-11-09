package clip

import "strings"

// Normalize extracts the first non-empty line from clipboard text.
func Normalize(raw string) string {
	raw = strings.TrimPrefix(raw, "\ufeff")
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	for _, line := range strings.Split(raw, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			return s
		}
	}
	return ""
}

// DecidePaste returns the text to put in the URL field.
// apply is false when there is no usable text; the current field must be kept.
func DecidePaste(current, clipboard string, hasText bool) (next string, apply bool) {
	if !hasText {
		return current, false
	}
	n := Normalize(clipboard)
	if n == "" {
		return current, false
	}
	return n, true
}
