package uistate

import "testing"

func TestPreviewReadyIsNotDownloadedAndHidesProgress(t *testing.T) {
	v := ViewFor(PhasePreviewReady, 100, true)
	if v.ProgressVisible {
		t.Fatal("preview-ready must hide the progress strip")
	}
	if v.IsCompletion || v.StatusKey == "status.saved" {
		t.Fatal("preview-ready is not a completed download")
	}
	if v.ProgressPercent != 0 {
		t.Fatalf("hidden progress percent=%d", v.ProgressPercent)
	}
}

func TestDownloadedShowsCompletionAndHidesProgress(t *testing.T) {
	v := ViewFor(PhaseDownloaded, 100, true)
	if v.ProgressVisible {
		t.Fatal("completed download must hide the idle progress strip")
	}
	if !v.IsCompletion || v.StatusKey != "status.saved" {
		t.Fatalf("completion: %+v", v)
	}
}

func TestWorkingShowsProgress(t *testing.T) {
	v := ViewFor(PhaseWorking, 42, true)
	if !v.ProgressVisible || !v.ProgressDeterminate || v.ProgressPercent != 42 || v.IsCompletion {
		t.Fatalf("working: %+v", v)
	}
	v = ViewFor(PhaseWorking, 150, true)
	if v.ProgressPercent != 100 {
		t.Fatalf("clamp high: %d", v.ProgressPercent)
	}
	v = ViewFor(PhaseWorking, 0, false)
	if !v.ProgressVisible || v.ProgressDeterminate || v.IsCompletion {
		t.Fatalf("indeterminate working: %+v", v)
	}
}

func TestPreviewLoadingAndConversionAreIndeterminate(t *testing.T) {
	v := ViewFor(PhasePreviewLoading, 0, false)
	if !v.ProgressVisible || v.ProgressDeterminate || v.StatusKey != "status.loading" {
		t.Fatalf("preview loading: %+v", v)
	}
	v = ViewFor(PhaseConverting, 100, true)
	if !v.ProgressVisible || v.ProgressDeterminate || v.StatusKey != "status.converting" {
		t.Fatalf("conversion must not look finished at network 100%%: %+v", v)
	}
	v = ViewFor(PhaseWriting, 100, true)
	if !v.ProgressVisible || v.ProgressDeterminate || v.StatusKey != "status.writing" || v.IsCompletion {
		t.Fatalf("writing is not success yet: %+v", v)
	}
}

func TestIdleHidesProgress(t *testing.T) {
	v := ViewFor(PhaseIdle, 0, false)
	if v.ProgressVisible || v.IsCompletion {
		t.Fatalf("idle: %+v", v)
	}
}

func TestStateBindingNeverFullWhenIdle(t *testing.T) {
	for _, phase := range []Phase{PhaseIdle, PhasePreviewReady, PhaseDownloaded} {
		v := ViewFor(phase, 100, true)
		if v.ProgressVisible {
			t.Fatalf("phase %v should not show a bar", phase)
		}
	}
	for _, phase := range []Phase{PhasePreviewLoading, PhaseWorking, PhaseConverting, PhaseWriting} {
		v := ViewFor(phase, 0, false)
		if !v.ProgressVisible {
			t.Fatalf("phase %v should show a bar", phase)
		}
	}
}
