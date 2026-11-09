package download

import (
	"os"
	"strings"
	"testing"
)

func TestDownloadArgsSafetyAndQuality(t *testing.T) {
	tools := Tools{
		Ytdlp:     `C:\cache\yt-dlp.exe`,
		FFmpegDir: `C:\cache\tools`,
		Deno:      `C:\cache\tools\deno.exe`,
	}
	url := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	tmpl := `C:\tmp\job\Song Title.%(ext)s`
	args := DownloadArgs(tools, url, tmpl)
	if args[len(args)-1] != url {
		t.Fatalf("URL must be last: %v", args)
	}
	if args[len(args)-2] != "--" {
		t.Fatalf("URL must be after --: %v", args)
	}
	need := []string{"--ignore-config", "--no-playlist", "bestaudio/best", "--audio-format", "mp3", "--audio-quality", "0", "--ffmpeg-location", "--js-runtimes", "--progress", "--newline", "--color", "never", DownloadProgressTemplate, PostprocessProgressTemplate}
	joined := strings.Join(args, " ")
	for _, n := range need {
		if !contains(args, n) && !strings.Contains(joined, n) {
			t.Fatalf("missing %q in %v", n, args)
		}
	}
	if !contains(args, "deno:"+tools.Deno) {
		t.Fatalf("deno runtime path missing: %v", args)
	}
	// pasted URL must not be able to inject extra options: it is a single argv slot
	for i, a := range args[:len(args)-1] {
		if a == url {
			t.Fatalf("URL appeared before -- at %d: %v", i, args)
		}
	}
}

func TestTitleArgs(t *testing.T) {
	args := TitleArgs(Tools{Deno: `D:\deno.exe`}, "https://youtu.be/dQw4w9WgXcQ")
	if !contains(args, "--skip-download") || !contains(args, "--print") {
		t.Fatalf("args=%v", args)
	}
	if args[len(args)-1] != "https://youtu.be/dQw4w9WgXcQ" || args[len(args)-2] != "--" {
		t.Fatalf("args=%v", args)
	}
}

func TestParseProgressLine(t *testing.T) {
	p, ok := ParseProgressLine("[download]  45.7% of 3.40MiB at 500KiB/s")
	if !ok || p != 46 {
		t.Fatalf("got %d %v", p, ok)
	}
	if _, ok := ParseProgressLine("WARNING: something"); ok {
		t.Fatal("expected no progress")
	}
	if !IsExtractLine("[ExtractAudio] Destination: foo.mp3") {
		t.Fatal("extract")
	}
}

func TestParseOutputLineTemplateAndEstimate(t *testing.T) {
	u, ok := ParseOutputLine("YTMP3	45.7	123456	270000	NA	1310720	12")
	if !ok || !u.HasPercent || u.Percent != 46 || u.Post {
		t.Fatalf("template: %+v %v", u, ok)
	}
	if u.Speed == "" || u.ETA == "" {
		t.Fatalf("speed/eta: %+v", u)
	}
	u, ok = ParseOutputLine("YTMP3	NA	5000	NA	20000	4096	8")
	if !ok || !u.HasPercent || u.Percent != 25 {
		t.Fatalf("estimate fallback: %+v %v", u, ok)
	}
	u, ok = ParseOutputLine("[download]  12.3% of ~  4.00MiB at  2.10MiB/s ETA 00:08")
	if !ok || !u.HasPercent || u.Percent != 12 || u.Speed != "2.10MiB/s" || u.ETA != "00:08" {
		t.Fatalf("classic estimate: %+v %v", u, ok)
	}
	u, ok = ParseOutputLine("[download] 100% of 3.40MiB in 00:05")
	if !ok || !u.HasPercent || u.Percent != 100 || u.Post {
		t.Fatalf("complete download is not convert: %+v %v", u, ok)
	}
	u, ok = ParseOutputLine("YTMP3POST")
	if !ok || !u.Post || u.HasPercent {
		t.Fatalf("post: %+v %v", u, ok)
	}
	if _, ok := ParseOutputLine("[download] Destination: foo.webm"); ok {
		t.Fatal("destination is not a percent line")
	}
	if pct, ok := ParseProgressLine("YTMP3POST"); ok || pct != 0 {
		t.Fatalf("extract must not look like percent: %d %v", pct, ok)
	}
}

func contains(args []string, v string) bool {
	for _, a := range args {
		if a == v {
			return true
		}
	}
	return false
}

func TestMetadataArgs(t *testing.T) {
	url := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	args := MetadataArgs(Tools{Deno: `D:\deno.exe`}, url)
	if !contains(args, "--dump-single-json") {
		t.Fatalf("args=%v", args)
	}
	if contains(args, "--print") || contains(args, "%(.{title,duration,thumbnail})j") {
		t.Fatalf("slim print should not be used for reusable metadata: %v", args)
	}
	if args[len(args)-1] != url || args[len(args)-2] != "--" {
		t.Fatalf("args=%v", args)
	}
	if !contains(args, "--ignore-config") || !contains(args, "--no-playlist") {
		t.Fatalf("args=%v", args)
	}
}

