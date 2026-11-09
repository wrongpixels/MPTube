package download

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"youtubemp3/internal/procutil"
	"youtubemp3/internal/sanitize"
)

type ProgressFunc func(percent int, status string)

type Result struct {
	Title    string
	TempMP3  string
	FinalMP3 string
}

// PlaceOutput copies the downloaded MP3 to destDir using a unique title-based name.
func PlaceOutput(tempDir, destDir, title string) (finalPath string, err error) {
	return PlaceNamedOutput(tempDir, destDir, sanitize.Filename(title))
}

// PlaceNamedOutput copies the job MP3 to destDir using destBase (already user-facing).
// It never overwrites an existing destination file.
func PlaceNamedOutput(tempDir, destDir, destBase string) (finalPath string, err error) {
	base := sanitize.OutputBase(destBase, destBase)
	matches, _ := filepath.Glob(filepath.Join(tempDir, base+".mp3"))
	if len(matches) == 0 {
		// yt-dlp might write a slightly different name; accept a single mp3 in the job dir.
		alt, _ := filepath.Glob(filepath.Join(tempDir, "*.mp3"))
		matches = alt
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("the conversion finished but no MP3 file was produced")
	}
	src := matches[0]
	dest, err := sanitize.UniqueMP3Path(destDir, base)
	if err != nil {
		return "", err
	}
	if err := copyFile(src, dest); err != nil {
		if _, statErr := os.Stat(dest); statErr == nil {
			// Only remove the file we just tried to create, and only if it is the unique new name.
			_ = os.Remove(dest)
		}
		return "", err
	}
	return dest, nil
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = out.Close()
		if !ok {
			_ = os.Remove(dest)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	ok = true
	return nil
}

// SafeRemoveJobDir deletes jobDir only if it is inside tempRoot.
func SafeRemoveJobDir(jobDir, tempRoot string) error {
	if jobDir == "" || tempRoot == "" {
		return fmt.Errorf("refusing to delete: empty path")
	}
	absJob, err := filepath.Abs(jobDir)
	if err != nil {
		return err
	}
	absRoot, err := filepath.Abs(tempRoot)
	if err != nil {
		return err
	}
	sep := string(os.PathSeparator)
	rootPrefix := absRoot
	if !strings.HasSuffix(rootPrefix, sep) {
		rootPrefix += sep
	}
	if absJob != absRoot && !strings.HasPrefix(absJob, rootPrefix) {
		return fmt.Errorf("refusing to delete unexpected path %s", absJob)
	}
	if absJob == absRoot {
		return fmt.Errorf("refusing to delete temp root")
	}
	return os.RemoveAll(absJob)
}

func StartCommand(ctx context.Context, exe string, args []string, extraEnv []string) (*exec.Cmd, io.ReadCloser, error) {
	if err := ValidateArgv(args); err != nil {
		return nil, nil, err
	}
	cmd := procutil.CommandContext(ctx, exe, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		return nil, nil, err
	}
	go func() {
		err := cmd.Wait()
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_ = pw.Close()
	}()
	return cmd, pr, nil
}

func ReadLines(r io.Reader, fn func(line string)) error {
	sc := bufio.NewScanner(r)
	// yt-dlp --dump-single-json is one line and can exceed 1 MiB.
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	sc.Split(scanLinesCRLF)
	for sc.Scan() {
		fn(sc.Text())
	}
	if err := sc.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return fmt.Errorf("tool output line too long")
		}
		return fmt.Errorf("could not read tool output")
	}
	return nil
}

func scanLinesCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			line := data[:i]
			if i > 0 && data[i-1] == '\r' {
				line = data[:i-1]
			}
			return i + 1, line, nil
		}
		if data[i] == '\r' {
			if i+1 < len(data) {
				adv := i + 1
				if data[i+1] == '\n' {
					adv++
				}
				return adv, data[:i], nil
			}
			if !atEOF {
				return 0, nil, nil
			}
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func CollectTitle(output string) string {
	lines := strings.Split(output, "\n")
	var title string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "WARNING:") || strings.HasPrefix(line, "ERROR:") {
			continue
		}
		title = line
	}
	return strings.TrimSpace(title)
}

type lineBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (l *lineBuffer) WriteString(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.b.WriteString(s)
	l.b.WriteByte('\n')
}

func (l *lineBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}
