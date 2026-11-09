package download

import (
	"youtubemp3/internal/i18n"
	"youtubemp3/internal/uistate"
)

// Chrome is the parsed download line the window turns into a footer event.
type Chrome struct {
	Changed     bool
	Kind        string
	Determinate bool
	Percent     int
	StatusMode  string // percent, bytes, converting, or empty
	Speed       string
	ETA         string
	Downloaded  int64
	HasPercent  bool
	HasTotal    bool
	Total       int64
}

// Step applies one captured subprocess line the same way the app window does.
func Step(prev View, line string) (View, Chrome) {
	next := UpdateView(prev, line)
	if next == prev {
		return prev, Chrome{}
	}
	ch := Chrome{
		Changed:    true,
		Kind:       next.Kind,
		Percent:    next.Percent,
		HasPercent: next.HasPercent,
		Speed:      next.Speed,
		ETA:        next.ETA,
		Downloaded: next.Downloaded,
		HasTotal:   next.HasTotal,
		Total:      next.Total,
	}
	switch next.Kind {
	case "converting":
		ch.StatusMode = "converting"
	case "downloading":
		ch.Determinate = true
		ch.StatusMode = "percent"
	case "bytes":
		ch.StatusMode = "bytes"
	}
	return next, ch
}

// FooterEvent builds the same uistate.Event the window publishes after Step.
func FooterEvent(ch Chrome, lang string, prev uistate.Footer) uistate.Event {
	switch ch.Kind {
	case "converting":
		return uistate.Event{
			Kind:  uistate.EvConverting,
			Line1: i18n.T(lang, "status.converting"),
			Line2: i18n.RetainedDownload(lang, prev.SavedPct, prev.SavedSpeed),
		}
	case "downloading":
		return uistate.Event{
			Kind:        uistate.EvDownloading,
			Determinate: true,
			Percent:     ch.Percent,
			Speed:       ch.Speed,
			Line1:       i18n.DownloadStatus(lang, ch.Percent, ch.Speed, ch.ETA),
		}
	case "bytes":
		return uistate.Event{
			Kind:        uistate.EvDownloading,
			Determinate: false,
			Speed:       ch.Speed,
			Line1:       i18n.DownloadBytesStatus(lang, FormatBytes(ch.Downloaded), ch.Speed),
		}
	default:
		return uistate.Event{}
	}
}

// ApplyFooter is Step plus Reduce: the exact model transition the window uses.
func ApplyFooter(prevView View, prevFooter uistate.Footer, line, lang string) (View, uistate.Footer, Chrome) {
	nextView, ch := Step(prevView, line)
	if !ch.Changed {
		return prevView, prevFooter, Chrome{}
	}
	return nextView, uistate.Reduce(prevFooter, FooterEvent(ch, lang, prevFooter)), ch
}
