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
	// Signal the entire process group so grandchildren (e.g. php-fpm spawned by sh)
	// also receive SIGTERM. With Setpgid=true the pgid equals the child's pid.
	if err := syscall.Kill(-p.Pid, syscall.SIGTERM); err != nil {
		_ = p.Signal(syscall.SIGTERM)
	}
}

func killProcess(p *os.Process) {
	if err := syscall.Kill(-p.Pid, syscall.SIGKILL); err != nil {
		_ = p.Signal(syscall.SIGKILL)
	}
}

func newCmd(command string) *exec.Cmd {
	// Use "exec" to replace the shell with the actual process so that:
	//   1. The pid overseer tracks IS the real process (correct exit code).
	//   2. SIGTERM is received by the process itself, not the shell (which would
	//      exit with -1 immediately, before the child finishes graceful shutdown).
	// For simple commands (no pipelines) the shell is fully replaced by the process.
	cmd := exec.Command("sh", "-c", "exec "+command)
	// Put each child in its own process group so signals reach the whole tree.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}
