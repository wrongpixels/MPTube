//go:build windows

package procutil

import (
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const createNoWindow = 0x08000000

// HideConsole configures a process so Windows does not flash a console window.
// This covers yt-dlp, and children such as FFmpeg and Deno inherit no console
// from a CREATE_NO_WINDOW parent.
func HideConsole(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}

// Start launches cmd and assigns it to a Windows job object so cancel
// terminates yt-dlp and any FFmpeg child it started.
func Start(cmd *exec.Cmd) (cleanup func(), err error) {
	HideConsole(cmd)
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		if startErr := cmd.Start(); startErr != nil {
			return func() {}, startErr
		}
		return func() {}, nil
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		windows.CloseHandle(job)
		if startErr := cmd.Start(); startErr != nil {
			return func() {}, startErr
		}
		return func() {}, nil
	}
	if err := cmd.Start(); err != nil {
		windows.CloseHandle(job)
		return func() {}, err
	}
	const rights = windows.PROCESS_SET_QUOTA | windows.PROCESS_TERMINATE | windows.PROCESS_QUERY_LIMITED_INFORMATION
	ph, openErr := windows.OpenProcess(rights, false, uint32(cmd.Process.Pid))
	if openErr == nil {
		_ = windows.AssignProcessToJobObject(job, ph)
		windows.CloseHandle(ph)
	}
	return func() {
		windows.CloseHandle(job)
	}, nil
}
