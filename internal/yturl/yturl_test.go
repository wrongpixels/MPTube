package yturl

import "testing"

func TestValidateAcceptsYouTube(t *testing.T) {
	ok := []string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"http://youtube.com/watch?v=dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ",
		"https://m.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://music.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://www.youtube.com/shorts/dQw4w9WgXcQ",
		"https://www.youtube.com/embed/dQw4w9WgXcQ",
		"https://www.youtube.com/live/dQw4w9WgXcQ",
		"https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PLxx", // playlist query kept; downloader uses --no-playlist
	}
	for _, u := range ok {
		got, err := Validate(u)
		if err != nil {
			t.Fatalf("%s: %v", u, err)
		}
		if got == "" {
			t.Fatalf("empty for %s", u)
		}
	}
}

func TestValidateRejects(t *testing.T) {
	bad := []string{
		"",
		"javascript:alert(1)",
		"file:///etc/passwd",
		"https://youtube.com.evil.com/watch?v=dQw4w9WgXcQ",
		"https://evil.com/?u=youtube.com",
		"https://user:pass@www.youtube.com/watch?v=dQw4w9WgXcQ",
		"https://vimeo.com/123",
		"https://youtu.be/",
		"https://www.youtube.com/",
		"--output /tmp/x",
		"https://www.youtube.com/watch?v=abc\n--proxy",
	}
	for _, u := range bad {
		if _, err := Validate(u); err == nil {
			t.Fatalf("expected reject: %q", u)
		}
	}
}

func TestVideoID(t *testing.T) {
	id := "dQw4w9WgXcQ"
	cases := []string{
		"https://www.youtube.com/watch?v=" + id,
		"https://youtu.be/" + id,
		"https://www.youtube.com/shorts/" + id,
		"https://www.youtube.com/embed/" + id,
		"https://www.youtube.com/live/" + id,
		"https://www.youtube.com/watch?v=" + id + "&list=PLxx",
	}
	for _, u := range cases {
		got, ok := VideoID(u)
		if !ok || got != id {
			t.Fatalf("%s -> %q %v", u, got, ok)
		}
	}
	if _, ok := VideoID("https://www.youtube.com/watch?v=short"); ok {
		t.Fatal("too short")
	}
	if _, ok := VideoID("https://vimeo.com/dQw4w9WgXcQ"); ok {
		t.Fatal("non-youtube")
	}
}

func TestReadyForFetch(t *testing.T) {
	if _, ok := ReadyForFetch("https://www.youtube.com/watch?v=dQw"); ok {
		t.Fatal("incomplete id should not be ready")
	}
	if _, ok := ReadyForFetch("https://www.youtube.com/watch?v="); ok {
		t.Fatal("empty id")
	}
	if _, ok := ReadyForFetch("https://youtube.com/watch?v=dQw4w9WgXcQ"); !ok {
		t.Fatal("complete watch URL should be ready")
	}
	if _, ok := ReadyForFetch("  https://youtu.be/dQw4w9WgXcQ  "); !ok {
		t.Fatal("trimmed complete short URL should be ready")
	}
	if _, ok := ReadyForFetch("not a url"); ok {
		t.Fatal("garbage")
	}
}

func TestCanonicalWatchURL(t *testing.T) {
	const id = "dQw4w9WgXcQ"
	want := "https://www.youtube.com/watch?v=" + id
	cases := []string{
		"https://www.youtube.com/watch?v=" + id,
		"http://youtube.com/watch?v=" + id,
		"https://youtu.be/" + id,
		"https://m.youtube.com/watch?v=" + id,
		"https://music.youtube.com/watch?v=" + id,
		"https://www.youtube.com/shorts/" + id,
		"https://www.youtube.com/embed/" + id,
		"https://www.youtube.com/live/" + id,
		"https://www.youtube-nocookie.com/embed/" + id,
		"www.youtube.com/watch?v=" + id,
		"youtube.com/watch?v=" + id,
		"youtu.be/" + id,
		"m.youtube.com/watch?v=" + id,
		"music.youtube.com/watch?v=" + id,
		"  https://youtu.be/" + id + "  ",
		"//www.youtube.com/watch?v=" + id,
	}
	for _, u := range cases {
		got, ok := ReadyForFetch(u)
		if !ok || got != want {
			t.Fatalf("%s -> %q ok=%v", u, got, ok)
		}
		if Analyze(u).Kind != KindVideo {
			t.Fatalf("kind for %s: %v", u, Analyze(u).Kind)
		}
	}
}

