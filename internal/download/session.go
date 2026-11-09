package download

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"youtubemp3/internal/infocache"
	"youtubemp3/internal/metadata"
	"youtubemp3/internal/procutil"
	"youtubemp3/internal/sanitize"
	"youtubemp3/internal/yturl"
)

func NewJobDir(tempRoot string) (string, error) {
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return "", err
	}
	job := fmt.Sprintf("job-%d-%d", time.Now().UnixNano(), os.Getpid())
	path := tempRoot + string(os.PathSeparator) + job
	if err := os.Mkdir(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func FetchMeta(ctx context.Context, t Tools, videoURL string, onLine func(string)) (metadata.Info, error) {
	if _, err := yturl.Validate(videoURL); err != nil {
		return metadata.Info{}, err
	}
	args := MetadataArgs(t, videoURL)
	out, err := runLogged(ctx, t.Ytdlp, args, onLine)
	if err != nil {
		return metadata.Info{}, fmt.Errorf("could not read the video details: %w", err)
	}
	raw, info, err := metadata.ParseDump(out)
	if err != nil {
		return metadata.Info{}, err
	}
	if info.Title == "" || !sanitize.IsMostlyPrintable(info.Title) {
		info.Title = "youtube-audio"
	}
	if id, ok := yturl.VideoID(videoURL); ok {
		if len(raw) == 0 {
			infocache.Remember(id, "no_raw")
		} else {
			infocache.Keep(id, raw)
			_, note, err := infocache.Save(id, raw)
			if note == "" {
				note = infocache.SaveClass(err)
			}
			infocache.Remember(id, note)
		}
	}
	return info, nil
}

func FetchTitle(ctx context.Context, t Tools, videoURL string, onLine func(string)) (string, error) {
	if _, err := yturl.Validate(videoURL); err != nil {
		return "", err
	}
	args := TitleArgs(t, videoURL)
	out, err := runLogged(ctx, t.Ytdlp, args, onLine)
	if err != nil {
		return "", fmt.Errorf("could not read the video title: %w", err)
	}
	title := CollectTitle(out)
	if title == "" {
		title = "youtube-audio"
	}
	if !sanitize.IsMostlyPrintable(title) {
		title = "youtube-audio"
	}
	return title, nil
}

func RunDownload(ctx context.Context, t Tools, videoURL, outputTemplate string, onLine func(string)) error {
	if _, err := yturl.Validate(videoURL); err != nil {
		return err
	}
	return runDownload(ctx, t, videoURL, outputTemplate, onLine)
}

// RunDownloadMaybeInfo uses a cached --load-info-json file when path is valid,
// otherwise extracts from videoURL. If the cached file fails, it retries once from the URL.
func RunDownloadMaybeInfo(ctx context.Context, t Tools, videoURL, infoJSON, outputTemplate string, onLine func(string)) error {
	if infoJSON != "" && infocache.ValidatePath(infoJSON) == nil {
		if _, err := os.Stat(infoJSON); err == nil {
			args := DownloadArgsFromInfo(t, infoJSON, outputTemplate)
			if _, err := runLogged(ctx, t.Ytdlp, args, onLine); err == nil {
				return nil
			} else if ctx.Err() != nil {
				return fmt.Errorf("download failed: %w", err)
			}
			if id, ok := yturl.VideoID(videoURL); ok {
				infocache.Remove(id)
			}
		}
	}
	return RunDownload(ctx, t, videoURL, outputTemplate, onLine)
}

// runDownload is the production subprocess path after URL policy.
// Tests inject a local HTTP fixture here so Validate (YouTube-only) is not bypassed in the app.
func runDownload(ctx context.Context, t Tools, videoURL, outputTemplate string, onLine func(string)) error {
	args := DownloadArgs(t, videoURL, outputTemplate)
	_, err := runLogged(ctx, t.Ytdlp, args, onLine)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	return nil
}

func runLogged(ctx context.Context, exe string, args []string, onLine func(string)) (string, error) {
	if err := ValidateArgv(args); err != nil {
		return "", err
	}
	cmd := procutil.CommandContext(ctx, exe, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "TERM=dumb", "PYTHONUNBUFFERED=1", "FORCE_COLOR=0")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	cleanup, err := procutil.Start(cmd)
	if err != nil {
		return "", err
	}
	var (
		buf    bytes.Buffer
		lineMu sync.Mutex
		wg     sync.WaitGroup
	)
	handle := func(line string) {
		lineMu.Lock()
		defer lineMu.Unlock()
		buf.WriteString(line)
		buf.WriteByte('\n')
		if onLine != nil {
			onLine(line)
		}
	}
	var scanErr error
	var scanMu sync.Mutex
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := ReadLines(stdout, handle); err != nil {
			scanMu.Lock()
			scanErr = err
			scanMu.Unlock()
		}
	}()
	go func() {
		defer wg.Done()
		if err := ReadLines(stderr, handle); err != nil {
			scanMu.Lock()
			scanErr = err
			scanMu.Unlock()
		}
	}()
	// os/exec: Wait closes StdoutPipe/StderrPipe, so drain both readers first.
	wg.Wait()
	waitErr := cmd.Wait()
	if cleanup != nil {
		cleanup()
	}
	if waitErr != nil {
		msg := strings.TrimSpace(buf.String())
		if msg == "" {
			return buf.String(), waitErr
		}
		return buf.String(), fmt.Errorf("%w\n%s", waitErr, lastLines(msg, 12))
	}
	if scanErr != nil {
		return buf.String(), scanErr
	}
	return buf.String(), nil
}

func lastLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

func YtdlpVersion(ctx context.Context, exe string) string {
	if exe == "" {
		return "unknown"
	}
	cmd := procutil.CommandContext(ctx, exe, "--version")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	v := strings.TrimSpace(string(out))
	if i := strings.IndexAny(v, "\r\n"); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return "unknown"
	}
	return v
}
