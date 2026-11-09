package sanitize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilenameSanitizesWindowsChars(t *testing.T) {
	got := Filename(`My Video: "Best/Hits"? <>|*`)
	if strings.ContainsAny(got, `<>:"/\|?*`) {
		t.Fatalf("unsafe name %q", got)
	}
	if got == "" {
		t.Fatal("empty")
	}
}

func TestFilenameReservedAndEmpty(t *testing.T) {
	if Filename("CON") != "CON_" {
		t.Fatalf("got %q", Filename("CON"))
	}
	if Filename("   ") != "youtube-audio" {
		t.Fatalf("got %q", Filename("   "))
	}
	if Filename("...") != "youtube-audio" {
		t.Fatalf("got %q", Filename("..."))
	}
}

func TestFilenameTrimsTrailingDotSpace(t *testing.T) {
	got := Filename("hello. ")
	if strings.HasSuffix(got, ".") || strings.HasSuffix(got, " ") {
		t.Fatalf("got %q", got)
	}
}

func TestUniqueMP3PathAvoidsOverwrite(t *testing.T) {
	dir := t.TempDir()
	base := "Song Title"
	first, err := UniqueMP3Path(dir, base)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(first) != "Song Title.mp3" {
		t.Fatalf("first=%q", first)
	}
	if err := os.WriteFile(first, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := UniqueMP3Path(dir, base)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(second) != "Song Title (1).mp3" {
		t.Fatalf("second=%q", second)
	}
	if err := os.WriteFile(second, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	third, err := UniqueMP3Path(dir, base)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(third) != "Song Title (2).mp3" {
		t.Fatalf("third=%q", third)
	}
	// original untouched
	data, _ := os.ReadFile(first)
	if string(data) != "one" {
		t.Fatalf("overwrote original: %q", data)
	}
}

func TestOutputBaseStripsMp3AndFallsBack(t *testing.T) {
	if OutputBase("", "My Video: Title") != Filename("My Video: Title") {
		t.Fatalf("blank fallback got %q", OutputBase("", "My Video: Title"))
	}
	if OutputBase("   ", "Hello") != Filename("Hello") {
		t.Fatal("whitespace should fall back")
	}
	got := OutputBase("Cool Song.mp3", "Other")
	if got != "Cool Song" {
		t.Fatalf("got %q", got)
	}
	got = OutputBase("Cool Song.MP3", "Other")
	if got != "Cool Song" {
		t.Fatalf("case suffix got %q", got)
	}
	got = OutputBase("Cool Song.mp3.mp3", "Other")
	if got != "Cool Song" {
		t.Fatalf("double suffix got %q", got)
	}
	got = OutputBase(`bad:name?*`, "Other")
	if strings.ContainsAny(got, `<>:"/\|?*`) {
		t.Fatalf("unsafe %q", got)
	}
	if OutputBase("...", "Fallback Title") != Filename("Fallback Title") && OutputBase("...", "Fallback Title") != "youtube-audio" {
		// "..." sanitizes to youtube-audio because it is only dots
		t.Fatalf("got %q", OutputBase("...", "Fallback Title"))
	}
	if OutputBase("...", "Fallback Title") != "youtube-audio" {
		t.Fatalf("dots should sanitize to youtube-audio, got %q", OutputBase("...", "Fallback Title"))
	}
}

func TestSuggestedFilenameStripsCJKWhenLatinRemains(t *testing.T) {
	got := SuggestedFilename("《记忆管理局》False Memory | OP DOUDOU《忘记了也没关系》")
	if got != "False Memory - OP DOUDOU" {
		t.Fatalf("got %q", got)
	}
}

func TestSuggestedFilenameKeepsAllCJK(t *testing.T) {
	got := SuggestedFilename("记忆管理局")
	if got != "记忆管理局" {
		t.Fatalf("got %q", got)
	}
}

func TestSuggestedFilenameKeepsSpanish(t *testing.T) {
	got := SuggestedFilename("Canción de la niña Nº 2")
	if got != "Canción de la niña Nº 2" {
		t.Fatalf("got %q", got)
	}
}

func TestOutputBaseManualKeepsCJK(t *testing.T) {
	got := OutputBase("《记忆管理局》demo", "False Memory")
	if got != "《记忆管理局》demo" {
		t.Fatalf("manual should keep CJK, got %q", got)
	}
	if OutputBase("", "《记忆管理局》False Memory | OP DOUDOU《忘记了也没关系》") != "False Memory - OP DOUDOU" {
		t.Fatalf("blank user should suggest cleaned name, got %q", OutputBase("", "《记忆管理局》False Memory | OP DOUDOU《忘记了也没关系》"))
	}
}
