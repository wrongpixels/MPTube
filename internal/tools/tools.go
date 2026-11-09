package tools

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	YtdlpVersion  = "2026.10.26"
	FFmpegVersion = "9.0.2"
	DenoVersion   = "2.9.8"

	YtdlpURL    = "https://github.com/yt-dlp/yt-dlp/releases/download/2026.10.26/yt-dlp.exe"
	YtdlpSHA256 = "f5389d3698ecde4ef06f4e6b1ce0294fe9ac1396228c8990b8d3416d6d177a7d"

	FFmpegURL    = "https://github.com/GyanD/codexffmpeg/releases/download/9.0.2/ffmpeg-9.0.2-essentials_build.zip"
	FFmpegSHA256 = "60f467265b1e312373dbcd92200c2618a74850f98d3d078e94296bb3fa2047ba"

	DenoURL    = "https://github.com/denoland/deno/releases/download/v2.9.8/deno-x86_64-pc-windows-msvc.zip"
	DenoSHA256 = "cba84b26e0df5ae7f74fc66b86a95b866872a86292ebd84fa1e959927edd93bf"
)

type ProgressFunc func(status string, percent int)

type Artifact struct {
	Name     string
	URL      string
	SHA256   string
	Kind     string // "exe" or "zip"
	DestName string
	FindFile string // filename to extract from a zip
}

func Artifacts() []Artifact {
	return []Artifact{
		{Name: "yt-dlp " + YtdlpVersion, URL: YtdlpURL, SHA256: YtdlpSHA256, Kind: "exe", DestName: "yt-dlp.exe"},
		{Name: "FFmpeg " + FFmpegVersion, URL: FFmpegURL, SHA256: FFmpegSHA256, Kind: "zip", DestName: "ffmpeg.exe", FindFile: "ffmpeg.exe"},
		{Name: "Deno " + DenoVersion, URL: DenoURL, SHA256: DenoSHA256, Kind: "zip", DestName: "deno.exe", FindFile: "deno.exe"},
	}
}

type Paths struct {
	Dir       string
	Ytdlp     string
	FFmpeg    string
	FFmpegDir string
	Deno      string
}

func Resolve(dir string) Paths {
	return Paths{
		Dir:       dir,
		Ytdlp:     filepath.Join(dir, "yt-dlp.exe"),
		FFmpeg:    filepath.Join(dir, "ffmpeg.exe"),
		FFmpegDir: dir,
		Deno:      filepath.Join(dir, "deno.exe"),
	}
}

func (p Paths) Ready() bool {
	return fileExists(p.Ytdlp) && fileExists(p.FFmpeg) && fileExists(p.Deno) && versionsMatch(p.Dir)
}

type versionsFile struct {
	Ytdlp  string `json:"yt-dlp"`
	FFmpeg string `json:"ffmpeg"`
	Deno   string `json:"deno"`
}

func versionsPath(dir string) string {
	return filepath.Join(dir, "versions.json")
}

func versionsMatch(dir string) bool {
	data, err := os.ReadFile(versionsPath(dir))
	if err != nil {
		return false
	}
	var v versionsFile
	if json.Unmarshal(data, &v) != nil {
		return false
	}
	return v.Ytdlp == YtdlpVersion && v.FFmpeg == FFmpegVersion && v.Deno == DenoVersion
}

func writeVersions(dir string) error {
	v := versionsFile{Ytdlp: YtdlpVersion, FFmpeg: FFmpegVersion, Deno: DenoVersion}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(versionsPath(dir), data, 0o644)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func AllowedDownloadHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	switch host {
	case "github.com",
		"objects.githubusercontent.com",
		"release-assets.githubusercontent.com",
		"github-releases.githubusercontent.com",
		"objects-origin.githubusercontent.com",
		"www.gyan.dev",
		"gyan.dev":
		return true
	default:
		return strings.HasSuffix(host, ".githubusercontent.com")
	}
}

