package reveal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExplorerSelectArgsKeepsSpacesInOneArg(t *testing.T) {
	args, err := ExplorerSelectArgs(`C:\Users\Me\Music\Song Title (1).mp3`)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 2 || !strings.EqualFold(args[0], "explorer.exe") {
		t.Fatalf("args=%v", args)
	}
	if !strings.HasPrefix(args[1], "/select,") {
		t.Fatalf("select=%q", args[1])
	}
	if strings.Contains(args[1], `"`) {
		t.Fatalf("must not shell-quote: %q", args[1])
	}
	if !strings.HasSuffix(args[1], `Song Title (1).mp3`) && !strings.Contains(args[1], "Song Title (1).mp3") {
		t.Fatalf("path missing: %q", args[1])
	}
}

func TestExplorerSelectArgsRejectsEmpty(t *testing.T) {
	if _, err := ExplorerSelectArgs("  "); err == nil {
		t.Fatal("expected error")
	}
}

func TestCheckFile(t *testing.T) {
	if CheckFile("") == nil || CheckFile("/no/such/file-xyz.mp3") == nil {
		t.Fatal("missing should fail")
	}
	dir := t.TempDir()
	if CheckFile(dir) == nil {
		t.Fatal("directory is not a file")
	}
	p := filepath.Join(dir, "Song (1).mp3")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckFile(p); err != nil {
		t.Fatal(err)
	}
}
