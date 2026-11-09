package metadata

import "testing"

func TestGateStaleFetchDoesNotCommit(t *testing.T) {
	var g Gate
	seq1 := g.Begin("https://www.youtube.com/watch?v=AAAAAAAAAAA")
	seq2 := g.Begin("https://www.youtube.com/watch?v=BBBBBBBBBBB")
	if seq2 == seq1 {
		t.Fatal("seq should increase")
	}
	if g.Commit(seq1, Fetched{SourceURL: "https://www.youtube.com/watch?v=AAAAAAAAAAA", Title: "Old"}) {
		t.Fatal("stale commit succeeded")
	}
	if _, ok := g.Snapshot(); ok {
		t.Fatal("stale result was stored")
	}
	if !g.Commit(seq2, Fetched{SourceURL: "https://www.youtube.com/watch?v=BBBBBBBBBBB", Title: "New"}) {
		t.Fatal("current commit failed")
	}
	got, ok := g.Snapshot()
	if !ok || got.Title != "New" {
		t.Fatalf("got %+v %v", got, ok)
	}
}

func TestGateURLChangeInvalidatesResultAndPending(t *testing.T) {
	var g Gate
	seq := g.Begin("https://www.youtube.com/watch?v=AAAAAAAAAAA")
	if !g.Commit(seq, Fetched{SourceURL: "https://www.youtube.com/watch?v=AAAAAAAAAAA", Title: "A"}) {
		t.Fatal("commit")
	}
	if !g.MatchesURL("https://www.youtube.com/watch?v=AAAAAAAAAAA") {
		t.Fatal("should match")
	}
	if g.SyncToURL("https://www.youtube.com/watch?v=AAAAAAAAAAA", true) {
		t.Fatal("same URL should not clear")
	}
	if !g.SyncToURL("https://www.youtube.com/watch?v=BBBBBBBBBBB", true) {
		t.Fatal("changed URL should clear")
	}
	if _, ok := g.Snapshot(); ok {
		t.Fatal("result should be gone")
	}
	if g.MatchesURL("https://www.youtube.com/watch?v=AAAAAAAAAAA") {
		t.Fatal("old URL should not match")
	}

	seq = g.Begin("https://www.youtube.com/watch?v=AAAAAAAAAAA")
	if g.SyncToURL("https://www.youtube.com/watch?v=AAAAAAAAAAA", true) {
		t.Fatal("pending same URL should not clear")
	}
	if !g.SyncToURL("https://youtu.be/BBBBBBBBBBB", true) {
		t.Fatal("pending different URL should clear")
	}
	if g.Commit(seq, Fetched{SourceURL: "https://www.youtube.com/watch?v=AAAAAAAAAAA", Title: "late"}) {
		t.Fatal("stale pending commit succeeded")
	}
}

func TestGateInvalidTypedURLClearsFetched(t *testing.T) {
	var g Gate
	seq := g.Begin("https://www.youtube.com/watch?v=AAAAAAAAAAA")
	_ = g.Commit(seq, Fetched{SourceURL: "https://www.youtube.com/watch?v=AAAAAAAAAAA", Title: "A"})
	if !g.SyncToURL("", false) {
		t.Fatal("invalid/empty URL should clear fetched data")
	}
	if _, ok := g.Snapshot(); ok {
		t.Fatal("expected clear")
	}
}
