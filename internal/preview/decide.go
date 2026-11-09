package preview

import "youtubemp3/internal/yturl"

type Op string

const (
	OpNone  Op = "none"
	OpFetch Op = "fetch"
	OpClear Op = "clear"
)

// Decide returns what auto-preview should do after the URL field settles.
// Fetch only starts for a complete video URL that is not already pending or shown.
func Decide(raw, pending, committed string) (Op, string) {
	cleaned, ready := yturl.ReadyForFetch(raw)
	if ready {
		if cleaned == committed || cleaned == pending {
			return OpNone, cleaned
		}
		return OpFetch, cleaned
	}
	if pending != "" || committed != "" {
		return OpClear, ""
	}
	return OpNone, ""
}
