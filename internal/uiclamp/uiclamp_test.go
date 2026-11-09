package uiclamp

import "testing"

func TestEllipsisEnd(t *testing.T) {
	if EllipsisEnd("hello", 10) != "hello" {
		t.Fatal("short")
	}
	if got := EllipsisEnd("abcdefghij", 6); got != "abcde…" {
		t.Fatalf("got %q", got)
	}
}

func TestEllipsisLinesKeepsNewline(t *testing.T) {
	got := EllipsisLines("Downloaded!\nC:\\Users\\Me\\Music\\Very Long File Name (1).mp3", 18)
	if got == "" || got == "Downloaded!\nC:\\Users\\Me\\Music\\Very Long File Name (1).mp3" {
		t.Fatalf("should clamp path line: %q", got)
	}
	if got[:12] != "Downloaded!\n" {
		t.Fatalf("got %q", got)
	}
}
