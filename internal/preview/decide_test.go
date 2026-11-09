package preview

import "testing"

func TestDecideFetchOnlyWhenCompleteAndNew(t *testing.T) {
	const complete = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	const other = "https://youtu.be/AAAAAAAAAAA"
	const otherCanon = "https://www.youtube.com/watch?v=AAAAAAAAAAA"
	op, url := Decide(complete, "", "")
	if op != OpFetch || url != complete {
		t.Fatalf("got %s %q", op, url)
	}
	op, _ = Decide(complete, complete, "")
	if op != OpNone {
		t.Fatalf("pending same should wait, got %s", op)
	}
	op, _ = Decide(complete, "", complete)
	if op != OpNone {
		t.Fatalf("already committed should not refetch, got %s", op)
	}
	op, url = Decide(other, "", complete)
	if op != OpFetch || url != otherCanon {
		t.Fatalf("new complete URL should fetch, got %s %q", op, url)
	}
	op, _ = Decide("https://youtu.be/dQw4w9WgXcQ?si=xx&list=PLxx", "", complete)
	if op != OpNone {
		t.Fatalf("same video after canonicalize should not refetch, got %s", op)
	}
}

func TestDecideNoFetchOnIncompleteOrInvalid(t *testing.T) {
	const complete = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	op, _ := Decide("https://www.youtube.com/watch?v=dQw", "", "")
	if op != OpNone {
		t.Fatalf("incomplete without preview: %s", op)
	}
	op, _ = Decide("https://www.youtube.com/watch?v=dQw", "", complete)
	if op != OpClear {
		t.Fatalf("incomplete after preview should clear: %s", op)
	}
	op, _ = Decide("not a url", complete, "")
	if op != OpClear {
		t.Fatalf("invalid during pending should clear: %s", op)
	}
	op, _ = Decide("", "", "")
	if op != OpNone {
		t.Fatalf("empty idle: %s", op)
	}
}
