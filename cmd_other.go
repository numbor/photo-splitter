//go:build !windows

package main

import "os/exec"

func hideExternalConsoleWindow(cmd *exec.Cmd) {
	_ = cmd
}
