package tools

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestAllowedDownloadHost(t *testing.T) {
	good := []string{"github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com", "www.gyan.dev"}
	for _, h := range good {
		if !AllowedDownloadHost(h) {
			t.Fatalf("should allow %s", h)
		}
	}
	if AllowedDownloadHost("evil.example") || AllowedDownloadHost("github.com.evil.com") {
		t.Fatal("should reject unexpected hosts")
	}
}

func TestPinnedHashesLookValid(t *testing.T) {
	for _, a := range Artifacts() {
		if len(a.SHA256) != 64 {
			t.Fatalf("%s hash length", a.Name)
		}
		if a.URL[:8] != "https://" {
			t.Fatalf("%s URL %s", a.Name, a.URL)
		}
		if !AllowedDownloadHost(mustHost(a.URL)) {
			t.Fatalf("artifact host not allowed: %s", a.URL)
		}
	}
}

func mustHost(raw string) string {
	// tiny helper without importing net/url again in a fancy way
	s := raw[len("https://"):]
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return s[:i]
		}
	}
	return s
}

func TestExtractNamedFile(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "in.zip")
	destDir := filepath.Join(dir, "out")
	if err := writeZip(zipPath, "nested/dir/deno.exe", []byte("hello-deno")); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(destDir, "deno.exe")
	if err := extractNamedFile(zipPath, destDir, "deno.exe", dest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello-deno" {
		t.Fatalf("got %q", data)
	}
}

func TestHashFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f.bin")
	payload := []byte("abc")
	if err := os.WriteFile(p, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	sum, err := hashFile(p)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(payload)
	if sum != hex.EncodeToString(h[:]) {
		t.Fatalf("sum=%s", sum)
	}
}

func writeZip(path, name string, body []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := zip.NewWriter(f)
	fw, err := w.Create(name)
	if err != nil {
		return err
	}
	if _, err := fw.Write(body); err != nil {
		return err
	}
	return w.Close()
}