func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 8 {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Scheme != "https" {
				return fmt.Errorf("refusing non-HTTPS redirect to %s", req.URL)
			}
			if !AllowedDownloadHost(req.URL.Hostname()) {
				return fmt.Errorf("unexpected redirect host %s", req.URL.Hostname())
			}
			return nil
		},
	}
}

func Ensure(dir string, progress ProgressFunc) (Paths, error) {
	p := Resolve(dir)
	if p.Ready() {
		if progress != nil {
			progress("Helper tools are ready.", 100)
		}
		return p, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Paths{}, err
	}
	client := NewHTTPClient()
	arts := Artifacts()
	for i, a := range arts {
		dest := filepath.Join(dir, a.DestName)
		if fileExists(dest) && versionsMatch(dir) {
			continue
		}
		base := (i * 100) / len(arts)
		span := 100 / len(arts)
		if progress != nil {
			progress("Downloading "+a.Name+" from the official source…", base)
		}
		if err := fetchArtifact(client, dir, a, func(pct int) {
			if progress != nil {
				progress("Downloading "+a.Name+" from the official source…", base+(pct*span)/100)
			}
		}); err != nil {
			return Paths{}, err
		}
	}
	if err := writeVersions(dir); err != nil {
		return Paths{}, err
	}
	p = Resolve(dir)
	if !fileExists(p.Ytdlp) || !fileExists(p.FFmpeg) || !fileExists(p.Deno) {
		return Paths{}, fmt.Errorf("tools downloaded but some files are still missing")
	}
	if progress != nil {
		progress("Helper tools are ready.", 100)
	}
	return p, nil
}

func fetchArtifact(client *http.Client, dir string, a Artifact, progress func(int)) error {
	u, err := url.Parse(a.URL)
	if err != nil || u.Scheme != "https" {
		return fmt.Errorf("invalid tool URL")
	}
	if !AllowedDownloadHost(u.Hostname()) {
		return fmt.Errorf("refusing download host %s", u.Hostname())
	}
	tmp := filepath.Join(dir, a.DestName+".download")
	_ = os.Remove(tmp)
	if err := downloadFile(client, a.URL, tmp, progress); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s download failed: %w", a.Name, err)
	}
	sum, err := hashFile(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if !strings.EqualFold(sum, a.SHA256) {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s checksum mismatch (got %s, expected %s). The file was discarded", a.Name, sum, a.SHA256)
	}
	dest := filepath.Join(dir, a.DestName)
	if a.Kind == "exe" {
		_ = os.Remove(dest)
		if err := os.Rename(tmp, dest); err != nil {
			_ = os.Remove(tmp)
			return err
		}
		return nil
	}
	if err := extractNamedFile(tmp, dir, a.FindFile, dest); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s extract failed: %w", a.Name, err)
	}
	_ = os.Remove(tmp)
	return nil
}

func downloadFile(client *http.Client, rawURL, dest string, progress func(int)) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "YouTubeMP3Saver/1.0 (personal downloader; +https://github.com/yt-dlp/yt-dlp)")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	var written int64
	buf := make([]byte, 32*1024)
	total := resp.ContentLength
	last := -1
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if progress != nil && total > 0 {
				pct := int(written * 100 / total)
				if pct != last {
					progress(pct)
					last = pct
				}
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	return nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func extractNamedFile(zipPath, destDir, name, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	var found *zip.File
	for i := range r.File {
		base := filepath.Base(r.File[i].Name)
		if strings.EqualFold(base, name) && !strings.HasSuffix(r.File[i].Name, "/") {
			found = r.File[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("%s not found inside the archive", name)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	rc, err := found.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	tmp := dest + ".extract"
	_ = os.Remove(tmp)
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(dest)
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func FirstRunExplanation() string {
	return "The first time you save a file, this app downloads official helper tools " +
		"(yt-dlp " + YtdlpVersion + ", FFmpeg " + FFmpegVersion + " essentials, and Deno " + DenoVersion + ") " +
		"over HTTPS, checks their SHA-256 checksums, and stores them in this app's local cache. " +
		"Together they are about 170 MB. No extra Python, Java, or administrator install is required."
}
