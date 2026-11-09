//go:build !windows

package clip

// ReadUnicodeText is not used on this OS; the Windows UI calls the Windows build.
func ReadUnicodeText() (string, bool, error) {
	return "", false, nil
}
