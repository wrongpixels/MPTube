package download

import (
	"strings"
	"testing"

	"youtubemp3/internal/i18n"
	"youtubemp3/internal/uistate"
)

func TestApplyFooterIsTheWindowAdapter(t *testing.T) {
	lines := []string{
		"YTMP3\t  0.1%\t1024\t1421554\tNA\tNA\tNA",
		"YTMP3\t 18.4%\t261120\t1421554\tNA\t21394711.189076204\t0",
		"YTMP3\t100.0%\t1421554\t1421554\tNA\t16421490.244530926\t0",
		"YTMP3POST",
	}
	var view View
	footer := uistate.Reduce(uistate.Footer{}, uistate.Event{Kind: uistate.EvResolving, Line1: i18n.T(i18n.ES, "status.resolving")})
	if !footer.BarLive || footer.Determinate || footer.Action != uistate.ActionCancel {
		t.Fatalf("resolving: %+v", footer)
	}
	var sawDet, sawSpeed bool
	for _, line := range lines {
		var ch Chrome
		view, footer, ch = ApplyFooter(view, footer, line, i18n.ES)
		if !ch.Changed {
			t.Fatalf("expected change on %q", line)
		}
		if footer.Determinate && footer.Percent > 0 {
			sawDet = true
		}
		if strings.Contains(footer.Line1, "MiB/s") || strings.Contains(footer.Line1, "KiB/s") {
			sawSpeed = true
		}
	}
	if !sawDet || !sawSpeed {
		t.Fatalf("download chrome missing det/speed: %+v", footer)
	}
	if footer.Stage != uistate.EvConverting || footer.Determinate {
		t.Fatalf("should end converting: %+v", footer)
	}
	if !strings.Contains(footer.Line1, "Convirtiendo") {
		t.Fatalf("convert line1=%q", footer.Line1)
	}
	if !strings.Contains(footer.Line2, "100") || !strings.Contains(footer.Line2, "Descarga") {
		t.Fatalf("retained summary missing: line2=%q saved=%d %s", footer.Line2, footer.SavedPct, footer.SavedSpeed)
	}
	if footer.Action != uistate.ActionCancel {
		t.Fatalf("convert should keep cancel: %+v", footer)
	}
}

func TestApplyFooterIdleToDoneSlots(t *testing.T) {
	idle := uistate.Reduce(uistate.Footer{}, uistate.Event{Kind: uistate.EvIdle, Line1: i18n.T(i18n.EN, "status.nothing")})
	if idle.Action != uistate.ActionRevealOff || idle.BarLive {
		t.Fatalf("idle %+v", idle)
	}
	done := uistate.Reduce(idle, uistate.Event{Kind: uistate.EvDone, Line1: i18n.T(i18n.EN, "status.savedHeader"), Line2: "song.mp3", Tip2: "song.mp3"})
	if done.Action != uistate.ActionRevealOn || done.BarLive || done.Line1 != "Downloaded!" {
		t.Fatalf("done %+v", done)
	}
}
