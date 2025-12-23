//go:build !windows

package tools

import (
	"context"
	"os/exec"
	"syscall"
)

func prepareCommand(ctx context.Context, command string) *exec.Cmd {
	// Use bash on Mac/Linux
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	
	// Create a process group so we can kill the whole tree if needed
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	
	return cmd
}