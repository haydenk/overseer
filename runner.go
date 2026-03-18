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
	instances := buildInstances(cfg)
	if len(instances) == 0 {
		fmt.Fprintln(os.Stderr, "no processes to run")
		return 1
	}

	if err := checkPorts(instances); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	maxWidth := computeMaxWidth(cfg.Entries, cfg.Formation, cfg.DefaultCount)
	out := NewWriter(os.Stdout, maxWidth, cfg.NoTimestamp, cfg.ColorOutput)

	var exitCode int32
	var exitOnce sync.Once
	done := make(chan struct{})
	var wg sync.WaitGroup

	for _, inst := range instances {
		if err := spawnInstance(inst, out, &wg, &exitCode, &exitOnce, done); err != nil {
			return 1
		}
	}

	sigCh := make(chan os.Signal, 8)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP,
		syscall.SIGUSR1, syscall.SIGUSR2)

	select {
	case sig := <-sigCh:
		switch sig {
		case syscall.SIGUSR1, syscall.SIGUSR2:
			forwardSignal(instances, sig.(syscall.Signal))
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

// computeMaxWidth returns the longest label width across all instances.
func computeMaxWidth(entries []Entry, formation map[string]int, defaultCount int) int {
	maxWidth := len("overseer")
	for _, e := range entries {
		count := CountFor(formation, defaultCount, e.Name)
		for i := 0; i < count; i++ {
			label := e.Name
			if count > 1 {
				label = fmt.Sprintf("%s.%d", e.Name, i+1)
			}
			if len(label) > maxWidth {
				maxWidth = len(label)
			}
		}
	}
	return maxWidth
}

// buildInstances creates all Instance values from the run configuration.
func buildInstances(cfg RunConfig) []*Instance {
	var instances []*Instance
	processIndex := 0
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
			inst.cmd = buildCmd(e.Command, env)
			instances = append(instances, inst)
		}
		colorIdx++
		processIndex++
	}
	return instances
}

// checkPorts verifies all instance ports are available and populates inst.Addrs.
func checkPorts(instances []*Instance) error {
	for _, inst := range instances {
		addrs := resolveAddresses(inst.Port)
		if len(addrs) == 0 {
			return fmt.Errorf("error: port %d for %q is already in use", inst.Port, inst.Name)
		}
		inst.Addrs = addrs
	}
	return nil
}

// spawnInstance starts a single process and launches its I/O and wait goroutines.
// A non-nil error indicates a fatal setup failure; a failed Start is logged but not fatal.
func spawnInstance(inst *Instance, out *Writer, wg *sync.WaitGroup, exitCode *int32, exitOnce *sync.Once, done chan struct{}) error {
	pr, pw, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pipe error: %v\n", err)
		return err
	}
	inst.cmd.Stdout = pw
	inst.cmd.Stderr = pw

	if err := inst.cmd.Start(); err != nil {
		out.WriteSystem("overseer", fmt.Sprintf("failed to start %s: %v", inst.Name, err))
		pw.Close()
		pr.Close()
		return nil
	}
	pw.Close() // parent doesn't write

	addrInfo := fmt.Sprintf(" (listening on %s)", strings.Join(inst.Addrs, ", "))
	out.WriteSystem("overseer", fmt.Sprintf("started %s with pid %d%s", inst.Name, inst.cmd.Process.Pid, addrInfo))

	wg.Add(1)
	go streamOutput(inst, pr, out, wg)

	wg.Add(1)
	go watchExit(inst, out, wg, exitCode, exitOnce, done)
	return nil
}

// streamOutput copies scanner lines from r to the writer until EOF.
func streamOutput(inst *Instance, r io.Reader, out *Writer, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		out.WriteLine(inst.ColorIdx, inst.Name, scanner.Text())
	}
}

// watchExit waits for inst to exit, records its code, and signals done.
func watchExit(inst *Instance, out *Writer, wg *sync.WaitGroup, exitCode *int32, exitOnce *sync.Once, done chan struct{}) {
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
		atomic.StoreInt32(exitCode, int32(code))
	})
	out.WriteSystem("overseer", fmt.Sprintf("%s exited with status %d", inst.Name, code))
	select {
	case <-done:
	default:
		close(done)
	}
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
