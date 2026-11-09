package uistate

import "testing"

func TestReduceIdleHasInactiveBarAndDisabledReveal(t *testing.T) {
	f := Reduce(Footer{}, Event{Kind: EvIdle, Line1: "Nothing downloaded yet"})
	if f.BarLive || f.Determinate || f.Action != ActionRevealOff || f.Line1 != "Nothing downloaded yet" {
		t.Fatalf("%+v", f)
	}
}

func TestToolStatusIsNotAudioCompletion(t *testing.T) {
	f := Reduce(Footer{}, ToolStatus("Helper tools are ready."))
	if f.Stage != EvResolving || f.Determinate || f.Percent != 0 || !f.BarLive {
		t.Fatalf("tool ready looked like audio progress: %+v", f)
	}
}

func TestReduceResolvingIsIndeterminateCancel(t *testing.T) {
	f := Reduce(Footer{}, Event{Kind: EvResolving, Line1: "Fetching video data…"})
	if !f.BarLive || f.Determinate || f.Action != ActionCancel || f.Stage != EvResolving {
		t.Fatalf("%+v", f)
	}
}

func TestReduceDownloadingKeepsPercentAndSpeed(t *testing.T) {
	f := Reduce(Footer{Stage: EvResolving, Action: ActionCancel, BarLive: true}, Event{
		Kind:        EvDownloading,
		Determinate: true,
		Percent:     18,
		Speed:       "20.4 MiB/s",
		Line1:       "Downloading audio… 18% • 20.4 MiB/s",
	})
	if !f.Determinate || f.Percent != 18 || f.SavedSpeed != "20.4 MiB/s" || f.SavedPct != 18 {
		t.Fatalf("%+v", f)
	}
	if f.Action != ActionCancel || !f.BarLive {
		t.Fatalf("%+v", f)
	}
}

func TestReduceConvertingRetainsDownloadSummary(t *testing.T) {
	prev := Footer{Stage: EvDownloading, SavedPct: 100, SavedSpeed: "15.7 MiB/s", Action: ActionCancel}
	f := Reduce(prev, Event{
		Kind:  EvConverting,
		Line1: "Converting to MP3…",
		Line2: "Download: 100% · 15.7 MiB/s",
	})
	if f.Determinate || !f.BarLive || f.Action != ActionCancel {
		t.Fatalf("convert chrome: %+v", f)
	}
	if f.SavedPct != 100 || f.SavedSpeed != "15.7 MiB/s" {
		t.Fatalf("lost retained stats: %+v", f)
	}
	if f.Line2 != "Download: 100% · 15.7 MiB/s" {
		t.Fatalf("line2=%q", f.Line2)
	}
}

func TestReduceDoneUsesFixedRevealAndInactiveBar(t *testing.T) {
	f := Reduce(Footer{SavedPct: 100, SavedSpeed: "10 MiB/s"}, Event{
		Kind:  EvDone,
		Line1: "Downloaded!",
		Line2: `C:\Music\song.mp3`,
		Tip2:  `C:\Music\song.mp3`,
	})
	if f.BarLive || f.Action != ActionRevealOn || f.Line1 != "Downloaded!" {
		t.Fatalf("%+v", f)
	}
	if !f.Determinate || f.Percent != 100 {
		t.Fatalf("success bar must stay full: %+v", f)
	}
}

func TestReduceErrorAndCancelClearBar(t *testing.T) {
	prev := Footer{Determinate: true, Percent: 80, SavedPct: 80, BarLive: true}
	errf := Reduce(prev, Event{Kind: EvError, Line1: "failed"})
	if errf.BarLive || errf.Determinate || errf.Percent != 0 || errf.Action != ActionRevealOff {
		t.Fatalf("error: %+v", errf)
	}
	canc := Reduce(prev, Event{Kind: EvCancelled, Line1: "Cancelled."})
	if canc.BarLive || canc.Determinate || canc.Percent != 0 {
		t.Fatalf("cancel: %+v", canc)
	}
}
