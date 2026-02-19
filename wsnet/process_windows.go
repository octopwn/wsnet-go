//go:build windows
// +build windows

package wsnet

import (
	"os/exec"
	"syscall"
)

// configureProcAttr configures process attributes for Windows systems
func configureProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true, // Hide console window for GUI applications
	}
}
