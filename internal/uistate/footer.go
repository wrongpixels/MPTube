package uistate

// Action is the single footer button's role. The control stays mounted;
// only its label, enabled state, and click handler change.
type Action string

const (
	ActionRevealOff Action = "reveal_off"
	ActionCancel    Action = "cancel"
	ActionRevealOn  Action = "reveal"
)

const (
	EvIdle        = "idle"
	EvResolving   = "resolving"
	EvDownloading = "downloading"
	EvConverting  = "converting"
	EvWriting     = "writing"
	EvDone        = "done"
	EvError       = "error"
	EvCancelled   = "cancelled"
)

// Footer is the fixed download chrome. Layout slots do not appear or disappear.
type Footer struct {
	Stage       string
	BarLive     bool
	Determinate bool
	Percent     int
	Line1       string
	Line2       string
	Tip1        string
	Tip2        string
	Action      Action
	SavedPct    int
	SavedSpeed  string
}

// Event is a pure input to Reduce. Localized strings are supplied by the caller
// (the same helper the window uses) so tests exercise the real adapter.
type Event struct {
	Kind        string
	Determinate bool
	Percent     int
	Speed       string
	Line1       string
	Line2       string
	Tip1        string
	Tip2        string
}

// ToolStatus is the footer event for helper-tool setup. It stays indeterminate.
// A tool report of 100 must not be stored as audio download progress.
func ToolStatus(line string) Event {
	return Event{Kind: EvResolving, Line1: line}
}

// Reduce applies one footer event. It never talks to widgets.
func Reduce(prev Footer, ev Event) Footer {
	next := prev
	switch ev.Kind {
	case EvIdle:
		return Footer{
			Stage:   EvIdle,
			Line1:   ev.Line1,
			Action:  ActionRevealOff,
			BarLive: false,
		}
	case EvResolving:
		next.Stage = EvResolving
		next.BarLive = true
		next.Determinate = false
		next.Percent = 0
		next.Line1 = ev.Line1
		next.Line2 = ""
		next.Tip1 = ev.Tip1
		next.Tip2 = ""
		next.Action = ActionCancel
	case EvDownloading:
		next.Stage = EvDownloading
		next.BarLive = true
		next.Determinate = ev.Determinate
		next.Percent = ev.Percent
		next.Line1 = ev.Line1
		next.Line2 = ev.Line2
		next.Tip1 = ev.Tip1
		next.Tip2 = ev.Tip2
		next.Action = ActionCancel
		if ev.Percent > 0 {
			next.SavedPct = ev.Percent
		}
		if ev.Speed != "" {
			next.SavedSpeed = ev.Speed
		}
	case EvConverting:
		next.Stage = EvConverting
		next.BarLive = true
		next.Determinate = false
		next.Percent = 0
		next.Line1 = ev.Line1
		next.Line2 = ev.Line2
		next.Tip1 = ev.Tip1
		next.Tip2 = ev.Tip2
		next.Action = ActionCancel
	case EvWriting:
		next.Stage = EvWriting
		next.BarLive = true
		next.Determinate = false
		next.Percent = 0
		next.Line1 = ev.Line1
		next.Line2 = ev.Line2
		next.Action = ActionCancel
	case EvDone:
		return Footer{
			Stage:       EvDone,
			BarLive:     false,
			Determinate: true,
			Percent:     100,
			Line1:       ev.Line1,
			Line2:       ev.Line2,
			Tip1:        ev.Tip1,
			Tip2:        ev.Tip2,
			Action:      ActionRevealOn,
			SavedPct:    100,
			SavedSpeed:  prev.SavedSpeed,
		}
	case EvError:
		next.Stage = EvError
		next.BarLive = false
		next.Determinate = false
		next.Percent = 0
		next.Line1 = ev.Line1
		next.Line2 = ev.Line2
		next.Tip1 = ev.Tip1
		next.Tip2 = ev.Tip2
		next.Action = ActionRevealOff
	case EvCancelled:
		next.Stage = EvCancelled
		next.BarLive = false
		next.Determinate = false
		next.Percent = 0
		next.Line1 = ev.Line1
		next.Line2 = ev.Line2
		next.Action = ActionRevealOff
	}
	return next
}
