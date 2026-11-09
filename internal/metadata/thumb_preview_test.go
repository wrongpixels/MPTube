package metadata

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"youtubemp3/internal/uidim"
)

func TestDecodePreviewFormats(t *testing.T) {
	cases := []struct {
		name   string
		file   string
		format string
		w, h   int
		inline []byte
	}{
		{name: "square webp", file: "testdata/square.webp", format: "webp", w: 64, h: 64},
		{name: "portrait webp", file: "testdata/portrait.webp", format: "webp", w: 30, h: 80},
		{name: "wide webp", file: "testdata/wide.webp", format: "webp", w: 160, h: 90},
		{name: "square jpeg", file: "testdata/square.jpg", format: "jpeg", w: 64, h: 64},
		{name: "png", file: "testdata/tiny.png", format: "png", w: 16, h: 16},
		{name: "gif", inline: gif1x1, format: "gif", w: 1, h: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := tc.inline
			if tc.file != "" {
				var err error
				data, err = os.ReadFile(tc.file)
				if err != nil {
					t.Fatal(err)
				}
			}
			img, format, err := DecodePreview(data)
			if err != nil {
				t.Fatal(err)
			}
			if format != tc.format {
				t.Fatalf("format %q", format)
			}
			b := img.Bounds()
			if b.Dx() != tc.w || b.Dy() != tc.h {
				t.Fatalf("bounds %dx%d", b.Dx(), b.Dy())
			}
		})
	}
	if _, _, err := DecodePreview([]byte("RIFF\x00\x00\x00\x00WEBP")); err == nil {
		t.Fatal("truncated webp should not decode")
	}
	if _, _, err := DecodePreview([]byte("<html>not an image</html>")); err == nil {
		t.Fatal("html should not decode")
	}
}

func TestPreviewCandidatesMusicShape(t *testing.T) {
	info := Info{
		Thumbnail: "https://i.ytimg.com/vi_webp/QsXnlRXZsVk/maxresdefault.webp",
		Thumbs: []Thumb{
			{URL: "https://i.ytimg.com/vi_webp/QsXnlRXZsVk/maxresdefault.webp", Width: 1280, Height: 720, Preference: 0, HasPref: true},
			{URL: "https://i.ytimg.com/vi/QsXnlRXZsVk/hqdefault.jpg", Width: 480, Height: 360, Preference: -7, HasPref: true},
			{URL: "https://evil.example/cover.jpg", Width: 9999, Height: 9999, Preference: 100, HasPref: true},
			{URL: "http://i.ytimg.com/vi/QsXnlRXZsVk/sddefault.jpg", Preference: 50, HasPref: true},
			{URL: "https://user:pass@i.ytimg.com/vi/QsXnlRXZsVk/mqdefault.jpg"},
		},
	}
	got := PreviewCandidates(info, "QsXnlRXZsVk")
	want := []string{
		"https://i.ytimg.com/vi_webp/QsXnlRXZsVk/maxresdefault.webp",
		"https://i.ytimg.com/vi/QsXnlRXZsVk/maxresdefault.jpg",
		"https://i.ytimg.com/vi/QsXnlRXZsVk/sddefault.jpg",
		"https://i.ytimg.com/vi/QsXnlRXZsVk/hqdefault.jpg",
		"https://i.ytimg.com/vi/QsXnlRXZsVk/mqdefault.jpg",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got %#v", got)
	}
}

