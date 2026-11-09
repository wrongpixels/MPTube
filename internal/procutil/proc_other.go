//go:build !windows

package procutil

import "os/exec"

func Start(cmd *exec.Cmd) (cleanup func(), err error) {
	if err := cmd.Start(); err != nil {
		return func() {}, err
	}
	return func() {}, nil
}

func HideConsole(cmd *exec.Cmd) {}
