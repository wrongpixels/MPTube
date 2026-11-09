package diaglog

import (
	"strings"
	"testing"
)

func TestSanitizeStripsURLPathAndSecrets(t *testing.T) {
	in := "Downloading https://www.youtube.com/watch?v=dQw4w9WgXcQ cookie=abc PATH=C:\\Users\\Me\\secret.mp3 token=xyz"
	got := sanitize(in)
	for _, leak := range []string{"youtube.com", "dQw4w9WgXcQ", `Users\Me`, "abc", "xyz"} {
		if strings.Contains(got, leak) {
			t.Fatalf("leaked %q in %q", leak, got)
		}
	}
	if !strings.Contains(got, "URL") || !strings.Contains(got, "REDACTED") {
		t.Fatalf("missing redaction marks: %q", got)
	}
}

func TestProgressPrefixKeepsNumericFields(t *testing.T) {
	p := progressPrefix("YTMP3\t  15.9%\t261120\t1638400\tNA\t465259.6\t2")
	if !strings.Contains(p, "15.9") || !strings.Contains(p, "261120") || !strings.Contains(p, "465259") {
		t.Fatalf("prefix=%q", p)
	}
}

func TestStampTMSInsertsAfterEventName(t *testing.T) {
	got := stampTMS("event=progress prefix=YTMP3,18.4% det=true\n")
	if !strings.HasPrefix(got, "event=progress t_ms=") || !strings.Contains(got, " prefix=YTMP3") {
		t.Fatalf("%q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("missing newline: %q", got)
	}
}
