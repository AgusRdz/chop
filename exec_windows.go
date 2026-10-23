//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// newCommand builds the wrapped command, overriding Go's command-line quoting
// when an argument is not plain (see quoteWindowsArg).
func newCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	if line := windowsCmdLine(name, args); line != "" {
		cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: line}
	}
	return cmd
}
