//go:build windows

package reveal

import (
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"

	"youtubemp3/internal/procutil"
)

// OpenAndSelect opens Explorer with path selected. It prefers the shell API
// and falls back to explorer.exe /select, without a console window.
func OpenAndSelect(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := CheckFile(abs); err != nil {
		return err
	}
	if err := shOpenAndSelect(abs); err == nil {
		return nil
	}
	return explorerSelect(abs)
}

func shOpenAndSelect(abs string) error {
	_ = windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED)
	shell32 := windows.NewLazySystemDLL("shell32.dll")
	parse := shell32.NewProc("SHParseDisplayName")
	open := shell32.NewProc("SHOpenFolderAndSelectItems")
	free := shell32.NewProc("ILFree")
	p, err := windows.UTF16PtrFromString(abs)
	if err != nil {
		return err
	}
	var pidl uintptr
	hr, _, _ := parse.Call(uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(&pidl)), 0, 0)
	if hr != 0 || pidl == 0 {
		return fmt.Errorf("SHParseDisplayName failed: 0x%x", hr)
	}
	defer free.Call(pidl)
	hr, _, _ = open.Call(pidl, 0, 0, 0)
	if hr != 0 {
		return fmt.Errorf("SHOpenFolderAndSelectItems failed: 0x%x", hr)
	}
	return nil
}

func explorerSelect(abs string) error {
	args, err := ExplorerSelectArgs(abs)
	if err != nil {
		return err
	}
	cmd := procutil.Command(args[0], args[1:]...)
	return cmd.Start()
}
