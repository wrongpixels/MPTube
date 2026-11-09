package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCanonicalThumbnailURL(t *testing.T) {
	okCases := []string{
		"https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
		"https://img.youtube.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
		"https://i9.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
	}
	for _, u := range okCases {
		if _, ok := CanonicalThumbnailURL(u); !ok {
			t.Fatalf("should allow %s", u)
		}
	}
	bad := []string{
		"",
		"http://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
		"https://evil.example/x.jpg",
		"https://ytimg.com.evil.com/x.jpg",
		"https://user:pass@i.ytimg.com/vi/x/hqdefault.jpg",
		"https://github.com/foo.png",
		"javascript:alert(1)",
		"https://i.ytimg.com/" + strings.Repeat("a", 2500),
	}
	for _, u := range bad {
		if _, ok := CanonicalThumbnailURL(u); ok {
			t.Fatalf("should reject %s", u)
		}
	}
}

func TestThumbnailCandidatesSkipsUnknownHosts(t *testing.T) {
	got := ThumbnailCandidates("https://evil.example/thumb.jpg", "dQw4w9WgXcQ")
	for _, u := range got {
		if strings.Contains(u, "evil.example") {
			t.Fatalf("rejected host was kept: %v", got)
		}
	}
	want := []string{
		"https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg",
		"https://i.ytimg.com/vi/dQw4w9WgXcQ/sddefault.jpg",
		"https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg",
		"https://i.ytimg.com/vi/dQw4w9WgXcQ/mqdefault.jpg",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got %v", got)
	}
}

func TestDownloadLimitedCapsSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 64))
	}))
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "t.bin")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := downloadLimited(ctx, srv.Client(), srv.URL, dest, 16)
	if err == nil {
		t.Fatal("expected too large")
	}
	if _, stat := os.Stat(dest); !os.IsNotExist(stat) {
		t.Fatal("partial dest should not remain")
	}
}

func TestDownloadLimitedOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("jpeg-bytes"))
	}))
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "t.bin")
	if err := downloadLimited(context.Background(), srv.Client(), srv.URL, dest, 100); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(dest)
	if string(data) != "jpeg-bytes" {
		t.Fatalf("got %q", data)
	}
}

func TestDownloadThumbnailRejectsUnknownHost(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "t.jpg")
	err := DownloadThumbnail(context.Background(), "https://example.com/a.jpg", dest)
	if err == nil {
		t.Fatal("expected reject")
	}
}
