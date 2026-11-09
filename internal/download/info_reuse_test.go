package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"youtubemp3/internal/infocache"
	"youtubemp3/internal/yturl"
)

// TestPreviewThenDownloadUsesCachedInfo is the shared preview-to-download path:
// FetchMeta (--dump-single-json, writes infocache) then Lookup then
// RunDownloadMaybeInfo. main.startSaveFooter returns that Lookup path, and
// runSave passes it here. A fresh preview must not extract from the URL again.
func TestPreviewThenDownloadUsesCachedInfo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake script is POSIX")
	}
	videoURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	id, ok := yturl.VideoID(videoURL)
	if !ok {
		t.Fatal("id")
	}
	future := time.Now().Add(2 * time.Hour).Unix()
	raw := fmt.Sprintf(`{"title":"Cached Song","duration":12,"thumbnail":"https://i.ytimg.com/vi/%s/hqdefault.jpg","formats":[{"url":"https://example.invalid/a?expire=%d"}]}`, id, future)

	log := runPreviewThenDownload(t, videoURL, raw, "")
	if strings.Join(log, ",") != "dump,load" {
		t.Fatalf("invocations %v", log)
	}
	path, _, reason, hit := infocache.Lookup(id)
	if !hit || reason != "ok" || path == "" {
		t.Fatalf("cache after download hit=%v reason=%s", hit, reason)
	}
}

func TestDumpLineSurvivesTrailingBraceAndIsReused(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake script is POSIX")
	}
	videoURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	future := time.Now().Add(2 * time.Hour).Unix()
	raw := fmt.Sprintf("[youtube] Extracting URL: %s\n"+
		`{"title":"Cached Song","duration":12,"formats":[{"url":"https://example.invalid/a?expire=%d"}]}`+
		"\nERROR: helper note }\n", videoURL, future)
	log := runPreviewThenDownload(t, videoURL, raw, "")
	if strings.Join(log, ",") != "dump,load" {
		t.Fatalf("trailing brace dropped the cache: %v", log)
	}
	id, _ := yturl.VideoID(videoURL)
	if infocache.SavedResult(id) != "ok" {
		t.Fatalf("save result %q", infocache.SavedResult(id))
	}
}

func TestReadLinesKeepsDumpOverOneMegabyte(t *testing.T) {
	payload := `{"title":"Cached Song","duration":1,"pad":"` + strings.Repeat("a", 1500*1024) + `"}`
	var got string
	if err := ReadLines(strings.NewReader(payload+"\n"), func(line string) { got = line }); err != nil {
		t.Fatal(err)
	}
	if len(got) < 1500*1024 {
		t.Fatalf("dropped long dump line: %d", len(got))
	}
}

func TestExpiredSignedURLRefetchesInsteadOfReuse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake script is POSIX")
	}
	videoURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	id, _ := yturl.VideoID(videoURL)
	soon := time.Now().Add(10 * time.Second).Unix()
	raw := fmt.Sprintf(`{"title":"Cached Song","duration":12,"formats":[{"url":"https://example.invalid/a?expire=%d"}]}`, soon)
	log := runPreviewThenDownload(t, videoURL, raw, "")
	if strings.Join(log, ",") != "dump,url" {
		t.Fatalf("near-expiry must extract again, invocations %v", log)
	}
	if _, err := os.Stat(filepath.Join(infocacheDir(t), id+".json")); !os.IsNotExist(err) {
		t.Fatal("expired cache file should be removed")
	}
}

func TestLoadInfoFailureRetriesURLOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake script is POSIX")
	}
	videoURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	id, _ := yturl.VideoID(videoURL)
	future := time.Now().Add(2 * time.Hour).Unix()
	raw := fmt.Sprintf(`{"title":"Cached Song","duration":12,"formats":[{"url":"https://example.invalid/a?expire=%d"}]}`, future)
	log := runPreviewThenDownload(t, videoURL, raw, "fail-once")
	if strings.Join(log, ",") != "dump,load,url" {
		t.Fatalf("want one failed reuse then one URL retry, invocations %v", log)
	}
	if _, err := os.Stat(filepath.Join(infocacheDir(t), id+".json")); !os.IsNotExist(err) {
		t.Fatal("failed info json should be removed before the URL retry")
	}
}

func infocacheDir(t *testing.T) string {
	t.Helper()
	dir, err := infocache.Dir()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// runPreviewThenDownload runs FetchMeta then the same Lookup decision as
// startSaveFooter, then RunDownloadMaybeInfo. marker "fail-once" makes the
// first --load-info-json exit 1.
func runPreviewThenDownload(t *testing.T, videoURL, infoJSON, marker string) []string {
	t.Helper()
	dir := t.TempDir()
	infocache.UseDir(dir)
	t.Cleanup(func() { infocache.UseDir("") })

	jsonPath := filepath.Join(dir, "info.json")
	if err := os.WriteFile(jsonPath, []byte(infoJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "calls.txt")
	failPath := filepath.Join(dir, "fail-once")
	if marker == "fail-once" {
		if err := os.WriteFile(failPath, []byte("1"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	script := filepath.Join(dir, "fake-ytdlp")
	body := `#!/bin/sh
dump=0
load=0
url=0
out=""
prev=""
for a in "$@"; do
  if [ "$prev" = "--load-info-json" ]; then load=1; fi
  if [ "$prev" = "-o" ]; then out="$a"; fi
  case "$a" in
    --dump-single-json) dump=1 ;;
    https://*|http://*) url=1 ;;
  esac
  prev="$a"
done
if [ "$dump" = 1 ]; then
  echo dump >> "$LOG"
  cat "$JSON"
  exit 0
fi
if [ "$load" = 1 ]; then
  echo load >> "$LOG"
  if [ -f "$FAIL" ]; then
    rm -f "$FAIL"
    echo fail >&2
    exit 1
  fi
  file=$(printf '%s' "$out" | sed 's/%(ext)s/mp3/')
  printf 'ID3' > "$file"
  exit 0
fi
if [ "$url" = 1 ]; then
  echo url >> "$LOG"
  file=$(printf '%s' "$out" | sed 's/%(ext)s/mp3/')
  printf 'ID3' > "$file"
  exit 0
fi
echo other >> "$LOG"
exit 1
`
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	tools := Tools{Ytdlp: script}
	t.Setenv("LOG", logPath)
	t.Setenv("JSON", jsonPath)
	t.Setenv("FAIL", failPath)

	if _, err := FetchMeta(context.Background(), tools, videoURL, nil); err != nil {
		t.Fatal(err)
	}
	infoPath := ""
	if id, ok := yturl.VideoID(videoURL); ok {
		path, _, _, hit := infocache.Lookup(id)
		if hit {
			infoPath = path
		}
	}
	job := filepath.Join(dir, "job")
	if err := os.Mkdir(job, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpl := OutputTemplate(job, "audio")
	if err := RunDownloadMaybeInfo(context.Background(), tools, videoURL, infoPath, tmpl, nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