func TestLoadPreviewUsesProductionCandidateOrder(t *testing.T) {
	webp := mustRead(t, "testdata/square.webp")
	jpeg := mustRead(t, "testdata/square.jpg")
	png := mustRead(t, "testdata/tiny.png")
	id := "QsXnlRXZsVk"
	webpURL := "https://i.ytimg.com/vi_webp/" + id + "/maxresdefault.webp"
	jpegURL := "https://i.ytimg.com/vi/" + id + "/maxresdefault.jpg"
	hqURL := "https://i.ytimg.com/vi/" + id + "/hqdefault.jpg"
	info := Info{
		Thumbnail: webpURL,
		Thumbs: []Thumb{
			{URL: webpURL, Width: 1280, Height: 720, Preference: 0, HasPref: true},
			{URL: hqURL, Width: 480, Height: 360, Preference: -7, HasPref: true},
			{URL: "https://evil.example/cover.jpg", Width: 9000, Height: 9000, Preference: 100, HasPref: true},
		},
	}
	cands := PreviewCandidates(info, id)

	t.Run("webp decodes and later urls are not fetched", func(t *testing.T) {
		rt := &mapTransport{files: map[string][]byte{webpURL: webp}}
		img, used, err := LoadPreview(context.Background(), rt.client(), cands)
		if err != nil {
			t.Fatal(err)
		}
		if used != webpURL {
			t.Fatalf("used %s", used)
		}
		if img.Bounds().Dx() != 64 || img.Bounds().Dy() != 64 {
			t.Fatalf("bounds %v", img.Bounds())
		}
		if len(rt.seen) != 1 || rt.seen[0] != webpURL {
			t.Fatalf("seen %v", rt.seen)
		}
	})

	t.Run("undecodable webp falls through to jpeg", func(t *testing.T) {
		rt := &mapTransport{files: map[string][]byte{
			webpURL: []byte("RIFF\x00\x00\x00\x00WEBPnot-an-image"),
			jpegURL: jpeg,
		}, types: map[string]string{
			webpURL: "image/webp",
			jpegURL: "image/jpeg",
		}}
		img, used, err := LoadPreview(context.Background(), rt.client(), cands)
		if err != nil {
			t.Fatal(err)
		}
		if used != jpegURL {
			t.Fatalf("used %s seen %v", used, rt.seen)
		}
		if img.Bounds().Dx() != 64 || img.Bounds().Dy() != 64 {
			t.Fatalf("bounds %v", img.Bounds())
		}
		for _, u := range rt.seen {
			if strings.Contains(u, "evil.example") {
				t.Fatalf("fetched disallowed host: %v", rt.seen)
			}
		}
	})

	t.Run("content type is not trusted over bytes", func(t *testing.T) {
		rt := &mapTransport{files: map[string][]byte{hqURL: png}, types: map[string]string{hqURL: "image/webp"}}
		only := []string{hqURL}
		img, _, err := LoadPreview(context.Background(), rt.client(), only)
		if err != nil {
			t.Fatal(err)
		}
		if img.Bounds().Dx() != 16 || img.Bounds().Dy() != 16 {
			t.Fatalf("png bytes labeled image/webp decoded as %v", img.Bounds())
		}
	})

	t.Run("oversized content-length skips to the next still", func(t *testing.T) {
		rt := &mapTransport{
			files: map[string][]byte{webpURL: webp, jpegURL: jpeg},
			clen:  map[string]int64{webpURL: MaxThumbnailBytes + 10},
		}
		_, used, err := LoadPreview(context.Background(), rt.client(), cands)
		if err != nil {
			t.Fatal(err)
		}
		if used != jpegURL {
			t.Fatalf("used %s", used)
		}
	})

	t.Run("disallowed urls are not requested", func(t *testing.T) {
		rt := &mapTransport{}
		_, _, err := LoadPreview(context.Background(), rt.client(), []string{
			"https://evil.example/a.jpg",
			"http://i.ytimg.com/vi/" + id + "/hqdefault.jpg",
			"https://user:pass@i.ytimg.com/vi/" + id + "/hqdefault.jpg",
		})
		if err == nil {
			t.Fatal("expected failure")
		}
		if len(rt.seen) != 0 {
			t.Fatalf("requests %v", rt.seen)
		}
	})

	x, y, w, h := uidim.Contain(uidim.ThumbWidth, uidim.ThumbHeight, 64, 64)
	if w != h || w != 90 || x != 35 || y != 0 {
		t.Fatalf("square contain in the 160x90 slot: %d,%d %dx%d", x, y, w, h)
	}
}

func TestThumbnailRedirectStaysOnAllowedHosts(t *testing.T) {
	c := NewThumbnailClient()
	req, err := http.NewRequest(http.MethodGet, "https://evil.example/x.jpg", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.CheckRedirect(req, nil); err == nil {
		t.Fatal("redirect to an unknown host should fail")
	}
	okReq, err := http.NewRequest(http.MethodGet, "https://i.ytimg.com/vi/QsXnlRXZsVk/hqdefault.jpg", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.CheckRedirect(okReq, []*http.Request{req}); err != nil {
		t.Fatal(err)
	}
}

type served struct {
	body  []byte
	ctype string
	clen  int64
	code  int
}

type mapTransport struct {
	files map[string][]byte
	types map[string]string
	clen  map[string]int64
	seen  []string
}

func (m *mapTransport) client() *http.Client {
	return &http.Client{Transport: m}
}

func (m *mapTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.seen = append(m.seen, req.URL.String())
	body, ok := m.files[req.URL.String()]
	if !ok {
		return textResponse(req, http.StatusNotFound, nil), nil
	}
	code := http.StatusOK
	clen := int64(len(body))
	if m.clen != nil {
		if n, has := m.clen[req.URL.String()]; has {
			clen = n
		}
	}
	ctype := "application/octet-stream"
	if m.types != nil && m.types[req.URL.String()] != "" {
		ctype = m.types[req.URL.String()]
	}
	resp := textResponse(req, code, body)
	resp.Header.Set("Content-Type", ctype)
	resp.ContentLength = clen
	return resp, nil
}

func textResponse(req *http.Request, code int, body []byte) *http.Response {
	if body == nil {
		body = []byte{}
	}
	return &http.Response{
		StatusCode:    code,
		Status:        fmt.Sprintf("%d", code),
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
	}
}

func mustRead(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// 1x1 GIF. Synthetic, not a thumbnail.
var gif1x1 = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00,
	0xff, 0x00, 0x00, 0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02,
	0x44, 0x01, 0x00, 0x3b,
}
