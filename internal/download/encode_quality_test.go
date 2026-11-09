package download

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ffmpegMP3QualityArgs matches yt-dlp --audio-quality 0 (best LAME VBR, not lossless).
func ffmpegMP3QualityArgs(src, dest string) []string {
	return []string{"-y", "-i", src, "-codec:a", "libmp3lame", "-q:a", "0", dest}
}

func TestFFmpegHighestSensibleMP3Encode(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed on this machine")
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe not installed on this machine")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "tone.wav")
	dest := filepath.Join(dir, "tone.mp3")
	gen := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=1", "-c:a", "pcm_s16le", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generate wav: %v\n%s", err, out)
	}
	enc := exec.Command("ffmpeg", ffmpegMP3QualityArgs(src, dest)...)
	if out, err := enc.CombinedOutput(); err != nil {
		t.Fatalf("encode mp3: %v\n%s", err, out)
	}
	st, err := os.Stat(dest)
	if err != nil || st.Size() < 100 {
		t.Fatalf("mp3 missing or tiny: %v", err)
	}
	probe := exec.Command(ffprobe, "-v", "error", "-show_entries", "stream=codec_name", "-of", "default=nw=1", dest)
	out, err := probe.CombinedOutput()
	if err != nil {
		t.Fatalf("ffprobe: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "codec_name=mp3") {
		t.Fatalf("expected mp3 codec, got %s", out)
	}
}
