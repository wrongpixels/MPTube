package download

import (
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

func TestProductionRunDownloadProgressAndChrome(t *testing.T) {
	ytdlp := findYtdlp()
	if ytdlp == "" {
		t.Skip("yt-dlp not installed")
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	wav := filepath.Join(dir, "tone.wav")
	gen := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=12",
		"-ar", "44100", "-ac", "2", wav)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generate wav: %v\n%s", err, out)
	}
	payload, err := os.ReadFile(wav)
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) < 200_000 {
		t.Fatalf("wav too small: %d", len(payload))
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
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

	job := filepath.Join(dir, "job")
	if err := os.Mkdir(job, 0o755); err != nil {
		t.Fatal(err)
	}
	tmpl := OutputTemplate(job, "audio")
	tools := Tools{Ytdlp: ytdlp, FFmpegDir: filepath.Dir(ffmpeg)}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	var view View
	var sawMid, saw100, sawSpeed bool
	var last Chrome
	err = runDownload(ctx, tools, srv.URL+"/tone.wav", tmpl, func(line string) {
		var ch Chrome
		view, ch = Step(view, line)
		if !ch.Changed {
			return
		}
		last = ch
		if ch.Determinate && ch.Percent > 0 && ch.Percent < 100 {
			sawMid = true
		}
		if ch.Determinate && ch.Percent == 100 {
			saw100 = true
		}
		if ch.Speed != "" {
			sawSpeed = true
		}
	})
	if err != nil {
		t.Fatalf("runDownload (production args): %v", err)
	}
	if !sawMid || !saw100 {
		t.Fatalf("app chrome never went determinate; last=%+v mid=%v hundred=%v", last, sawMid, saw100)
	}
	if !sawSpeed {
		t.Fatalf("app chrome never received speed; last=%+v", last)
	}
	matches, _ := filepath.Glob(filepath.Join(job, "*.mp3"))
	if len(matches) == 0 {
		t.Fatalf("expected mp3 from production -x path in %s", job)
	}
}

func TestRunLoggedKeepsStdoutProgressDespiteStderrNoise(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 required")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "noisy.py")
	body := "import sys, time, threading\n" +
		"def noise():\n" +
		"    for i in range(40):\n" +
		"        sys.stderr.write('WARN interleave %d [info] garbage\\n' % i)\n" +
		"        sys.stderr.flush()\n" +
		"        time.sleep(0.01)\n" +
		"th = threading.Thread(target=noise)\n" +
		"th.start()\n" +
		"for pct, down in [(5.0, 1000), (40.0, 8000), (72.5, 14500), (100.0, 20000)]:\n" +
		"    sys.stdout.write('YTMP3\\t  %.1f%%\\t%d\\t20000\\tNA\\t50000\\t1\\n' % (pct, down))\n" +
		"    sys.stdout.flush()\n" +
		"    time.sleep(0.03)\n" +
		"th.join()\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	var view View
	var percents []int
	_, err := runLogged(context.Background(), "python3", []string{script}, func(line string) {
		var ch Chrome
		view, ch = Step(view, line)
		if ch.Changed && ch.Determinate {
			percents = append(percents, ch.Percent)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := ""
	for i, p := range percents {
		if i > 0 {
			joined += ","
		}
		joined += strconv.Itoa(p)
	}
	if len(percents) < 3 || percents[len(percents)-1] != 100 {
		t.Fatalf("progress lines lost or interleaved: %v", percents)
	}
	var sawMid bool
	for _, p := range percents {
		if p > 0 && p < 100 {
			sawMid = true
		}
	}
	if !sawMid {
		t.Fatalf("missing intermediate: %v", percents)
	}
	_ = strings.TrimSpace(joined)
}
