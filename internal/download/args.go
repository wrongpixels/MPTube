package download

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Tools holds resolved absolute paths to helper executables.
type Tools struct {
	Ytdlp     string
	FFmpegDir string
	Deno      string
}

func TitleArgs(t Tools, videoURL string) []string {
	args := []string{
		"--ignore-config",
		"--no-playlist",
		"--no-warnings",
		"--skip-download",
		"--print", "title",
	}
	args = append(args, jsRuntimeArgs(t)...)
	args = append(args, "--", videoURL)
	return args
}

func MetadataArgs(t Tools, videoURL string) []string {
	args := []string{
		"--ignore-config",
		"--no-playlist",
		"--no-warnings",
		"--dump-single-json",
	}
	args = append(args, jsRuntimeArgs(t)...)
	args = append(args, "--", videoURL)
	return args
}

func DownloadArgs(t Tools, videoURL, outputTemplate string) []string {
	return downloadArgs(t, outputTemplate, videoURL, "")
}

func DownloadArgsFromInfo(t Tools, infoJSON, outputTemplate string) []string {
	return downloadArgs(t, outputTemplate, "", infoJSON)
}

func downloadArgs(t Tools, outputTemplate, videoURL, infoJSON string) []string {
	args := []string{
		"--ignore-config",
		"--no-playlist",
		"--no-mtime",
		"--newline",
		"--progress",
		"--color", "never",
		"--progress-template", DownloadProgressTemplate,
		"--progress-template", PostprocessProgressTemplate,
		"--no-write-comments",
		"-f", "bestaudio/best",
		"-x",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"-o", outputTemplate,
	}
	if t.FFmpegDir != "" {
		args = append(args, "--ffmpeg-location", t.FFmpegDir)
	}
	if strings.TrimSpace(infoJSON) != "" {
		args = append(args, "--load-info-json", infoJSON)
	}
	args = append(args, jsRuntimeArgs(t)...)
	if strings.TrimSpace(videoURL) != "" {
		args = append(args, "--", videoURL)
	}
	return args
}

func jsRuntimeArgs(t Tools) []string {
	if strings.TrimSpace(t.Deno) == "" {
		return nil
	}
	return []string{"--js-runtimes", "deno:" + t.Deno}
}

func OutputTemplate(tempDir, sanitizedBase string) string {
	return filepath.Join(tempDir, sanitizedBase+".%(ext)s")
}

func ValidateArgv(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("empty command")
	}
	joined := strings.Join(argv, "\x00")
	if strings.Contains(joined, "\x00\x00") {
		return fmt.Errorf("empty argument")
	}
	return nil
}
