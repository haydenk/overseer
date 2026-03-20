//go:build !windows

package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func registerExtraSignals(sigCh chan os.Signal) {
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGUSR1, syscall.SIGUSR2)
}

func isForwardSignal(sig os.Signal) bool {
	return sig == syscall.SIGUSR1 || sig == syscall.SIGUSR2
}

func doForwardSignal(instances []*Instance, sig os.Signal) {
	for _, inst := range instances {
		if inst.cmd.Process != nil {
			_ = syscall.Kill(inst.cmd.Process.Pid, sig.(syscall.Signal))
		}
	}
}

func terminateProcess(p *os.Process) {
	_ = p.Signal(syscall.SIGTERM)
}

func killProcess(p *os.Process) {
	_ = p.Signal(syscall.SIGKILL)
}

func newCmd(command string) *exec.Cmd {
	return exec.Command("sh", "-c", command)
}
