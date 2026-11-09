package folderui

import (
	"testing"

	"youtubemp3/internal/settings"
)

func TestSelectPromotesAndRestoresText(t *testing.T) {
	recents := []string{`C:\Music`, `D:\Other`, `E:\Stick`}
	path, ok := PathFromClick(recents, 1, `D:\Other`)
	if !ok || path != `D:\Other` {
		t.Fatalf("click %q %v", path, ok)
	}
	st := settings.Settings{RecentFolders: append([]string(nil), recents...), OutputFolder: `C:\Music`}
	st.ApplyChosenFolder(path, true)
	view := DisplayOf(st.RecentFolders, st.OutputFolder)
	if view.Text != `D:\Other` {
		t.Fatalf("cleared or lost text: %+v", view)
	}
	if view.Index != 0 || view.Model[0] != `D:\Other` {
		t.Fatalf("not promoted: %+v", view)
	}
}

func TestResetIndexDoesNotClear(t *testing.T) {
	if !ShouldIgnoreIndexEvent(false, -1) {
		t.Fatal("index -1 is the SetModel reset")
	}
	if path, ok := PathFromClick([]string{`C:\Music`}, -1, ""); ok || path != "" {
		t.Fatalf("reset must not yield a path: %q %v", path, ok)
	}
	view := DisplayOf([]string{`C:\Music`, `D:\Other`}, `C:\Music`)
	if view.Text == "" {
		t.Fatal("display text must survive a model rebuild")
	}
}

func TestUnavailableKeepsExactPath(t *testing.T) {
	unavailable := `E:\Unplugged`
	st := settings.Settings{RecentFolders: []string{`C:\Music`, unavailable}, OutputFolder: `C:\Music`}
	path, ok := PathFromClick(st.RecentFolders, 1, unavailable)
	if !ok {
		t.Fatal("select unavailable")
	}
	st.ApplyChosenFolder(path, false)
	view := DisplayOf(st.RecentFolders, st.OutputFolder)
	if view.Text != unavailable {
		t.Fatalf("unavailable path lost: %+v", view)
	}
}

func TestManualEditUnchangedByIgnoreAndDedup(t *testing.T) {
	if !AlreadyShowing(`C:\Music`, `C:\Music`) {
		t.Fatal("already showing")
	}
	view := DisplayOf([]string{`C:\Music`}, `D:\Typed`)
	if view.Index != -1 || view.Text != `D:\Typed` {
		t.Fatalf("typed path not in recents should still display: %+v", view)
	}
}

func TestUpdatingFlagDropsReentrantChange(t *testing.T) {
	if !ShouldIgnoreIndexEvent(true, 0) {
		t.Fatal("updating rebuild must ignore CurrentIndexChanged")
	}
}
