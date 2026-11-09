package metadata

import (
	"strings"
	"testing"
)

func TestParseOutputTitleDurationThumbnail(t *testing.T) {
	out := "WARNING: something\n" +
		`{"title":"A Cool Song","duration":125,"thumbnail":"https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg"}` + "\n"
	info, err := ParseOutput(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "A Cool Song" {
		t.Fatalf("title=%q", info.Title)
	}
	if !info.HasDur || info.Duration != 125 {
		t.Fatalf("duration=%v %v", info.Duration, info.HasDur)
	}
	if !strings.Contains(info.Thumbnail, "ytimg.com") {
		t.Fatalf("thumb=%q", info.Thumbnail)
	}
	if FormatDuration(info.Duration) != "2:05" {
		t.Fatalf("fmt=%q", FormatDuration(info.Duration))
	}
}

func TestParseOutputNullDurationAndEmptyTitle(t *testing.T) {
	info, err := ParseOutput(`{"title":"","duration":null,"thumbnail":""}`)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "youtube-audio" {
		t.Fatalf("title=%q", info.Title)
	}
	if info.HasDur {
		t.Fatal("expected no duration")
	}
}

func TestFormatDuration(t *testing.T) {
	if FormatDuration(3661) != "1:01:01" {
		t.Fatalf("got %q", FormatDuration(3661))
	}
	if FormatDuration(5) != "0:05" {
		t.Fatalf("got %q", FormatDuration(5))
	}
	if FormatDuration(-1) != "" {
		t.Fatal("negative")
	}
}

func TestParseIgnoresBrokenJSONLines(t *testing.T) {
	_, err := ParseOutput("not json\n{nope}\n")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseDumpKeepsFullJSONAndIgnoresExtraFields(t *testing.T) {
	out := "WARNING: noisy\n{\n  \"title\": \"Cached Song\",\n  \"duration\": 12,\n  \"thumbnail\": \"https://i.ytimg.com/vi/dQw4w9WgXcQ/mqdefault.jpg\",\n  \"formats\": [{\"url\": \"https://example.invalid/a?expire=1999999999\"}]\n}\n"
	raw, info, err := ParseDump(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Cached Song" || !info.HasDur {
		t.Fatalf("%+v", info)
	}
	if !jsonContains(raw, "formats") || !jsonContains(raw, "expire") {
		t.Fatalf("full JSON not preserved: %s", raw)
	}
}

func TestParseDumpIgnoresTrailingBraceOutsideTheDumpLine(t *testing.T) {
	out := "[youtube] Extracting URL: https://www.youtube.com/watch?v=dQw4w9WgXcQ\n" +
		`{"title":"Cached Song","duration":12,"formats":[{"url":"https://example.invalid/a?expire=1999999999"}]}` + "\n" +
		"ERROR: helper note }\n"
	raw, info, err := ParseDump(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "Cached Song" || len(raw) == 0 || !jsonContains(raw, "formats") {
		t.Fatalf("raw dropped: info=%+v raw=%s", info, raw)
	}
	if jsonContains(raw, "helper note") {
		t.Fatalf("stderr included in cache JSON: %s", raw)
	}
}

func TestParseThumbnailsArrayKeepsUntrustedURLs(t *testing.T) {
	raw := `{
	  "title": "Colour Your Night",
	  "duration": 212,
	  "thumbnail": "https://i.ytimg.com/vi_webp/QsXnlRXZsVk/maxresdefault.webp",
	  "thumbnails": [
	    {"url": "https://i.ytimg.com/vi_webp/QsXnlRXZsVk/maxresdefault.webp", "width": 1280, "height": 720, "preference": 0},
	    {"url": "https://i.ytimg.com/vi/QsXnlRXZsVk/hqdefault.jpg", "width": 480, "height": 360, "preference": -7},
	    {"url": "https://evil.example/cover.jpg", "width": 9999, "height": 9999, "preference": 100}
	  ]
	}`
	info, err := ParsePrintJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if info.Thumbnail == "" || len(info.Thumbs) != 3 {
		t.Fatalf("%+v", info)
	}
	if !info.Thumbs[0].HasPref || info.Thumbs[0].Preference != 0 || info.Thumbs[0].Width != 1280 {
		t.Fatalf("first thumb %+v", info.Thumbs[0])
	}
	if info.Thumbs[2].URL != "https://evil.example/cover.jpg" {
		t.Fatalf("raw list must keep the untrusted url for the allowlist to reject: %+v", info.Thumbs[2])
	}
}

func jsonContains(raw []byte, s string) bool { return strings.Contains(string(raw), s) }
