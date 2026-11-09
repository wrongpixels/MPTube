package metadata

import (
	"context"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	MaxThumbnailBytes  = 2 << 20
	ThumbnailTimeout   = 15 * time.Second
	maxPreviewAttempts = 12
	maxPreviewEdge     = 4096
)

func AllowedThumbnailHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "ytimg.com" || host == "img.youtube.com" {
		return true
	}
	return strings.HasSuffix(host, ".ytimg.com")
}

// CanonicalThumbnailURL accepts only https YouTube thumbnail CDN URLs.
func CanonicalThumbnailURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 2000 {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	if u.Scheme != "https" {
		return "", false
	}
	if u.User != nil {
		return "", false
	}
	if !AllowedThumbnailHost(u.Hostname()) {
		return "", false
	}
	u.Fragment = ""
	return u.String(), true
}

func FallbackThumbnailURL(videoID string) string {
	if !validVideoID(videoID) {
		return ""
	}
	return "https://i.ytimg.com/vi/" + videoID + "/hqdefault.jpg"
}

func validVideoID(id string) bool {
	if len(id) != 11 {
		return false
	}
	for _, r := range id {
		if !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func ThumbnailCandidates(infoThumb, videoID string) []string {
	return PreviewCandidates(Info{Thumbnail: infoThumb}, videoID)
}

// PreviewCandidates is the URL list the preview fetcher walks.
// Metadata URLs and the fixed i.ytimg.com JPEG stills are merged, then ordered
// by yt-dlp preference and pixel area. Hosts outside the YouTube image
// allowlist are dropped. Nothing is fetched.
func PreviewCandidates(info Info, videoID string) []string {
	var incoming []Thumb
	if info.Thumbnail != "" {
		incoming = append(incoming, Thumb{URL: info.Thumbnail})
	}
	incoming = append(incoming, info.Thumbs...)
	if validVideoID(videoID) {
		incoming = append(incoming,
			Thumb{URL: "https://i.ytimg.com/vi/" + videoID + "/maxresdefault.jpg", Width: 1280, Height: 720},
			Thumb{URL: "https://i.ytimg.com/vi/" + videoID + "/sddefault.jpg", Width: 640, Height: 480},
			Thumb{URL: "https://i.ytimg.com/vi/" + videoID + "/hqdefault.jpg", Width: 480, Height: 360},
			Thumb{URL: "https://i.ytimg.com/vi/" + videoID + "/mqdefault.jpg", Width: 320, Height: 180},
		)
	}
	byURL := map[string]Thumb{}
	var order []string
	for _, th := range incoming {
		canon, ok := CanonicalThumbnailURL(th.URL)
		if !ok {
			continue
		}
		th.URL = canon
		prev, seen := byURL[canon]
		if !seen {
			order = append(order, canon)
			byURL[canon] = th
			continue
		}
		byURL[canon] = richerThumb(prev, th)
	}
	ranked := make([]Thumb, 0, len(order))
	for _, u := range order {
		ranked = append(ranked, byURL[u])
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return betterThumb(ranked[i], ranked[j])
	})
	out := make([]string, len(ranked))
	for i, th := range ranked {
		out[i] = th.URL
	}
	return out
}

func richerThumb(a, b Thumb) Thumb {
	if b.HasPref && (!a.HasPref || b.Preference > a.Preference) {
		a.Preference = b.Preference
		a.HasPref = true
	}
	if int64(b.Width)*int64(b.Height) > int64(a.Width)*int64(a.Height) {
		a.Width = b.Width
		a.Height = b.Height
	}
	return a
}

func betterThumb(a, b Thumb) bool {
	if a.HasPref && b.HasPref && a.Preference != b.Preference {
		return a.Preference > b.Preference
	}
	ai := int64(a.Width) * int64(a.Height)
	bi := int64(b.Width) * int64(b.Height)
	if ai != bi {
		return ai > bi
	}
	if a.HasPref != b.HasPref {
		return a.HasPref
	}
	return false
}

// LoadPreview fetches candidates with client until one decodes as a still image.
// A download that is the wrong codec, an error page, or an oversized body is
// skipped. client may be nil, in which case NewThumbnailClient is used.
// Only CanonicalThumbnailURL targets are requested.
func LoadPreview(ctx context.Context, client *http.Client, candidates []string) (image.Image, string, error) {
	if client == nil {
		client = NewThumbnailClient()
	}
	var last error
	tried := 0
	seen := map[string]struct{}{}
	for _, raw := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		canon, ok := CanonicalThumbnailURL(raw)
		if !ok {
			continue
		}
		if _, dup := seen[canon]; dup {
			continue
		}
		seen[canon] = struct{}{}
		if tried >= maxPreviewAttempts {
			break
		}
		tried++
		data, err := getLimited(ctx, client, canon, MaxThumbnailBytes)
		if err != nil {
			last = err
			continue
		}
		img, _, err := DecodePreview(data)
		if err != nil {
			last = err
			continue
		}
		b := img.Bounds()
		if b.Dx() < 1 || b.Dy() < 1 || b.Dx() > maxPreviewEdge || b.Dy() > maxPreviewEdge {
			last = fmt.Errorf("thumbnail size is not usable")
			continue
		}
		return img, canon, nil
	}
	if last == nil {
		last = fmt.Errorf("no decodable thumbnail")
	}
	return nil, "", last
}

func NewThumbnailClient() *http.Client {
	return &http.Client{
		Timeout: ThumbnailTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many thumbnail redirects")
			}
			if req.URL.Scheme != "https" || !AllowedThumbnailHost(req.URL.Hostname()) {
				return fmt.Errorf("thumbnail redirected to a disallowed host")
			}
			return nil
		},
	}
}

func DownloadThumbnail(ctx context.Context, rawURL, dest string) error {
	canon, ok := CanonicalThumbnailURL(rawURL)
	if !ok {
		return fmt.Errorf("thumbnail URL is not an allowed YouTube image")
	}
	return downloadLimited(ctx, NewThumbnailClient(), canon, dest, MaxThumbnailBytes)
}

func downloadLimited(ctx context.Context, client *http.Client, rawURL, dest string, maxBytes int64) error {
	data, err := getLimited(ctx, client, rawURL, maxBytes)
	if err != nil {
		return err
	}
	tmp := dest + ".part"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
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

func getLimited(ctx context.Context, client *http.Client, rawURL string, maxBytes int64) ([]byte, error) {
	if client == nil {
		client = NewThumbnailClient()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "YouTubeMP3Saver/12.0")
	req.Header.Set("Accept", "image/jpeg, image/png, image/gif, image/webp, */*;q=0.1")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("thumbnail HTTP %s", resp.Status)
	}
	if resp.ContentLength > maxBytes {
		return nil, fmt.Errorf("thumbnail is too large")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("thumbnail is too large")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("thumbnail was empty")
	}
	return data, nil
}
