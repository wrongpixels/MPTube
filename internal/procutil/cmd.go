package procutil

import (
	"context"
	"os/exec"
)

// CommandContext builds an exec.Cmd that never flashes a Windows console.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	HideConsole(cmd)
	return cmd
}

// Command builds an exec.Cmd that never flashes a Windows console.
func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	HideConsole(cmd)
	return cmd
}
