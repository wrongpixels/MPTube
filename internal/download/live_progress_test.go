package download

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func findYtdlp() string {
	if p := os.Getenv("YTDLP"); p != "" {
		return p
	}
	if p, err := exec.LookPath("yt-dlp"); err == nil {
		return p
	}
	for _, p := range []string{
		"/tmp/ytdlp-venv/bin/yt-dlp",
		"/usr/local/bin/yt-dlp",
		"/usr/bin/yt-dlp",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func TestLiveYtdlpProgressTemplateIntegration(t *testing.T) {
	ytdlp := findYtdlp()
	if ytdlp == "" {
		t.Skip("yt-dlp not installed")
	}
	payload := bytes.Repeat([]byte("x"), 1_600_000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		flusher, _ := w.(http.Flusher)
		const chunk = 32 * 1024
		for i := 0; i < len(payload); i += chunk {
			end := i + chunk
			if end > len(payload) {
				end = len(payload)
			}
			_, _ = w.Write(payload[i:end])
			if flusher != nil {
				flusher.Flush()
			}
			time.Sleep(70 * time.Millisecond)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.%(ext)s")
	args := []string{
		"--ignore-config",
		"--no-playlist",
		"--no-mtime",
		"--newline",
		"--progress",
		"--progress-template", DownloadProgressTemplate,
		"--progress-template", PostprocessProgressTemplate,
		"-o", dest,
		"--", srv.URL + "/media.bin",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var lines []string
	var view View
	var sawMid, saw100, sawSpeed bool
	out, err := runLogged(ctx, ytdlp, args, func(line string) {
		lines = append(lines, line)
		view = UpdateView(view, line)
		if view.HasPercent && view.Percent > 0 && view.Percent < 100 {
			sawMid = true
		}
		if view.HasPercent && view.Percent == 100 {
			saw100 = true
		}
		if view.Speed != "" {
			sawSpeed = true
		}
	})
	if err != nil {
		t.Fatalf("yt-dlp: %v\n%s", err, out)
	}
	if !sawMid || !saw100 {
		t.Fatalf("need intermediate and 100%% progress; view=%+v lines=%q", view, lines)
	}
	if !sawSpeed {
		t.Fatalf("expected numeric speed; view=%+v lines=%q", view, lines)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "out.*"))
	if len(matches) == 0 {
		t.Fatalf("expected downloaded file in %s; output:\n%s", dir, out)
	}
	info, err := os.Stat(matches[0])
	if err != nil || info.Size() == 0 {
		t.Fatalf("output file: %v size=%v", err, info)
	}
}

func TestReadLinesSplitsCarriageReturn(t *testing.T) {
	var got []string
	ReadLines(strings.NewReader("a\rb\r\nc\n"), func(line string) {
		got = append(got, line)
	})
	if strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("got %#v", got)
	}
}
