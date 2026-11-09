package main

import (
	"fmt"
	"os"
	"path/filepath"

	"youtubemp3/internal/appicon"
)

func main() {
	out := "cmd/youtubemp3/app.ico"
	if len(os.Args) > 1 {
		out = os.Args[1]
	}
	data, err := appicon.ICO()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d bytes)\n", out, len(data))
}
