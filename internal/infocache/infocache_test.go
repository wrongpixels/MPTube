package infocache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLookupAndTTL(t *testing.T) {
	dir := t.TempDir()
	UseDir(dir)
	t.Cleanup(func() { UseDir("") })

	id := "dQw4w9WgXcQ"
	raw := []byte(`{"id":"dQw4w9WgXcQ","title":"Demo","formats":[{"url":"https://example.invalid/a"}]}`)
	p, note, err := Save(id, raw)
	if note != "ok" {
		t.Fatalf("note=%s", note)
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePath(p); err != nil {
		t.Fatal(err)
	}
	got, age, reason, ok := Lookup(id)
	if !ok || reason != "ok" || got != p {
		t.Fatalf("lookup %q %s %v %v", got, reason, age, ok)
	}
	if age < 0 || age > time.Minute {
		t.Fatalf("age=%v", age)
	}
}

func TestLookupExpiredURL(t *testing.T) {
	dir := t.TempDir()
	UseDir(dir)
	t.Cleanup(func() { UseDir("") })
	id := "abcdefghijk"
	expired := time.Now().Add(-time.Minute).Unix()
	raw := []byte(fmt.Sprintf(`{"title":"x","formats":[{"url":"https://example.invalid/v?expire=%d"}]}`, expired))
	if _, _, err := Save(id, raw); err != nil {
		t.Fatal(err)
	}
	_, _, reason, ok := Lookup(id)
	if ok || reason != "expired_url" {
		t.Fatalf("want expired_url, got ok=%v reason=%s", ok, reason)
	}
	if _, err := os.Stat(filepath.Join(dir, id+".json")); !os.IsNotExist(err) {
		t.Fatal("expired file should be removed")
	}
}

func TestCleanupLeavesForeignFiles(t *testing.T) {
	dir := t.TempDir()
	UseDir(dir)
	t.Cleanup(func() { UseDir("") })
	foreign := filepath.Join(dir, "not-ours.txt")
	if err := os.WriteFile(foreign, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldID := filepath.Join(dir, "aaaaaaaaaaa.json")
	if err := os.WriteFile(oldID, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-TTL - time.Minute)
	if err := os.Chtimes(oldID, past, past); err != nil {
		t.Fatal(err)
	}
	Cleanup()
	if _, err := os.Stat(foreign); err != nil {
		t.Fatal("must not delete foreign file")
	}
	if _, err := os.Stat(oldID); !os.IsNotExist(err) {
		t.Fatal("old own file should be cleaned")
	}
}

func TestRenameFailureStillLeavesTheFile(t *testing.T) {
	dir := t.TempDir()
	UseDir(dir)
	prev := renameFile
	renameFile = func(string, string) error { return os.ErrPermission }
	t.Cleanup(func() {
		renameFile = prev
		UseDir("")
	})
	id := "abcdefghijk"
	p, note, err := Save(id, []byte(`{"title":"x"}`))
	if err != nil || note != "rename_fallback" {
		t.Fatalf("note=%s err=%v", note, err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	if _, _, reason, ok := Lookup(id); !ok || reason != "ok" {
		t.Fatalf("lookup ok=%v reason=%s", ok, reason)
	}
}

func TestUnreadableCacheFileIsNotDeleted(t *testing.T) {
	dir := t.TempDir()
	UseDir(dir)
	t.Cleanup(func() { UseDir("") })
	id := "abcdefghijk"
	p := filepath.Join(dir, id+".json")
	if err := os.Mkdir(p, 0o755); err != nil {
		t.Fatal(err)
	}
	_, _, reason, ok := Lookup(id)
	if ok || reason != "unreadable" {
		t.Fatalf("ok=%v reason=%s", ok, reason)
	}
	st, err := os.Stat(p)
	if err != nil || !st.IsDir() {
		t.Fatal("unreadable cache entry was removed")
	}
}

func TestValidatePathRejectsEscape(t *testing.T) {
	dir := t.TempDir()
	UseDir(dir)
	t.Cleanup(func() { UseDir("") })
	if err := ValidatePath(filepath.Join(t.TempDir(), "dQw4w9WgXcQ.json")); err == nil {
		t.Fatal("expected reject")
	}
}
