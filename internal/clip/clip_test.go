package clip

import "testing"

func TestNormalizeFirstLineAndBOM(t *testing.T) {
	got := Normalize("\ufeffhttps://youtu.be/dQw4w9WgXcQ\r\nextra")
	if got != "https://youtu.be/dQw4w9WgXcQ" {
		t.Fatalf("got %q", got)
	}
	if Normalize("   \n  https://www.youtube.com/watch?v=dQw4w9WgXcQ  \n") != "https://www.youtube.com/watch?v=dQw4w9WgXcQ" {
		t.Fatal("leading blank lines")
	}
	if Normalize("   \n\t") != "" {
		t.Fatal("whitespace only")
	}
}

func TestDecidePasteDoesNotOverwriteWhenEmpty(t *testing.T) {
	cur := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	next, apply := DecidePaste(cur, "", false)
	if apply || next != cur {
		t.Fatalf("no-text: apply=%v next=%q", apply, next)
	}
	next, apply = DecidePaste(cur, "   \n", true)
	if apply || next != cur {
		t.Fatalf("blank: apply=%v next=%q", apply, next)
	}
	next, apply = DecidePaste(cur, "https://youtu.be/AAAAAAAAAAA", true)
	if !apply || next != "https://youtu.be/AAAAAAAAAAA" {
		t.Fatalf("text: apply=%v next=%q", apply, next)
	}
}
