package download

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"youtubemp3/internal/metadata"
)

func TestFakeDownloaderPipeline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake script is POSIX")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-ytdlp")
	body := `#!/bin/sh
set -e
out=""
printtitle=0
printmeta=0
prev=""
for a in "$@"; do
  if [ "$prev" = "-o" ]; then out="$a"; fi
  if [ "$a" = "title" ] && [ "$prev" = "--print" ]; then printtitle=1; fi
  case "$a" in
    *title,duration,thumbnail*) printmeta=1 ;;
  esac
  prev="$a"
done
if [ "$printtitle" = "1" ]; then
  echo "Fake: Song/Title?"
  exit 0
fi
if [ "$printmeta" = "1" ]; then
  echo '{"title":"Fake: Song/Title?","duration":125,"thumbnail":"https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg"}'
  exit 0
fi
if [ -z "$out" ]; then
  echo "missing -o" >&2
  exit 1
fi
# emulate yt-dlp template replacement
file=$(printf '%s' "$out" | sed 's/%(ext)s/mp3/')
printf 'ID3FAKE' > "$file"
echo "[download] 100% of 1.00KiB"
echo "[ExtractAudio] Destination: $file"
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	tools := Tools{Ytdlp: script}
	title, err := FetchTitle(context.Background(), tools, "https://www.youtube.com/watch?v=dQw4w9WgXcQ", nil)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Fake: Song/Title?" {
		t.Fatalf("title=%q", title)
	}
	job := filepath.Join(dir, "job")
	if err := os.Mkdir(job, 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "dest")
	if err := os.Mkdir(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpl := OutputTemplate(job, "Fake_ Song_Title_")
	if err := RunDownload(context.Background(), tools, "https://youtu.be/dQw4w9WgXcQ", tmpl, nil); err != nil {
		t.Fatal(err)
	}
	final, err := PlaceOutput(job, dest, title)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(final) != "Fake_ Song_Title_.mp3" {
		t.Fatalf("final=%q", final)
	}
	data, _ := os.ReadFile(final)
	if string(data) != "ID3FAKE" {
		t.Fatalf("data=%q", data)
	}
	_ = exec.Command
}

func TestFakeFetchMeta(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake script is POSIX")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-ytdlp")
	body := `#!/bin/sh
for a in "$@"; do
  case "$a" in
    --dump-single-json|*title,duration,thumbnail*)
      echo '{"title":"Hello: World?","duration":65,"thumbnail":"https://evil.example/x.jpg","formats":[{"url":"https://example.invalid/a"}]}'
      exit 0
      ;;
  esac
done
exit 1
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := FetchMeta(context.Background(), Tools{Ytdlp: script}, "https://youtu.be/dQw4w9WgXcQ", nil)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Hello: World?" {
		t.Fatalf("title=%q", info.Title)
	}
	if !info.HasDur || info.Duration != 65 {
		t.Fatalf("dur=%v", info.Duration)
	}
	// raw thumbnail may be untrusted; callers must allowlist before download.
	if info.Thumbnail != "https://evil.example/x.jpg" {
		t.Fatalf("thumb=%q", info.Thumbnail)
	}
}

func TestFetchMetaMusicThumbnailCandidates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake script is POSIX")
	}
	dir := t.TempDir()
	payload := `{
	  "title": "Colour Your Night",
	  "duration": 212,
	  "thumbnail": "https://i.ytimg.com/vi_webp/QsXnlRXZsVk/maxresdefault.webp",
	  "thumbnails": [
	    {"url": "https://i.ytimg.com/vi_webp/QsXnlRXZsVk/maxresdefault.webp", "width": 1280, "height": 720, "preference": 0},
	    {"url": "https://i.ytimg.com/vi/QsXnlRXZsVk/hqdefault.jpg", "width": 480, "height": 360, "preference": -7},
	    {"url": "https://evil.example/cover.jpg", "width": 9999, "height": 9999, "preference": 100}
	  ],
	  "formats": [{"url": "https://example.invalid/a"}]
	}`
	jsonPath := filepath.Join(dir, "info.json")
	if err := os.WriteFile(jsonPath, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "fake-ytdlp")
	body := "#!/bin/sh\ncat " + shellQuote(jsonPath) + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := FetchMeta(context.Background(), Tools{Ytdlp: script}, "https://www.youtube.com/watch?v=QsXnlRXZsVk", nil)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Colour Your Night" || !info.HasDur || info.Duration != 212 {
		t.Fatalf("%+v", info)
	}
	if len(info.Thumbs) != 3 {
		t.Fatalf("thumbs %+v", info.Thumbs)
	}
	cands := metadata.PreviewCandidates(info, "QsXnlRXZsVk")
	if len(cands) == 0 || cands[0] != "https://i.ytimg.com/vi_webp/QsXnlRXZsVk/maxresdefault.webp" {
		t.Fatalf("cands %v", cands)
	}
	for _, u := range cands {
		if strings.Contains(u, "evil.example") || strings.Contains(u, "example.invalid") {
			t.Fatalf("disallowed url in candidates: %v", cands)
		}
	}
	joined := strings.Join(cands, "\n")
	if !strings.Contains(joined, "https://i.ytimg.com/vi/QsXnlRXZsVk/maxresdefault.jpg") {
		t.Fatalf("missing jpeg fallback: %v", cands)
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
