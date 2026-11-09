package folderui

import (
	"strings"

	"youtubemp3/internal/settings"
)

// Display is what the folder combo should show after a model refresh.
type Display struct {
	Model []string
	Text  string
	Index int // -1 if Text is not in the model
}

// DisplayOf builds the combo view from persisted recents and the chosen folder.
// Text is never empty when output is set, even if a widget SetModel just cleared the edit.
func DisplayOf(recents []string, output string) Display {
	model := append([]string(nil), recents...)
	d := Display{Model: model, Text: output, Index: -1}
	for i, r := range model {
		if settings.PathEqual(r, output) {
			d.Index = i
			break
		}
	}
	return d
}

// PathFromClick reads the selected folder from a dropdown event.
// comboText is preferred (what the user saw); index is the fallback into recents.
// index < 0 means a reset/clear event and must not yield a path.
func PathFromClick(recents []string, index int, comboText string) (string, bool) {
	if index < 0 {
		return "", false
	}
	if strings.TrimSpace(comboText) != "" {
		return comboText, true
	}
	if index < len(recents) && strings.TrimSpace(recents[index]) != "" {
		return recents[index], true
	}
	return "", false
}

func ShouldIgnoreIndexEvent(updating bool, index int) bool {
	return updating || index < 0
}

func AlreadyShowing(output, path string) bool {
	return settings.PathEqual(output, path) && strings.TrimSpace(path) != ""
}
