package uistate

// Phase is the chrome state of the status line and progress strip.
type Phase int

const (
	PhaseIdle Phase = iota
	PhasePreviewReady
	PhasePreviewLoading
	PhaseWorking
	PhaseConverting
	PhaseWriting
	PhaseDownloaded
)

// View describes what the bottom chrome should show.
// Progress is visible only while work is active. Completion text is used only
// after a file was actually written.
type View struct {
	ProgressVisible     bool
	ProgressDeterminate bool
	ProgressPercent     int
	StatusKey           string
	IsCompletion        bool
}

func clampPercent(p int) int {
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

// ViewFor maps an app phase to progress visibility and status.
// Preview-ready is not a download completion, even if a caller passes 100%.
// Network 100% is still PhaseWorking, not conversion or completion.
func ViewFor(phase Phase, percent int, determinate bool) View {
	switch phase {
	case PhasePreviewLoading:
		return View{
			ProgressVisible:     true,
			ProgressDeterminate: false,
			StatusKey:           "status.loading",
		}
	case PhaseWorking:
		return View{
			ProgressVisible:     true,
			ProgressDeterminate: determinate,
			ProgressPercent:     clampPercent(percent),
		}
	case PhaseConverting:
		return View{
			ProgressVisible:     true,
			ProgressDeterminate: false,
			StatusKey:           "status.converting",
		}
	case PhaseWriting:
		return View{
			ProgressVisible:     true,
			ProgressDeterminate: false,
			StatusKey:           "status.writing",
		}
	case PhaseDownloaded:
		return View{
			ProgressVisible: false,
			ProgressPercent: 0,
			StatusKey:       "status.saved",
			IsCompletion:    true,
		}
	case PhasePreviewReady:
		return View{
			ProgressVisible: false,
			ProgressPercent: 0,
			StatusKey:       "status.hint",
			IsCompletion:    false,
		}
	default:
		return View{
			ProgressVisible: false,
			ProgressPercent: 0,
			StatusKey:       "status.hint",
			IsCompletion:    false,
		}
	}
}
