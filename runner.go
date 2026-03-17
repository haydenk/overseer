package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Instance represents one running process instance.
type Instance struct {
	Name     string
	Index    int // 0-based instance index within its process type
	Port     int
	Addrs    []string
	ColorIdx int
	cmd      *exec.Cmd
}

// RunConfig holds configuration for the start command.
type RunConfig struct {
	Entries      []Entry
	EnvVars      map[string]string
	Formation    map[string]int
	DefaultCount int
	BasePort     int
	Timeout      int
	NoTimestamp  bool
	ColorOutput  bool
}

// Run spawns all process instances and manages their lifecycle.
func Run(cfg RunConfig) int {
	var instances []*Instance
	processIndex := 0
	maxWidth := len("overseer")

	for _, e := range cfg.Entries {
		count := CountFor(cfg.Formation, cfg.DefaultCount, e.Name)
		for i := 0; i < count; i++ {
			label := e.Name
			if count > 1 {
				label = fmt.Sprintf("%s.%d", e.Name, i+1)
			}
			if len(label) > maxWidth {
				maxWidth = len(label)
			}
		}
		processIndex++
	}
	if len("overseer") > maxWidth {
		maxWidth = len("overseer")
	}

	out := NewWriter(os.Stdout, maxWidth, cfg.NoTimestamp, cfg.ColorOutput)

	processIndex = 0
	colorIdx := 0
	for _, e := range cfg.Entries {
		count := CountFor(cfg.Formation, cfg.DefaultCount, e.Name)
		for i := 0; i < count; i++ {
			port := cfg.BasePort + (processIndex * 100) + i
			label := e.Name
			if count > 1 {
				label = fmt.Sprintf("%s.%d", e.Name, i+1)
			}
			inst := &Instance{
				Name:     label,
				Index:    i,
				Port:     port,
				ColorIdx: colorIdx,
			}

			env := buildEnv(cfg.EnvVars, port, label)
			cmd := buildCmd(e.Command, env)
			inst.cmd = cmd
			instances = append(instances, inst)
		}
		colorIdx++
		processIndex++
	}

	if len(instances) == 0 {
		fmt.Fprintln(os.Stderr, "no processes to run")
		return 1
	}

	// Pre-flight: verify all ports are available before starting anything.
	for _, inst := range instances {
		addrs := resolveAddresses(inst.Port)
		if len(addrs) == 0 {
			fmt.Fprintf(os.Stderr, "error: port %d for %q is already in use\n", inst.Port, inst.Name)
			return 1
		}
		inst.Addrs = addrs
	}

	// exitCode tracks the first non-zero exit (or zero if all succeed).
	var exitCode int32
	var exitOnce sync.Once
	done := make(chan struct{})
	var wg sync.WaitGroup

	// Start all instances.
	for _, inst := range instances {
		pr, pw, err := os.Pipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "pipe error: %v\n", err)
			return 1
		}
		inst.cmd.Stdout = pw
		inst.cmd.Stderr = pw

		if err := inst.cmd.Start(); err != nil {
			out.WriteSystem("overseer", fmt.Sprintf("failed to start %s: %v", inst.Name, err))
			pw.Close()
			pr.Close()
			continue
		}
		pw.Close() // parent doesn't write

		addrInfo := fmt.Sprintf(" (listening on %s)", strings.Join(inst.Addrs, ", "))
		out.WriteSystem("overseer", fmt.Sprintf("started %s with pid %d%s", inst.Name, inst.cmd.Process.Pid, addrInfo))

		wg.Add(1)
		go func(inst *Instance, r io.Reader) {
			defer wg.Done()
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				out.WriteLine(inst.ColorIdx, inst.Name, scanner.Text())
			}
		}(inst, pr)

		wg.Add(1)
		go func(inst *Instance) {
			defer wg.Done()
			err := inst.cmd.Wait()
			code := 0
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					code = ee.ExitCode()
				} else {
					code = 1
				}
			}
			exitOnce.Do(func() {
				atomic.StoreInt32(&exitCode, int32(code))
			})
			out.WriteSystem("overseer", fmt.Sprintf("%s exited with status %d", inst.Name, code))
			select {
			case <-done:
			default:
				close(done)
			}
		}(inst)
	}

	// Signal handling.
	sigCh := make(chan os.Signal, 8)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP,
		syscall.SIGUSR1, syscall.SIGUSR2)

	select {
	case sig := <-sigCh:
		switch sig {
		case syscall.SIGUSR1, syscall.SIGUSR2:
			forwardSignal(instances, sig.(syscall.Signal))
			// Re-arm signal handling and wait again.
		default:
			out.WriteSystem("overseer", fmt.Sprintf("received %s, shutting down", sig))
		}
	case <-done:
		// A process exited; initiate shutdown.
	}

	signal.Stop(sigCh)
	shutdown(instances, out, cfg.Timeout)

	wg.Wait()
	return int(atomic.LoadInt32(&exitCode))
}

func shutdown(instances []*Instance, out *Writer, timeout int) {
	out.WriteSystem("overseer", "sending SIGTERM to all processes")
	for _, inst := range instances {
		if inst.cmd.Process != nil {
			_ = inst.cmd.Process.Signal(syscall.SIGTERM)
		}
	}

	if timeout <= 0 {
		timeout = 5
	}
	deadline := time.After(time.Duration(timeout) * time.Second)

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			out.WriteSystem("overseer", "timeout reached, sending SIGKILL")
			for _, inst := range instances {
				if inst.cmd.Process != nil {
					_ = inst.cmd.Process.Signal(syscall.SIGKILL)
				}
			}
			return
		case <-ticker.C:
			allDone := true
			for _, inst := range instances {
				if inst.cmd.ProcessState == nil {
					allDone = false
					break
				}
			}
			if allDone {
				return
			}
		}
	}
}

func forwardSignal(instances []*Instance, sig syscall.Signal) {
	for _, inst := range instances {
		if inst.cmd.Process != nil {
			_ = syscall.Kill(inst.cmd.Process.Pid, sig)
		}
	}
}

// buildEnv constructs the environment slice for a child process.
func buildEnv(envVars map[string]string, port int, label string) []string {
	merged := make(map[string]string)
	for _, kv := range os.Environ() {
		idx := strings.Index(kv, "=")
		if idx < 0 {
			continue
		}
		merged[kv[:idx]] = kv[idx+1:]
	}
	for k, v := range envVars {
		merged[k] = v
	}
	merged["PORT"] = strconv.Itoa(port)
	merged["PS"] = label

	env := make([]string, 0, len(merged))
	for k, v := range merged {
		env = append(env, k+"="+v)
	}
	return env
}

// buildCmd creates an exec.Cmd for the given shell command string.
func buildCmd(command string, env []string) *exec.Cmd {
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = env
	return cmd
}
