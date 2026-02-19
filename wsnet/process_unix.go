//go:build !windows
// +build !windows

package wsnet

import (
	"os/exec"
	"syscall"
)

// configureProcAttr configures process attributes for Unix-like systems
func configureProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session (equivalent to setsid)
	}
}
