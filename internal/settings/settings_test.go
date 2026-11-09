package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathEqualWindowsStyle(t *testing.T) {
	pairs := [][2]string{
		{`E:\Music\YouTube`, `e:/music/youtube`},
		{`E:\Music\YouTube\`, `E:\Music\YouTube`},
		{`C:`, `C:\`},
		{`C:\`, `c:/`},
	}
	for _, p := range pairs {
		if !PathEqual(p[0], p[1]) {
			t.Fatalf("expected equal: %q and %q (norm %q vs %q)", p[0], p[1], normalizeForCompare(p[0]), normalizeForCompare(p[1]))
		}
	}
	if PathEqual(`D:\Music`, `E:\Music`) {
		t.Fatal("different drives should not be equal")
	}
}

func TestPromoteDedupAndCap(t *testing.T) {
	var recents []string
	for i := 0; i < 20; i++ {
		recents = Promote(recents, `E:\folder`+string(rune('A'+i%20))+`\Music`)
	}
	// Build 16 distinct paths more clearly
	recents = nil
	for i := 0; i < 16; i++ {
		recents = Promote(recents, filepath.Join("E:\\lib", string(rune('a'+i))))
	}
	if len(recents) != MaxRecent {
		t.Fatalf("len=%d want %d: %#v", len(recents), MaxRecent, recents)
	}
	// Most recently promoted is first: last was E:\lib + 'a'+15 = 'p'
	if !PathEqual(recents[0], `E:\lib\p`) {
		t.Fatalf("first=%q", recents[0])
	}
	// Promote an older entry
	recents = Promote(recents, `E:\lib\a`)
	if !PathEqual(recents[0], `E:\lib\a`) {
		t.Fatalf("promoted first=%q", recents[0])
	}
	countA := 0
	for _, r := range recents {
		if PathEqual(r, `E:\lib\a`) {
			countA++
		}
	}
	if countA != 1 {
		t.Fatalf("duplicate count=%d", countA)
	}
	if len(recents) != MaxRecent {
		t.Fatalf("after promote len=%d", len(recents))
	}
}

func TestApplyChosenValidPromotes(t *testing.T) {
	st := Settings{
		OutputFolder:  `E:\old`,
		RecentFolders: []string{`E:\old`, `F:\other`, `G:\more`},
	}
	st.ApplyChosenFolder(`f:\other`, true)
	if st.OutputFolder != `f:\other` {
		t.Fatalf("output=%q", st.OutputFolder)
	}
	if !PathEqual(st.RecentFolders[0], `f:\other`) {
		t.Fatalf("recents=%v", st.RecentFolders)
	}
	// stored spelling of the newly chosen path is what the user just chose
	if st.RecentFolders[0] != `f:\other` {
		t.Fatalf("expected exact chosen spelling, got %q", st.RecentFolders[0])
	}
}

func TestApplyChosenUnavailableRetainsExactPath(t *testing.T) {
	unavailable := `E:\Music\FromUSB`
	st := Settings{
		OutputFolder:  unavailable,
		RecentFolders: []string{`C:\Users\Me\Music`, unavailable, `D:\Backup`},
	}
	copied := append([]string(nil), st.RecentFolders...)
	st.ApplyChosenFolder(unavailable, false)
	if st.OutputFolder != unavailable {
		t.Fatalf("output was rewritten to %q", st.OutputFolder)
	}
	if len(st.RecentFolders) != len(copied) {
		t.Fatalf("recents changed length: %v", st.RecentFolders)
	}
	for i := range copied {
		if st.RecentFolders[i] != copied[i] {
			t.Fatalf("recents[%d] changed from %q to %q", i, copied[i], st.RecentFolders[i])
		}
	}
}

func TestApplyChosenUnavailableNewPathIsRemembered(t *testing.T) {
	st := Settings{RecentFolders: []string{`C:\Music`}}
	st.ApplyChosenFolder(`E:\Stick\Mixes`, false)
	if st.OutputFolder != `E:\Stick\Mixes` {
		t.Fatalf("output=%q", st.OutputFolder)
	}
	found := false
	for _, r := range st.RecentFolders {
		if r == `E:\Stick\Mixes` {
			found = true
		}
	}
	if !found {
		t.Fatalf("new unavailable path was not retained: %v", st.RecentFolders)
	}
}

func TestSaveLoadPreservesUnavailablePathExactly(t *testing.T) {
	dir := t.TempDir()
	store := Store{Path: filepath.Join(dir, "settings.json")}
	unavailable := `E:\Does\Not\Exist\YouTube Mix`
	st := Settings{
		OutputFolder:  unavailable,
		RecentFolders: []string{unavailable, `C:\Present`},
	}
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.OutputFolder != unavailable {
		t.Fatalf("loaded output=%q", got.OutputFolder)
	}
	if got.RecentFolders[0] != unavailable {
		t.Fatalf("loaded recents=%v", got.RecentFolders)
	}
	if err := FolderAvailable(got.OutputFolder); err == nil {
		t.Fatal("expected unavailable")
	} else if !errors.Is(err, ErrFolderUnavailable) {
		t.Fatalf("err=%v", err)
	}
	msg := UserFolderError(got.OutputFolder, FolderAvailable(got.OutputFolder))
	if !strings.Contains(msg, unavailable) || !strings.Contains(msg, "not changed") {
		t.Fatalf("message=%q", msg)
	}
	// Reload again and confirm still exact
	got2, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got2.OutputFolder != unavailable {
		t.Fatalf("second load rewritten: %q", got2.OutputFolder)
	}
}

func TestFolderAvailableDoesNotCreate(t *testing.T) {
	parent := t.TempDir()
	missing := filepath.Join(parent, "not-created-subdir")
	if err := FolderAvailable(missing); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("directory was created: %v", err)
	}
	// existing dir is fine
	if err := FolderAvailable(parent); err != nil {
		t.Fatal(err)
	}
}

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "nope.json")}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.OutputFolder != "" || len(got.RecentFolders) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestSaveDoesNotRewriteJSONPathString(t *testing.T) {
	dir := t.TempDir()
	store := Store{Path: filepath.Join(dir, "settings.json")}
	raw := `E:\Music\USB Folder`
	if err := store.Save(Settings{OutputFolder: raw}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatal(err)
	}
	var probe map[string]any
	if err := json.Unmarshal(data, &probe); err != nil {
		t.Fatal(err)
	}
	if probe["output_folder"] != raw {
		t.Fatalf("json value=%v", probe["output_folder"])
	}
}

func TestLanguageDoesNotDropRecentsOrUnavailablePath(t *testing.T) {
	dir := t.TempDir()
	store := Store{Path: filepath.Join(dir, "settings.json")}
	unavailable := `E:\Stick\Mixes`
	st := Settings{
		OutputFolder:  unavailable,
		RecentFolders: []string{unavailable, `C:\Music`},
	}
	if err := store.Save(st); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Language != "" {
		t.Fatalf("old files should have empty language, got %q", got.Language)
	}
	if got.OutputFolder != unavailable || got.RecentFolders[0] != unavailable {
		t.Fatalf("paths changed: %+v", got)
	}
	got.Language = "es"
	if err := store.Save(got); err != nil {
		t.Fatal(err)
	}
	got2, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got2.Language != "es" {
		t.Fatalf("language=%q", got2.Language)
	}
	if got2.OutputFolder != unavailable {
		t.Fatalf("output rewritten: %q", got2.OutputFolder)
	}
	if len(got2.RecentFolders) != 2 || got2.RecentFolders[0] != unavailable {
		t.Fatalf("recents=%v", got2.RecentFolders)
	}
}
