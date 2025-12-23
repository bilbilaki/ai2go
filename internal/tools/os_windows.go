//go:build windows

package tools

import (
	"context"
	"os/exec"
	"syscall"
)

func prepareCommand(ctx context.Context, command string) *exec.Cmd {
	// Use cmd.exe on Windows. 
	// Note: If the AI wants PowerShell, it can explicitly run "powershell -Command ..."
	cmd := exec.CommandContext(ctx, "cmd", "/C", command)
	
	// CREATE_NEW_PROCESS_GROUP (0x200) allows clean termination signals on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	
	return cmd
}