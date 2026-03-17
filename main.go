package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const usageText = `overseer — Procfile-based process manager

Usage:
  overseer check  [-f Procfile]
  overseer start  [PROCESS] [-f Procfile] [-e .env] [-m formation] [-p port] [-t timeout] [-c] [--no-timestamp]
  overseer run    COMMAND   [-f Procfile] [-e .env]
  overseer help

Commands:
  check   Validate a Procfile and list detected process types.
  start   Start all (or selected) processes defined in the Procfile.
  run     Run a single command with the environment loaded from the Procfile/env file.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "check":
		os.Exit(cmdCheck(os.Args[2:]))
	case "start":
		os.Exit(cmdStart(os.Args[2:]))
	case "run":
		os.Exit(cmdRun(os.Args[2:]))
	case "help", "-h", "--help":
		fmt.Print(usageText)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n%s", os.Args[1], usageText)
		os.Exit(1)
	}
}

// cmdCheck validates and lists the Procfile.
func cmdCheck(args []string) int {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	procfilePath := fs.String("f", "Procfile", "Procfile path")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: overseer check [-f Procfile]")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	entries, err := ParseProcfile(*procfilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	fmt.Printf("valid procfile detected (%s)\n", strings.Join(names, ", "))
	return 0
}

// cmdStart is the main process manager.
func cmdStart(args []string) int {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	procfilePath := fs.String("f", "Procfile", "Procfile path")
	envFile := fs.String("e", ".env", ".env file path")
	formation := fs.String("m", "", "process formation, e.g. all=1,web=2")
	portFlag := fs.String("p", "", "base port (overrides $PORT, default 3000)")
	timeout := fs.Int("t", 5, "graceful shutdown timeout in seconds")
	colorOutput := fs.Bool("c", true, "enable colored output")
	noTimestamp := fs.Bool("no-timestamp", false, "disable timestamps in output")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: overseer start [PROCESS] [-f Procfile] [-e .env] [-m formation] [-p port] [-t timeout] [-c] [--no-timestamp]")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	// Optional positional: a single process name → formation shorthand.
	processFilter := fs.Arg(0)
	formationSpec := *formation
	if processFilter != "" && formationSpec == "" {
		formationSpec = processFilter + "=1"
	}

	entries, err := ParseProcfile(*procfilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	envVars := map[string]string{}
	if *envFile != "" {
		if ev, err := ParseEnvFile(*envFile); err == nil {
			envVars = ev
		}
		// Silently ignore missing .env file.
	}

	formationMap, defaultCount, err := ParseFormation(formationSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing formation: %v\n", err)
		return 1
	}

	basePort := 3000
	if p := os.Getenv("PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			basePort = n
		}
	}
	if *portFlag != "" {
		if n, err := strconv.Atoi(*portFlag); err == nil {
			basePort = n
		}
	}

	// Filter entries if a specific process was named.
	filtered := entries
	if processFilter != "" {
		filtered = nil
		for _, e := range entries {
			if e.Name == processFilter {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "process %q not found in Procfile\n", processFilter)
			return 1
		}
	}

	return Run(RunConfig{
		Entries:      filtered,
		EnvVars:      envVars,
		Formation:    formationMap,
		DefaultCount: defaultCount,
		BasePort:     basePort,
		Timeout:      *timeout,
		NoTimestamp:  *noTimestamp,
		ColorOutput:  *colorOutput,
	})
}

// cmdRun executes a single command (or Procfile entry) with the loaded environment.
func cmdRun(args []string) int {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	procfilePath := fs.String("f", "Procfile", "Procfile path")
	envFile := fs.String("e", ".env", ".env file path")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: overseer run COMMAND [-f Procfile] [-e .env]")
		fs.PrintDefaults()
	}
	_ = fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: no command specified")
		fs.Usage()
		return 1
	}

	envVars := map[string]string{}
	if *envFile != "" {
		if ev, err := ParseEnvFile(*envFile); err == nil {
			envVars = ev
		}
	}

	// Determine the base port.
	basePort := 3000
	if p := os.Getenv("PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			basePort = n
		}
	}

	// Look up a Procfile entry by name; fall back to running the literal args.
	commandStr := ""
	name := fs.Arg(0)

	if entries, err := ParseProcfile(*procfilePath); err == nil {
		for _, e := range entries {
			if e.Name == name {
				commandStr = e.Command
				break
			}
		}
	}
	if commandStr == "" {
		commandStr = strings.Join(fs.Args(), " ")
	}

	env := buildEnv(envVars, basePort, name)

	// Resolve the actual binary for the first word, to support IPv4+IPv6 aware net tools.
	// For "run" we just exec a shell command, so bindings respect the OS stack.
	cmd := exec.Command("sh", "-c", commandStr)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Forward SIGINT to the child.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	signal.Stop(sigCh)
	return 0
}

// resolveAddresses returns both IPv4 and IPv6 listen addresses for a port,
// using a dual-stack listener where possible (useful for diagnostics/info).
func resolveAddresses(port int) []string {
	portStr := strconv.Itoa(port)
	var addrs []string

	// Try IPv6 dual-stack first (":port" listens on both :: and 0.0.0.0 on most OSes).
	ln, err := net.Listen("tcp", ":"+portStr)
	if err == nil {
		ln.Close()
		addrs = append(addrs, "[::]"+":"+portStr, "0.0.0.0:"+portStr)
		return addrs
	}

	// Fall back to explicit IPv4.
	ln4, err4 := net.Listen("tcp4", "0.0.0.0:"+portStr)
	if err4 == nil {
		ln4.Close()
		addrs = append(addrs, "0.0.0.0:"+portStr)
	}

	// And explicit IPv6.
	ln6, err6 := net.Listen("tcp6", "[::]:"+portStr)
	if err6 == nil {
		ln6.Close()
		addrs = append(addrs, "[::]:"+portStr)
	}

	return addrs
}
