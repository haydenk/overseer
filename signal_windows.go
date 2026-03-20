//go:build windows

package main

import (
	"os"
	"os/exec"
)

func registerExtraSignals(sigCh chan os.Signal) {}

func isForwardSignal(sig os.Signal) bool { return false }

func doForwardSignal(instances []*Instance, sig os.Signal) {}

func terminateProcess(p *os.Process) {
	_ = p.Signal(os.Interrupt)
}

func killProcess(p *os.Process) {
	_ = p.Kill()
}

func newCmd(command string) *exec.Cmd {
	return exec.Command("cmd", "/C", command)
}
