//go:build windows

package clip

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const cfUnicodeText = 13 // CF_UNICODETEXT

var (
	modUser32                      = windows.NewLazySystemDLL("user32.dll")
	modKernel32                    = windows.NewLazySystemDLL("kernel32.dll")
	procOpenClipboard              = modUser32.NewProc("OpenClipboard")
	procCloseClipboard             = modUser32.NewProc("CloseClipboard")
	procIsClipboardFormatAvailable = modUser32.NewProc("IsClipboardFormatAvailable")
	procGetClipboardData           = modUser32.NewProc("GetClipboardData")
	procGlobalLock                 = modKernel32.NewProc("GlobalLock")
	procGlobalUnlock               = modKernel32.NewProc("GlobalUnlock")
)

// ReadUnicodeText reads CF_UNICODETEXT. It must only be called from an explicit Paste click.
// ok is false when the clipboard has no Unicode text (empty or another format).
func ReadUnicodeText() (text string, ok bool, err error) {
	r, _, callErr := procOpenClipboard.Call(0)
	if r == 0 {
		return "", false, callErr
	}
	defer procCloseClipboard.Call()

	avail, _, _ := procIsClipboardFormatAvailable.Call(cfUnicodeText)
	if avail == 0 {
		return "", false, nil
	}
	h, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return "", false, nil
	}
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return "", false, nil
	}
	defer procGlobalUnlock.Call(h)
	return utf16FromUintptr(ptr), true, nil
}

func utf16FromUintptr(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	p := *(**uint16)(unsafe.Pointer(&ptr))
	return windows.UTF16PtrToString(p)
}
