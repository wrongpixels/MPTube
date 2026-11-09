// Package uiclamp shortens labels so walk TextLabel min-size cannot stretch the window.
package uiclamp

import "strings"

// EllipsisEnd trims s to maxRunes and adds a trailing ellipsis when needed.
func EllipsisEnd(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	if maxRunes == 1 {
		return "…"
	}
	return string(r[:maxRunes-1]) + "…"
}

// EllipsisLines applies EllipsisEnd to each line separately.
func EllipsisLines(s string, maxRunes int) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = EllipsisEnd(line, maxRunes)
	}
	return strings.Join(lines, "\n")
}
