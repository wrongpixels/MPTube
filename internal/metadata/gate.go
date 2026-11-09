package metadata

import "sync"

// Fetched is the metadata committed for a specific validated URL.
type Fetched struct {
	SourceURL   string
	Title       string
	Duration    string
	DefaultName string
	ThumbURL    string
}

// Gate prevents a stale Fetch from updating the UI after the URL changes
// or a newer Fetch starts.
type Gate struct {
	mu         sync.Mutex
	seq        uint64
	pendingURL string
	result     *Fetched
}

// Begin starts a new fetch for validatedURL and invalidates any previous result.
func (g *Gate) Begin(validatedURL string) uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seq++
	g.pendingURL = validatedURL
	g.result = nil
	return g.seq
}

// SyncToURL drops pending or committed metadata when the current URL no longer
// matches it. Typing that still validates to the same URL is kept.
func (g *Gate) SyncToURL(validated string, valid bool) (cleared bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.result != nil && valid && validated == g.result.SourceURL {
		return false
	}
	if g.result == nil && g.pendingURL != "" && valid && validated == g.pendingURL {
		return false
	}
	if g.result == nil && g.pendingURL == "" {
		return false
	}
	g.seq++
	g.pendingURL = ""
	g.result = nil
	return true
}

func (g *Gate) Commit(seq uint64, f Fetched) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if seq != g.seq {
		return false
	}
	cp := f
	g.result = &cp
	g.pendingURL = ""
	return true
}

func (g *Gate) Invalidate() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seq++
	g.pendingURL = ""
	g.result = nil
}

func (g *Gate) Snapshot() (Fetched, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.result == nil {
		return Fetched{}, false
	}
	return *g.result, true
}

func (g *Gate) MatchesURL(validated string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.result != nil && g.result.SourceURL == validated
}

func (g *Gate) Seq() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.seq
}

func (g *Gate) PendingURL() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.pendingURL
}
