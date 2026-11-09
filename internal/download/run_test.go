package download

import (
	"os"
	"path/filepath"
	"testing"

	"youtubemp3/internal/sanitize"
)

func TestPlaceOutputUsesTitleAndAvoidsOverwrite(t *testing.T) {
	tempRoot := t.TempDir()
	job := filepath.Join(tempRoot, "job1")
	dest := t.TempDir()
	if err := os.MkdirAll(job, 0o755); err != nil {
		t.Fatal(err)
	}
	title := `My Video: Title`
	base := sanitize.Filename(title)
	src := filepath.Join(job, base+".mp3")
	if err := os.WriteFile(src, []byte("audio-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(dest, base+".mp3")
	if err := os.WriteFile(existing, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PlaceOutput(job, dest, title)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != base+" (1).mp3" {
		t.Fatalf("got %q", got)
	}
	old, _ := os.ReadFile(existing)
	if string(old) != "old" {
		t.Fatal("overwrote existing dest file")
	}
	data, _ := os.ReadFile(got)
	if string(data) != "audio-bytes" {
		t.Fatalf("copied %q", data)
	}
}

func TestSafeRemoveJobDir(t *testing.T) {
	root := t.TempDir()
	job := filepath.Join(root, "job-abc")
	if err := os.MkdirAll(job, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(job, "x.mp3")
	if err := os.WriteFile(marker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "user-file.mp3")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SafeRemoveJobDir(outside, root); err == nil {
		t.Fatal("should refuse deleting a path outside the temp root")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("user file was touched")
	}
	if err := SafeRemoveJobDir(root, root); err == nil {
		t.Fatal("should refuse deleting the temp root")
	}
	if err := SafeRemoveJobDir(job, root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(job); !os.IsNotExist(err) {
		t.Fatal("job dir should be gone")
	}
}

func TestCollectTitle(t *testing.T) {
	got := CollectTitle("WARNING: foo\nA Cool Song\n")
	if got != "A Cool Song" {
		t.Fatalf("got %q", got)
	}
}

func TestPlaceNamedOutputUsesCustomName(t *testing.T) {
	tempRoot := t.TempDir()
	job := filepath.Join(tempRoot, "job2")
	dest := t.TempDir()
	if err := os.MkdirAll(job, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(job, "audio.mp3"), []byte("audio-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := PlaceNamedOutput(job, dest, "My Custom Name.mp3")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "My Custom Name.mp3" {
		t.Fatalf("got %q", got)
	}
}
