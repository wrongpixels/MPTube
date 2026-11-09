//go:build windows

package folderpick

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	fosPickFolders     = 0x00000020
	fosForceFileSystem = 0x00000040
	fosNoChangeDir     = 0x00000008
	fosPathMustExist   = 0x00000800
	sigdnFileSysPath   = 0x80058000
	rpcEChangedMode    = syscall.Errno(0x80010106)
	hresultCancelled   = 0x800704C7
	sFalse             = syscall.Errno(1)
)

var (
	clsidFileOpenDialog = windows.GUID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE, Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	iidIFileOpenDialog  = windows.GUID{Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768, Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	iidIShellItem       = windows.GUID{Data1: 0x43826D1E, Data2: 0xE718, Data3: 0x42EE, Data4: [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}

	modOle32                        = windows.NewLazySystemDLL("ole32.dll")
	modShell32                      = windows.NewLazySystemDLL("shell32.dll")
	procCoCreateInstance            = modOle32.NewProc("CoCreateInstance")
	procSHCreateItemFromParsingName = modShell32.NewProc("SHCreateItemFromParsingName")
)

type iFileOpenDialogVtbl struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	Show                uintptr
	SetFileTypes        uintptr
	SetFileTypeIndex    uintptr
	GetFileTypeIndex    uintptr
	Advise              uintptr
	Unadvise            uintptr
	SetOptions          uintptr
	GetOptions          uintptr
	SetDefaultFolder    uintptr
	SetFolder           uintptr
	GetFolder           uintptr
	GetCurrentSelection uintptr
	SetFileName         uintptr
	GetFileName         uintptr
	SetTitle            uintptr
	SetOkButtonLabel    uintptr
	SetFileNameLabel    uintptr
	GetResult           uintptr
	AddPlace            uintptr
	SetDefaultExtension uintptr
	Close               uintptr
	SetClientGuid       uintptr
	ClearClientData     uintptr
	SetFilter           uintptr
	GetResults          uintptr
	GetSelectedItems    uintptr
}

type iFileOpenDialog struct {
	vtbl *iFileOpenDialogVtbl
}

type iShellItemVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	BindToHandler  uintptr
	GetParent      uintptr
	GetDisplayName uintptr
	GetAttributes  uintptr
	Compare        uintptr
}

type iShellItem struct {
	vtbl *iShellItemVtbl
}

func (d *iFileOpenDialog) Release() {
	if d == nil {
		return
	}
	syscall.SyscallN(d.vtbl.Release, uintptr(unsafe.Pointer(d)))
}

func callHR(fn uintptr, args ...uintptr) uintptr {
	r1, _, _ := syscall.SyscallN(fn, args...)
	return r1
}

func (s *iShellItem) Release() {
	if s == nil {
		return
	}
	syscall.SyscallN(s.vtbl.Release, uintptr(unsafe.Pointer(s)))
}

// PickFolder shows the modern Vista+ folder dialog (IFileDialog, FOS_PICKFOLDERS)
// with the navigation sidebar (Quick Access, Desktop, Downloads, etc.).
// owner may be 0. Cancel returns accepted=false and no error.
func PickFolder(owner uintptr, title, initialPath string) (path string, accepted bool, err error) {
	uninit, err := ensureSTA()
	if err != nil {
		return "", false, err
	}
	defer uninit()

	var dlg *iFileOpenDialog
	r1, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidFileOpenDialog)),
		0,
		uintptr(windows.CLSCTX_INPROC_SERVER),
		uintptr(unsafe.Pointer(&iidIFileOpenDialog)),
		uintptr(unsafe.Pointer(&dlg)),
	)
	if hr := uint32(r1); hr != 0 {
		return "", false, fmt.Errorf("could not create the folder picker (0x%08X)", hr)
	}
	if dlg == nil {
		return "", false, errors.New("could not create the folder picker")
	}
	defer dlg.Release()

	var opts uint32
	syscall.SyscallN(dlg.vtbl.GetOptions, uintptr(unsafe.Pointer(dlg)), uintptr(unsafe.Pointer(&opts)))
	opts |= fosPickFolders | fosForceFileSystem | fosPathMustExist | fosNoChangeDir
	if hr := callHR(dlg.vtbl.SetOptions, uintptr(unsafe.Pointer(dlg)), uintptr(opts)); hr != 0 {
		return "", false, fmt.Errorf("could not configure the folder picker (0x%08X)", uint32(hr))
	}

	if title == "" {
		title = "Choose folder to save MP3 files"
	}
	titlePtr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return "", false, err
	}
	_ = callHR(dlg.vtbl.SetTitle, uintptr(unsafe.Pointer(dlg)), uintptr(unsafe.Pointer(titlePtr)))

	if item := shellItemFromPath(initialPath); item != nil {
		// SetFolder prefers the current destination when it exists.
		// If the drive is disconnected, SHCreateItemFromParsingName fails and we leave the default view.
		_ = callHR(dlg.vtbl.SetFolder, uintptr(unsafe.Pointer(dlg)), uintptr(unsafe.Pointer(item)))
		item.Release()
	}

	hr := callHR(dlg.vtbl.Show, uintptr(unsafe.Pointer(dlg)), owner)
	if uint32(hr) == hresultCancelled {
		return "", false, nil
	}
	if hr != 0 {
		return "", false, fmt.Errorf("folder picker failed (0x%08X)", uint32(hr))
	}

	var result *iShellItem
	if hr := callHR(dlg.vtbl.GetResult, uintptr(unsafe.Pointer(dlg)), uintptr(unsafe.Pointer(&result))); hr != 0 {
		return "", false, fmt.Errorf("could not read the selected folder (0x%08X)", uint32(hr))
	}
	if result == nil {
		return "", false, errors.New("could not read the selected folder")
	}
	defer result.Release()

	var namePtr *uint16
	if hr := callHR(result.vtbl.GetDisplayName, uintptr(unsafe.Pointer(result)), uintptr(sigdnFileSysPath), uintptr(unsafe.Pointer(&namePtr))); hr != 0 {
		return "", false, fmt.Errorf("could not read the selected folder path (0x%08X)", uint32(hr))
	}
	if namePtr == nil {
		return "", false, errors.New("could not read the selected folder path")
	}
	defer windows.CoTaskMemFree(unsafe.Pointer(namePtr))
	return windows.UTF16PtrToString(namePtr), true, nil
}

func shellItemFromPath(path string) *iShellItem {
	if path == "" {
		return nil
	}
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil
	}
	var item *iShellItem
	r1, _, _ := procSHCreateItemFromParsingName.Call(
		uintptr(unsafe.Pointer(ptr)),
		0,
		uintptr(unsafe.Pointer(&iidIShellItem)),
		uintptr(unsafe.Pointer(&item)),
	)
	if r1 != 0 || item == nil {
		return nil
	}
	return item
}

func ensureSTA() (func(), error) {
	err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED)
	if err == nil {
		return windows.CoUninitialize, nil
	}
	if errors.Is(err, sFalse) || err == sFalse {
		// Already initialized on this thread (typical for a walk UI thread).
		return func() {}, nil
	}
	if errors.Is(err, rpcEChangedMode) || err == rpcEChangedMode {
		// COM is already initialized with a different model; do not uninitialize it.
		return func() {}, nil
	}
	return func() {}, fmt.Errorf("COM initialize: %w", err)
}