func TestStripExtraParams(t *testing.T) {
	const id = "dQw4w9WgXcQ"
	want := "https://www.youtube.com/watch?v=" + id
	cases := []string{
		"https://www.youtube.com/watch?v=" + id + "&list=PLxx&index=3",
		"https://www.youtube.com/watch?v=" + id + "&si=abcdef&t=30",
		"https://www.youtube.com/watch?v=" + id + "&utm_source=share&utm_medium=ios&feature=share",
		"https://youtu.be/" + id + "?si=xxx&t=43",
		"https://www.youtube.com/watch?v=" + id + "&ab_channel=Rick&pp=ygU",
		"https://www.youtube.com/embed/" + id + "?start=15",
		"https://music.youtube.com/watch?v=" + id + "&list=RDAMVM",
		"https://www.youtube.com/watch?time_continue=12&v=" + id + "&feature=emb_title",
	}
	for _, u := range cases {
		got, ok := ReadyForFetch(u)
		if !ok || got != want {
			t.Fatalf("%s -> %q ok=%v", u, got, ok)
		}
	}
}

func TestPlaylistOnlyIsNotAVideo(t *testing.T) {
	cases := []string{
		"https://www.youtube.com/playlist?list=PLxxxxxxxx",
		"https://www.youtube.com/watch?list=PLxxxxxxxx",
		"https://youtube.com/playlist?list=PLxx",
		"www.youtube.com/playlist?list=PLxx",
		"https://www.youtube.com/channel/UCxxxxxxxx",
		"https://www.youtube.com/@someone",
	}
	for _, u := range cases {
		if _, ok := ReadyForFetch(u); ok {
			t.Fatalf("should not fetch: %s", u)
		}
		if got := Analyze(u).Kind; got != KindNotVideo {
			t.Fatalf("kind %v for %s", got, u)
		}
		if _, err := Validate(u); err == nil {
			t.Fatalf("validate should reject: %s", u)
		}
	}
}

func TestRejectSpoofedAndForeignHosts(t *testing.T) {
	bad := []string{
		"https://youtube.com.evil.com/watch?v=dQw4w9WgXcQ",
		"https://www.youtube.com.evil.tld/watch?v=dQw4w9WgXcQ",
		"https://evil-youtube.com/watch?v=dQw4w9WgXcQ",
		"https://notyoutube.com/watch?v=dQw4w9WgXcQ",
		"https://evil.com/watch?v=dQw4w9WgXcQ",
		"https://example.com/?v=dQw4w9WgXcQ&youtube.com=1",
		"https://evil.com/?u=https://youtube.com.evil.com/watch?v=dQw4w9WgXcQ",
	}
	for _, u := range bad {
		if _, ok := ReadyForFetch(u); ok {
			t.Fatalf("accepted spoof/foreign: %s", u)
		}
		if Analyze(u).Kind == KindVideo {
			t.Fatalf("Analyze video for %s", u)
		}
	}
}

func TestExtractFromSurroundingText(t *testing.T) {
	const want = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	cases := []string{
		"Check this: https://youtu.be/dQw4w9WgXcQ?si=abc thanks",
		"mira esto\nhttps://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PLxx\nfin",
		"watch youtu.be/dQw4w9WgXcQ now",
		"link: www.youtube.com/watch?v=dQw4w9WgXcQ.",
	}
	for _, u := range cases {
		got, ok := ReadyForFetch(u)
		if !ok || got != want {
			t.Fatalf("%s -> %q ok=%v", u, got, ok)
		}
	}
}

func TestIncompleteWatchIsNotReady(t *testing.T) {
	cases := []string{
		"https://www.youtube.com/watch?v=dQw",
		"https://www.youtube.com/watch?v=",
		"https://youtu.be/dQw4w9",
		"www.youtube.com/watch?v=dQw4w9WgXc",
	}
	for _, u := range cases {
		if _, ok := ReadyForFetch(u); ok {
			t.Fatalf("incomplete ready: %s", u)
		}
		if Analyze(u).Kind != KindIncomplete {
			t.Fatalf("want incomplete for %s got %v", u, Analyze(u).Kind)
		}
	}
}