func TestDownloadArgsFromInfoSkipsURLExtract(t *testing.T) {
	info := `C:\cache\info\dQw4w9WgXcQ.json`
	args := DownloadArgsFromInfo(Tools{FFmpegDir: `C:\ff`, Deno: `C:\deno.exe`}, info, `C:\tmp\a.%(ext)s`)
	if !contains(args, "--load-info-json") || !contains(args, info) {
		t.Fatalf("missing load-info-json: %v", args)
	}
	if contains(args, "https://") {
		t.Fatalf("URL should not force a second extract: %v", args)
	}
	if !contains(args, "-x") || !contains(args, "mp3") {
		t.Fatalf("conversion flags missing: %v", args)
	}
}

func TestRecordedYtdlpHTTPProgressIsDeterminate(t *testing.T) {
	data, err := os.ReadFile("testdata/ytdlp_http_progress.txt")
	if err != nil {
		t.Fatal(err)
	}
	var view View
	var percents []int
	var sawSpeed bool
	for _, line := range strings.Split(string(data), "\n") {
		prev := view
		view = UpdateView(view, line)
		if strings.HasPrefix(strings.TrimSpace(line), "[generic]") || strings.Contains(line, "Destination:") {
			if view != prev {
				t.Fatalf("metadata changed view: %q -> %+v", line, view)
			}
		}
		if view.HasPercent {
			percents = append(percents, view.Percent)
		}
		if view.Speed != "" {
			sawSpeed = true
		}
	}
	if len(percents) < 5 {
		t.Fatalf("percents %v", percents)
	}
	var sawMid bool
	for _, p := range percents {
		if p > 0 && p < 100 {
			sawMid = true
		}
	}
	if !sawMid || percents[len(percents)-1] != 100 {
		t.Fatalf("need intermediate and 100: %v", percents)
	}
	if !sawSpeed {
		t.Fatal("expected numeric speed")
	}
	if view.Kind != "converting" {
		t.Fatalf("final kind %q", view.Kind)
	}
}

func TestOldPercentFieldNAFallsBackToBytes(t *testing.T) {
	u, ok := ParseOutputLine("YTMP3\tNA\t81920\t1638400\tNA\t500000\t2")
	if !ok || !u.HasPercent || u.Percent < 4 || u.Percent > 6 {
		t.Fatalf("byte ratio: %+v %v", u, ok)
	}
	u, ok = ParseOutputLine("YTMP3\tNA\t81920\tNA\tNA\t500000\t2")
	if !ok || u.HasPercent {
		t.Fatalf("must not invent percent without total: %+v %v", u, ok)
	}
	if u.Speed == "" || !u.HasDownloaded {
		t.Fatalf("bytes+speed: %+v", u)
	}
	v := UpdateView(View{Kind: "downloading", Progress: Progress{HasPercent: true, Percent: 40}}, "[generic] Extracting URL: http://127.0.0.1/x")
	if !v.HasPercent || v.Percent != 40 {
		t.Fatalf("metadata must not reset: %+v", v)
	}
}

func TestUpdateViewDoesNotResetOnMetadata(t *testing.T) {
	v := UpdateView(View{}, "YTMP3\t  15.9%\t261120\t1638400\tNA\t465259.6\t2")
	if v.Kind != "downloading" || !v.HasPercent || v.Percent < 15 || v.Percent > 16 || v.Speed == "" {
		t.Fatalf("got %+v", v)
	}
	v = UpdateView(v, "[info] media: Downloading 1 format(s): 0")
	if v.Kind != "downloading" || v.Percent < 15 {
		t.Fatalf("reset: %+v", v)
	}
}

func TestParseOutputLineStripsANSI(t *testing.T) {
	colored := "YTMP3\t\x1b[0;94m  15.9%\x1b[0m\t261120\t1638400\tNA\t465259.6\t2"
	u, ok := ParseOutputLine(colored)
	if !ok || !u.HasPercent || u.Percent < 15 || u.Percent > 16 || u.Speed == "" {
		t.Fatalf("ansi template: %+v %v", u, ok)
	}
	u, ok = ParseOutputLine("[download]  \x1b[0;94m12.3%\x1b[0m of  4.00MiB at  2.10MiB/s ETA 00:08")
	if !ok || !u.HasPercent || u.Percent != 12 {
		t.Fatalf("ansi classic: %+v %v", u, ok)
	}
}

func TestStepMatchesAppChrome(t *testing.T) {
	data, err := os.ReadFile("testdata/ytdlp_http_progress.txt")
	if err != nil {
		t.Fatal(err)
	}
	var view View
	var sawMid, saw100, sawSpeed, sawConv bool
	for _, line := range strings.Split(string(data), "\n") {
		var ch Chrome
		view, ch = Step(view, line)
		if !ch.Changed {
			continue
		}
		if ch.Determinate && ch.Percent > 0 && ch.Percent < 100 {
			sawMid = true
		}
		if ch.Determinate && ch.Percent == 100 {
			saw100 = true
		}
		if ch.Speed != "" {
			sawSpeed = true
		}
		if ch.StatusMode == "converting" {
			sawConv = true
		}
	}
	if !sawMid || !saw100 || !sawSpeed || !sawConv {
		t.Fatalf("chrome mid=%v 100=%v speed=%v conv=%v", sawMid, saw100, sawSpeed, sawConv)
	}
}
