//go:build windows

package i18n

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func SystemLocale() string {
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	proc := kernel.NewProc("GetUserDefaultLocaleName")
	if proc.Find() == nil {
		buf := make([]uint16, 85)
		r, _, _ := proc.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if r > 0 {
			return syscall.UTF16ToString(buf)
		}
	}
	if v := os.Getenv("LANG"); v != "" {
		return v
	}
	return "en"
}
