package metadata

import (
	"context"
	"os"
	"testing"
	"time"

	"youtubemp3/internal/uidim"
)

// TestLivePublicMusicThumb hits the public stills for QsXnlRXZsVk.
// It does not download audio. Set YTMP3_LIVE_THUMB=1 to run it.
func TestLivePublicMusicThumb(t *testing.T) {
	if os.Getenv("YTMP3_LIVE_THUMB") == "" {
		t.Skip("set YTMP3_LIVE_THUMB=1 to fetch public stills for QsXnlRXZsVk")
	}
	info := Info{
		Thumbnail: "https://i.ytimg.com/vi_webp/QsXnlRXZsVk/maxresdefault.webp",
		Thumbs: []Thumb{
			{URL: "https://i.ytimg.com/vi_webp/QsXnlRXZsVk/maxresdefault.webp", Width: 1280, Height: 720, Preference: 0, HasPref: true},
			{URL: "https://i.ytimg.com/vi/QsXnlRXZsVk/hqdefault.jpg", Width: 480, Height: 360, Preference: -7, HasPref: true},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	img, used, err := LoadPreview(ctx, NewThumbnailClient(), PreviewCandidates(info, "QsXnlRXZsVk"))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() < 2 || b.Dy() < 2 {
		t.Fatalf("empty image from %s", used)
	}
	x, y, w, h := uidim.Contain(uidim.ThumbWidth, uidim.ThumbHeight, b.Dx(), b.Dy())
	if w < 1 || h < 1 || x+w > uidim.ThumbWidth || y+h > uidim.ThumbHeight {
		t.Fatalf("contain %d,%d %dx%d for %dx%d from %s", x, y, w, h, b.Dx(), b.Dy(), used)
	}
	t.Logf("used %s decoded %dx%d contain %d,%d %dx%d", used, b.Dx(), b.Dy(), x, y, w, h)
}
