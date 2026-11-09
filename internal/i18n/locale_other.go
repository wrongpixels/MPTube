//go:build !windows

package i18n

import "os"

func SystemLocale() string {
	if v := os.Getenv("LC_ALL"); v != "" {
		return v
	}
	if v := os.Getenv("LC_MESSAGES"); v != "" {
		return v
	}
	return os.Getenv("LANG")
}
